package render3d

import (
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// タイル世界の高さ。タイル1個分を 1.0 として測る
const (
	// WallHeight は壁の高さ。壁は箱として描かれるので、壁タイルを指す印は天面へ置く
	WallHeight = 1.0
	// BillboardHeight はエンティティの立て板の高さ。頭の上に出す表示の基準になる
	BillboardHeight = 1.0
	// cameraTargetHeight はカメラが見つめる高さ。床と壁の中間を見ることで立て板が画面中央に収まる
	cameraTargetHeight = 0.4
)

// カメラの投影パラメータ。視野角と可視距離の範囲を決める
const (
	fovDeg = 52.0
	near   = 0.1
	far    = 200.0
)

// Projector はタイル座標をスクリーン座標へ写す。
// カメラと画面寸法から組んだ view-projection 行列を保持し、1回の描画のあいだ使い回す。
type Projector struct {
	vp     mat
	view   mat
	eye    Vec
	target Vec
	sw, sh int
}

// New は視点からの投影を組む。
//
// orbit はカメラの向きと距離、center は注視するタイルで、通常はプレイヤーの立つタイルを渡す。
func New(orbit gc.Camera, center consts.Coord[consts.Tile], sw, sh int) Projector {
	target := Vec{float64(center.X) + 0.5, cameraTargetHeight, float64(center.Y) + 0.5}
	yaw, pitch := orbit.Yaw(), orbit.Pitch
	// カメラはプレイヤーの南側から北を見下ろす。画面の上を北に合わせ、北上のミニマップと向きをそろえる
	dir := Vec{math.Cos(pitch) * math.Sin(yaw), math.Sin(pitch), math.Cos(pitch) * math.Cos(yaw)}
	eye := Add(target, Scale(dir, orbit.Dist))
	view := lookAt(eye, target, Vec{0, 1, 0})

	return Projector{
		vp:     mul(perspective(fovDeg, float64(sw)/float64(sh), near, far), view),
		view:   view,
		eye:    eye,
		target: target,
		sw:     sw,
		sh:     sh,
	}
}

// Right はカメラから見た画面右方向の単位ベクトル。立て板をカメラへ正対させるのに使う
func (p Projector) Right() Vec {
	return norm(cross(norm(sub(p.target, p.eye)), Vec{0, 1, 0}))
}

// Depth は点のカメラ空間での奥行きを返す。奥ほど小さい。画家アルゴリズムの並べ替えに使う
func (p Projector) Depth(v Vec) float64 {
	_, _, z, _ := apply(p.view, v)
	return z
}

// Point は3D空間の点をスクリーン座標へ写す。カメラ後方なら ok=false
func (p Projector) Point(v Vec) (consts.Coord[consts.ScreenPixel], bool) {
	cx, cy, _, cw := apply(p.vp, v)
	if cw <= 0.001 {
		return consts.Coord[consts.ScreenPixel]{}, false
	}
	return consts.Coord[consts.ScreenPixel]{
		X: consts.ScreenPixel((cx/cw*0.5 + 0.5) * float64(p.sw)),
		Y: consts.ScreenPixel((1 - (cy/cw*0.5 + 0.5)) * float64(p.sh)),
	}, true
}

// TileCenter はタイル中心をスクリーン座標へ写す。
// height はタイル上面の高さで、床なら 0、壁なら WallHeight を渡す。カメラ後方なら ok=false。
func (p Projector) TileCenter(c consts.Coord[consts.Tile], height float64) (consts.Coord[consts.ScreenPixel], bool) {
	return p.Point(Vec{float64(c.X) + 0.5, height, float64(c.Y) + 0.5})
}

// TileCorners はタイルの四隅を高さ height でスクリーン座標へ写す。並びは北西・北東・南東・南西。
// 透視投影ではタイルが台形になるので、タイルを囲う枠はこの4点を結んで描く。
// 4隅すべて投影できたときだけ ok=true を返す。
func (p Projector) TileCorners(c consts.Coord[consts.Tile], height float64) ([4]consts.Coord[consts.ScreenPixel], bool) {
	fx, fz := float64(c.X), float64(c.Y)
	src := [4]Vec{
		{fx, height, fz},
		{fx + 1, height, fz},
		{fx + 1, height, fz + 1},
		{fx, height, fz + 1},
	}
	var out [4]consts.Coord[consts.ScreenPixel]
	for i, v := range src {
		sp, ok := p.Point(v)
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
	return p.Point(Vec{float64(c.X) + 0.5, BillboardHeight, float64(c.Y) + 0.5})
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
