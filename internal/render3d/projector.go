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

// maxStandHeight は TileOnScreen が映りうる範囲の矩形で落とす天面の高さの上限
const maxStandHeight = max(WallHeight, BillboardHeight)

// minDepth はカメラの前とみなす奥行きの下限。Point はこれより手前の点を投影せず、映りうる範囲の矩形も
// この奥行きから視錐台を張る
const minDepth = 0.001

// visibleReach は映りうる範囲の矩形を求めるときの視錐台の奥行き。Point は遠平面で切らず far より遠い点も
// 画面へ写すので、矩形も far で切らずに十分遠くまで伸ばす
const visibleReach = 1e4

// Projector はタイル座標をスクリーン座標へ写す。
// カメラと画面寸法から組んだ view-projection 行列を保持し、1回の描画のあいだ使い回す。
type Projector struct {
	vp     mat
	view   mat
	eye    Vec
	target Vec
	sw, sh int
	// visMin と visMax は高さ 0 から maxStandHeight までの層のうち、視錐台に入りうるタイルを包む矩形。両端を含む
	visMin, visMax consts.Coord[consts.Tile]
}

// NewProjector は視点からの投影を組む。
//
// orbit はカメラの向きと距離、center は注視するタイルで、通常はプレイヤーの立つタイルを渡す。
func NewProjector(orbit gc.Camera, center consts.Coord[consts.Tile], sw, sh int) Projector {
	target := Vec{float64(center.X) + 0.5, cameraTargetHeight, float64(center.Y) + 0.5}
	yaw, pitch := orbit.Yaw(), orbit.Pitch
	// カメラはプレイヤーの南側から北を見下ろす。画面の上を北に合わせ、北上のミニマップと向きをそろえる
	dir := Vec{math.Cos(pitch) * math.Sin(yaw), math.Sin(pitch), math.Cos(pitch) * math.Cos(yaw)}
	eye := Add(target, Scale(dir, orbit.Dist))
	view := lookAt(eye, target, Vec{0, 1, 0})

	visMin, visMax := visibleTileRect(eye, target, float64(sw)/float64(sh))
	return Projector{
		vp:     mul(perspective(fovDeg, float64(sw)/float64(sh), near, far), view),
		view:   view,
		eye:    eye,
		target: target,
		sw:     sw,
		sh:     sh,
		visMin: visMin,
		visMax: visMax,
	}
}

// visibleTileRect は視錐台と高さ 0 から maxStandHeight までの層の重なりを包むタイルの矩形を返す。
// 重なりは凸なので、その頂点は層の中にある視錐台の隅か、視錐台の辺と層の面の交点のどちらかになる。
// それらの点の範囲を外へ1タイル広げて包む。重ならなければ min が max を超える空の矩形を返す
func visibleTileRect(eye, target Vec, aspect float64) (consts.Coord[consts.Tile], consts.Coord[consts.Tile]) {
	f := norm(sub(target, eye))
	s := norm(cross(f, Vec{0, 1, 0}))
	u := cross(s, f)
	tanHalf := math.Tan(fovDeg * math.Pi / 360)
	var corners [8]Vec
	for i, d := range [2]float64{minDepth, visibleReach} {
		c := Add(eye, Scale(f, d))
		h, w := d*tanHalf, d*tanHalf*aspect
		corners[i*4+0] = Add(c, Add(Scale(s, -w), Scale(u, h)))
		corners[i*4+1] = Add(c, Add(Scale(s, w), Scale(u, h)))
		corners[i*4+2] = Add(c, Add(Scale(s, w), Scale(u, -h)))
		corners[i*4+3] = Add(c, Add(Scale(s, -w), Scale(u, -h)))
	}
	var edges [12][2]int
	for i := range 4 {
		edges[i] = [2]int{i, (i + 1) % 4}
		edges[i+4] = [2]int{i + 4, (i+1)%4 + 4}
		edges[i+8] = [2]int{i, i + 4}
	}

	minX, minZ := math.Inf(1), math.Inf(1)
	maxX, maxZ := math.Inf(-1), math.Inf(-1)
	include := func(v Vec) {
		minX, maxX = min(minX, v.X), max(maxX, v.X)
		minZ, maxZ = min(minZ, v.Z), max(maxZ, v.Z)
	}
	for _, v := range corners {
		if v.Y >= 0 && v.Y <= maxStandHeight {
			include(v)
		}
	}
	for _, e := range edges {
		a, b := corners[e[0]], corners[e[1]]
		for _, y := range [2]float64{0, maxStandHeight} {
			if (a.Y-y)*(b.Y-y) >= 0 {
				continue
			}
			include(Add(a, Scale(sub(b, a), (y-a.Y)/(b.Y-a.Y))))
		}
	}
	if minX > maxX {
		return consts.Coord[consts.Tile]{X: 1, Y: 1}, consts.Coord[consts.Tile]{}
	}
	return consts.Coord[consts.Tile]{X: consts.Tile(math.Floor(minX)) - 1, Y: consts.Tile(math.Floor(minZ)) - 1},
		consts.Coord[consts.Tile]{X: consts.Tile(math.Floor(maxX)) + 1, Y: consts.Tile(math.Floor(maxZ)) + 1}
}

// FacesCamera は外から見て時計回りに張った面 q が、カメラに表を向けているかを返す
func (p Projector) FacesCamera(q [4]Vec) bool {
	return dot(cross(sub(q[1], q[0]), sub(q[2], q[0])), sub(p.eye, q[0])) < 0
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
	if cw <= minDepth {
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

// TileOnScreen はタイル c が画面に掛かるかを返す。先に映りうる範囲の矩形の外を整数の比較で落とし、
// 中のタイルだけ8隅を投影して確かめる。topHeight が maxStandHeight を超えるときは矩形で落とさない
func (p Projector) TileOnScreen(c consts.Coord[consts.Tile], topHeight float64) bool {
	if topHeight <= maxStandHeight && !p.inVisibleRect(c) {
		return false
	}
	return p.boxOnScreen(c, topHeight)
}

// inVisibleRect はタイル c が映りうる範囲の矩形に入るかを返す
func (p Projector) inVisibleRect(c consts.Coord[consts.Tile]) bool {
	return c.X >= p.visMin.X && c.X <= p.visMax.X && c.Y >= p.visMin.Y && c.Y <= p.visMax.Y
}

// boxOnScreen は視錐台カリングの判定。基底 y=0 と天面 y=topHeight の
// 8隅を投影し、そのスクリーン bbox が画面矩形と重なるかを見る。壁やビルボードは上へ伸びるので、
// 天面の高さ topHeight を渡して縦の広がりも含める。
//
// 近平面をまたぐタイルは一部の隅がカメラ後方で投影が破綻するため、bbox を信じず保守的に描く。
// 全隅がカメラ後方のときだけ確実に落とす。bbox はタイルの実クアッドを必ず包むので、映るタイルを
// 誤って落とすことはなく、映らないタイルを稀に残すだけ。ゆえに描画結果は変わらない。
func (p Projector) boxOnScreen(c consts.Coord[consts.Tile], topHeight float64) bool {
	fx, fz := float64(c.X), float64(c.Y)
	corners := [8]Vec{
		{fx, 0, fz}, {fx + 1, 0, fz}, {fx + 1, 0, fz + 1}, {fx, 0, fz + 1},
		{fx, topHeight, fz}, {fx + 1, topHeight, fz}, {fx + 1, topHeight, fz + 1}, {fx, topHeight, fz + 1},
	}
	behind := 0
	front := false
	var minX, minY, maxX, maxY float64
	for _, v := range corners {
		sp, ok := p.Point(v)
		if !ok {
			behind++
			continue
		}
		x, y := float64(sp.X), float64(sp.Y)
		if !front {
			minX, maxX, minY, maxY = x, x, y, y
			front = true
			continue
		}
		minX, maxX = min(minX, x), max(maxX, x)
		minY, maxY = min(minY, y), max(maxY, y)
	}
	if !front {
		return false // 全隅がカメラ後方。確実に映らない
	}
	if behind > 0 {
		return true // 近平面をまたぐ。bbox が壊れるので保守的に描く
	}
	sw, sh := float64(p.sw), float64(p.sh)
	return maxX >= 0 && minX <= sw && maxY >= 0 && minY <= sh
}

// BillboardTop はタイルに立つ立て板の頭をスクリーン座標へ写す。
// エンティティの上へ出すダメージテキストやHP表示の基準点にする。
func (p Projector) BillboardTop(c consts.Coord[consts.Tile]) (consts.Coord[consts.ScreenPixel], bool) {
	return p.Point(Vec{float64(c.X) + 0.5, BillboardHeight, float64(c.Y) + 0.5})
}
