package systems

import (
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// 3Dカメラの投影パラメータ。視野角と可視距離の範囲を決める
const (
	projectionFOVDeg = 52.0
	projectionNear   = 0.1
	projectionFar    = 200.0
)

// Projector はタイル座標をスクリーン座標へ写す。
// 3Dカメラと画面寸法から組んだ view-projection 行列を保持し、1回の Draw のあいだ使い回す。
//
// 世界を描く Render3DSystem と、その上に重ねるカーソル・エフェクト・HUD が、
// どちらもこの型を通ることで投影先が一致する。片方だけ直して片方が取り残される形を作らない。
type Projector struct {
	vp     r3mat
	view   r3mat
	eye    r3vec
	target r3vec
	sw, sh int
}

// NewProjector は world のカメラと画面寸法から投影を組む。
//
// 画面寸法は描画先の実寸を渡す。世界レイヤと重ねるレイヤが同じ画像へ描く限り、
// 双方が同じ行列を得る。
//
// カメラが無い、あるいは値が可動域の外なら既定の視点で組む。ここで投影を諦めると
// カーソルが消えてしまい、位置がずれるより分かりにくい壊れ方になる。
func NewProjector(world w.World, sw, sh int) Projector {
	orbit := gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}
	if camera := getCamera(world); camera != nil {
		orbit = *camera
	}
	orbit.NormalizeOrbit()

	pcx, pcz := playerTileCenter(world)
	target := r3vec{pcx + 0.5, cameraTargetHeight, pcz + 0.5}
	yaw, pitch := orbit.Yaw(), orbit.Pitch
	// カメラはプレイヤーの南側から北を見下ろす。画面の上を北に合わせ、北上のミニマップと向きをそろえる
	dir := r3vec{math.Cos(pitch) * math.Sin(yaw), math.Sin(pitch), math.Cos(pitch) * math.Cos(yaw)}
	eye := r3add(target, r3scale(dir, orbit.Dist))
	view := r3lookAt(eye, target, r3vec{0, 1, 0})

	return Projector{
		vp:     r3mul(r3perspective(projectionFOVDeg, float64(sw)/float64(sh), projectionNear, projectionFar), view),
		view:   view,
		eye:    eye,
		target: target,
		sw:     sw,
		sh:     sh,
	}
}

// cameraTargetHeight はカメラが見つめる高さ。床と壁の中間あたりを見ることで、
// プレイヤーの立て板が画面中央に収まる
const cameraTargetHeight = 0.4

// playerTileCenter はプレイヤーのタイル座標を返す。いなければ既定値を返す
func playerTileCenter(world w.World) (float64, float64) {
	if pe, err := query.GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(pe) {
		g := world.Components.GridElement.Get(pe)
		return float64(g.X), float64(g.Y)
	}
	return 25, 25
}

// right はカメラから見た画面右方向の単位ベクトル。立て板をカメラへ正対させるのに使う
func (p Projector) right() r3vec {
	return r3norm(r3cross(r3norm(r3sub(p.target, p.eye)), r3vec{0, 1, 0}))
}

// point は3D空間の点をスクリーン座標へ写す。カメラ後方なら ok=false
func (p Projector) point(v r3vec) (consts.Coord[consts.ScreenPixel], bool) {
	x, y, ok := projectToScreen(p.vp, v, p.sw, p.sh)
	if !ok {
		return consts.Coord[consts.ScreenPixel]{}, false
	}
	return consts.Coord[consts.ScreenPixel]{X: consts.ScreenPixel(x), Y: consts.ScreenPixel(y)}, true
}

// TileCenter はタイル中心をスクリーン座標へ写す。
// height はタイル上面の高さで、床なら 0、壁なら WallHeight を渡す。カメラ後方なら ok=false。
func (p Projector) TileCenter(c consts.Coord[consts.Tile], height float64) (consts.Coord[consts.ScreenPixel], bool) {
	return p.point(r3vec{float64(c.X) + 0.5, height, float64(c.Y) + 0.5})
}

// TileCorners はタイルの四隅を高さ height でスクリーン座標へ写す。並びは北西・北東・南東・南西。
// 透視投影ではタイルが台形になるので、タイルを囲う枠はこの4点を結んで描く。
// 4隅すべて投影できたときだけ ok=true を返す。
func (p Projector) TileCorners(c consts.Coord[consts.Tile], height float64) ([4]consts.Coord[consts.ScreenPixel], bool) {
	fx, fz := float64(c.X), float64(c.Y)
	src := [4]r3vec{
		{fx, height, fz},
		{fx + 1, height, fz},
		{fx + 1, height, fz + 1},
		{fx, height, fz + 1},
	}
	var out [4]consts.Coord[consts.ScreenPixel]
	for i, v := range src {
		sp, ok := p.point(v)
		if !ok {
			return out, false
		}
		out[i] = sp
	}
	return out, true
}

// BillboardTop はタイルに立つ立て板の頭をスクリーン座標へ写す。
// エンティティの上へ出すダメージテキストやHP表示の基準点にする。
func (p Projector) BillboardTop(c consts.Coord[consts.Tile]) (consts.Coord[consts.ScreenPixel], bool) {
	return p.point(r3vec{float64(c.X) + 0.5, BillboardHeight, float64(c.Y) + 0.5})
}

// BillboardScale は立て板1枚分の高さが画面上で何ピクセルになるかを返す。
// 立て板と同じ大きさで重ねたいスプライトの拡大率を決めるのに使う。投影できなければ ok=false。
func (p Projector) BillboardScale(c consts.Coord[consts.Tile]) (float64, bool) {
	top, okTop := p.BillboardTop(c)
	base, okBase := p.TileCenter(c, 0)
	if !okTop || !okBase {
		return 0, false
	}
	return math.Abs(float64(base.Y - top.Y)), true
}

// TileTopHeight はタイル上面の高さを返す。壁は高さのある箱として描かれるので天面、床は地面になる。
// タイルを指すカーソルはこの高さへ描く。壁タイルを地面の高さで描くと箱に埋もれて見えなくなる。
func TileTopHeight(world w.World, c consts.Coord[consts.Tile]) float64 {
	for _, e := range query.GetEntitiesAt(world, c.X, c.Y) {
		if world.Components.Tile.Has(e) && world.Components.BlockPass.Has(e) {
			return WallHeight
		}
	}
	return 0
}
