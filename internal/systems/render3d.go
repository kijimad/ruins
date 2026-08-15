package systems

import (
	"image/color"
	"math"
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// Render3DSystem は壁と床をローポリの3Dで描く実験的レンダラである。
// 既存の RenderSpriteSystem と同じ ECS ワールドを読み、床タイルを水平クアッド、
// 壁タイルを箱、それ以外のエンティティをビルボードとして透視投影する。
// テクスチャは既存スプライトシートをそのまま流用する。Ebiten は深度バッファを持たないため、
// 各クアッドをカメラ空間の奥行きで並べ替える画家アルゴリズムで隠面を解く。
type Render3DSystem struct {
	// Yaw/Pitch/Dist はプレイヤーを中心に回すオービットカメラ。マウスドラッグで動かす。
	Yaw, Pitch, Dist float64
	// UseFOV は視界データに従い、隠れたタイルを描かず記憶タイルを減光する。
	// 本番のダンジョンでは true、部屋全体を見せたいデモでは false にする
	UseFOV bool
}

// NewRender3DSystem は既定のオービットカメラで初期化する。
func NewRender3DSystem() *Render3DSystem {
	return &Render3DSystem{Yaw: 0, Pitch: 0.62, Dist: 13, UseFOV: true}
}

// String は w.Renderer を満たす。
func (sys *Render3DSystem) String() string { return "Render3DSystem" }

// --- 3D数学。行優先4x4、点は列ベクトルとして M*v で変換する ---

type r3vec struct{ x, y, z float64 }

func r3sub(a, b r3vec) r3vec           { return r3vec{a.x - b.x, a.y - b.y, a.z - b.z} }
func r3add(a, b r3vec) r3vec           { return r3vec{a.x + b.x, a.y + b.y, a.z + b.z} }
func r3scale(a r3vec, s float64) r3vec { return r3vec{a.x * s, a.y * s, a.z * s} }
func r3dot(a, b r3vec) float64         { return a.x*b.x + a.y*b.y + a.z*b.z }
func r3cross(a, b r3vec) r3vec {
	return r3vec{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}
func r3norm(a r3vec) r3vec {
	l := math.Sqrt(r3dot(a, a))
	if l == 0 {
		return a
	}
	return r3scale(a, 1/l)
}

type r3mat [16]float64

func r3mul(a, b r3mat) r3mat {
	var c r3mat
	for i := range 4 {
		for j := range 4 {
			s := 0.0
			for k := range 4 {
				s += a[i*4+k] * b[k*4+j]
			}
			c[i*4+j] = s
		}
	}
	return c
}

func r3apply(m r3mat, p r3vec) (x, y, z, wc float64) {
	x = m[0]*p.x + m[1]*p.y + m[2]*p.z + m[3]
	y = m[4]*p.x + m[5]*p.y + m[6]*p.z + m[7]
	z = m[8]*p.x + m[9]*p.y + m[10]*p.z + m[11]
	wc = m[12]*p.x + m[13]*p.y + m[14]*p.z + m[15]
	return
}

func r3perspective(fovyDeg, aspect, near, far float64) r3mat {
	f := 1.0 / math.Tan(fovyDeg*math.Pi/180/2)
	return r3mat{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), (2 * far * near) / (near - far),
		0, 0, -1, 0,
	}
}

func r3lookAt(eye, center, up r3vec) r3mat {
	f := r3norm(r3sub(center, eye))
	s := r3norm(r3cross(f, up))
	u := r3cross(s, f)
	return r3mat{
		s.x, s.y, s.z, -r3dot(s, eye),
		u.x, u.y, u.z, -r3dot(u, eye),
		-f.x, -f.y, -f.z, r3dot(f, eye),
		0, 0, 0, 1,
	}
}

// --- クアッド ---

type r3quad struct {
	p     [4]r3vec
	uv    [4][2]float64
	atlas *ebiten.Image
	col   [3]float64 // 頂点色の乗算。テクスチャ面は灰色の減光、フラット面は平均色
	key   float64
	// 以下は VRT の命令列に出すメタ情報。描画そのものには使わない
	kind      string // floor / wallTop / wallSide / billboard
	tile      [2]int // 由来タイルのグリッド座標
	atlasName string // アトラス識別子。壁側面のフラット塗りは flat
}

// r3meta はクアッドへ添える命令列メタ情報をまとめる。addQuad 系の引数肥大を避ける。
type r3meta struct {
	kind  string
	tile  [2]int
	atlas string
}

// R3DrawCommand は3Dシーンを描く1クアッドの命令。VRTで命令列を安定に差分比較するため、
// 投影後スクリーン座標と色を整数へ丸めて float ノイズを排除して記録する。
// 配列の並びが描画順、すなわち画家アルゴリズムでソートした後の奥から手前の順である。
type R3DrawCommand struct {
	Kind   string    // floor / wallTop / wallSide / billboard
	Tile   [2]int    // 由来タイルのグリッド座標
	Atlas  string    // アトラス識別子。壁側面のフラット塗りは flat
	Screen [4][2]int // 投影後スクリーン座標。整数へ丸める
	Color  [3]int    // 頂点色 0..255
}

const (
	r3wallHeight = 1.0  // 壁の高さ。タイル1個分
	r3cullRadius = 60.0 // プレイヤーからこのタイル数だけ描く。カメラの視錐台より広めに取る
)

// visFunc はタイルの明るさ・状態・光源色を返す。bright は 0..1 の明るさで Darkness を反映する。
// drawable はタイルを描くか、visible は今まさに見えているか。light は光源色の乗算で、
// 無色なら {1,1,1}。動体は visible のときだけ描く。
type visFunc func(*gc.GridElement) (bright float64, drawable, visible bool, light [3]float64)

// normalizeLight は光源色を最大成分で正規化した乗算色にする。明るさは bright が持つので、
// ここでは色味だけを表す。無色や未設定は {1,1,1} を返す。
func normalizeLight(c color.RGBA) [3]float64 {
	m := max(c.R, max(c.G, c.B))
	if c.A == 0 || m == 0 {
		return [3]float64{1, 1, 1}
	}
	f := 1.0 / float64(m)
	return [3]float64{float64(c.R) * f, float64(c.G) * f, float64(c.B) * f}
}

// spriteRect は SpriteRender からアトラス画像と切り出し矩形を解決する。見つからなければ ok=false。
func (sys *Render3DSystem) spriteRect(world w.World, sr *gc.SpriteRender) (atlas *ebiten.Image, x, y, ww, hh float64, ok bool) {
	if world.Resources.SpriteSheets == nil {
		return nil, 0, 0, 0, 0, false
	}
	sheet, exists := world.Resources.SpriteSheets[sr.SpriteSheetName]
	if !exists || sheet.Texture.Image == nil {
		return nil, 0, 0, 0, 0, false
	}
	sp, exists := sheet.Sprites[sr.SpriteKey]
	if !exists {
		return nil, 0, 0, 0, 0, false
	}
	return sheet.Texture.Image, float64(sp.X), float64(sp.Y), float64(sp.Width), float64(sp.Height), true
}

// scaleCol は色に係数を掛ける。面ごとのシェードを乗算色へ乗せるのに使う。
func scaleCol(c [3]float64, s float64) [3]float64 {
	return [3]float64{c[0] * s, c[1] * s, c[2] * s}
}

func (sys *Render3DSystem) addQuad(out *[]r3quad, p0, p1, p2, p3 r3vec, atlas *ebiten.Image, x, y, ww, hh float64, col [3]float64, meta r3meta) {
	*out = append(*out, r3quad{
		p:         [4]r3vec{p0, p1, p2, p3},
		uv:        [4][2]float64{{x, y}, {x + ww, y}, {x + ww, y + hh}, {x, y + hh}},
		atlas:     atlas,
		col:       col,
		kind:      meta.kind,
		tile:      meta.tile,
		atlasName: meta.atlas,
	})
}

// addFlatQuad は面をスプライトの平均色でフラットに塗る。壁の側面に使う。真上視点用テクスチャを
// 垂直面へ引き伸ばす違和感を避け、新規アートなしでローポリらしい平板シェードにする。
// 白1pxテクスチャに平均色×shade を頂点色で乗せる。
func (sys *Render3DSystem) addFlatQuad(out *[]r3quad, p0, p1, p2, p3 r3vec, atlas *ebiten.Image, x, y, ww, hh float64, col [3]float64, meta r3meta) {
	c := avgSpriteColor(atlas, x, y, ww, hh)
	*out = append(*out, r3quad{
		p:         [4]r3vec{p0, p1, p2, p3},
		uv:        [4][2]float64{{0, 0}, {0, 0}, {0, 0}, {0, 0}},
		atlas:     whitePixel(),
		col:       [3]float64{c[0] * col[0], c[1] * col[1], c[2] * col[2]},
		kind:      meta.kind,
		tile:      meta.tile,
		atlasName: meta.atlas,
	})
}

type flatColorKey struct {
	atlas      *ebiten.Image
	x, y, w, h int
}

var (
	r3whiteImg  *ebiten.Image
	r3whiteOnce sync.Once
	r3flatColor sync.Map // flatColorKey -> [3]float64
)

// whitePixel はフラット塗り用の白1pxを返す。頂点色をそのまま出すために使う。
func whitePixel() *ebiten.Image {
	r3whiteOnce.Do(func() {
		r3whiteImg = ebiten.NewImage(1, 1)
		r3whiteImg.Fill(color.White)
	})
	return r3whiteImg
}

// avgSpriteColor はアトラス上の矩形の平均色を 0..1 で返す。透明画素は除く。結果はキャッシュする。
func avgSpriteColor(atlas *ebiten.Image, x, y, w, h float64) [3]float64 {
	key := flatColorKey{atlas, int(x), int(y), int(w), int(h)}
	if v, ok := r3flatColor.Load(key); ok {
		if c, ok2 := v.([3]float64); ok2 {
			return c
		}
	}
	var sr, sg, sb, n float64
	for py := int(y); py < int(y+h); py++ {
		for px := int(x); px < int(x+w); px++ {
			cr, cg, cb, ca := atlas.At(px, py).RGBA()
			if ca == 0 {
				continue
			}
			sr += float64(cr >> 8)
			sg += float64(cg >> 8)
			sb += float64(cb >> 8)
			n++
		}
	}
	res := [3]float64{0.5, 0.5, 0.5}
	if n > 0 {
		res = [3]float64{sr / n / 255, sg / n / 255, sb / n / 255}
	}
	r3flatColor.Store(key, res)
	return res
}

// Draw は w.Renderer を満たす。3Dシーンを screen へ描く。
func (sys *Render3DSystem) Draw(world w.World, screen *ebiten.Image) error {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	quads, vp, view := sys.buildScene(world, sw, sh)
	sys.emit(screen, quads, vp, view, sw, sh)
	return nil
}

// buildScene はカメラ行列とクアッド列を組み立てる。Draw と DrawList が共有し、
// 幾何を1箇所に集約する。カメラ・収集ロジックを別々に持つと描画とVRTでズレるため。
func (sys *Render3DSystem) buildScene(world w.World, sw, sh int) (quads []r3quad, vp, view r3mat) {
	pcx, pcz := sys.playerCenter(world)

	target := r3vec{pcx + 0.5, 0.4, pcz + 0.5}
	dist, pitch, yaw := sys.Dist, sys.Pitch, sys.Yaw
	if dist <= 0 {
		dist, pitch, yaw = 9, 0.62, 0 // ゼロ値の保険
	}
	// カメラはプレイヤーの南側から北を見下ろす。画面の上を北に合わせ、北上のミニマップと向きをそろえる
	dir := r3vec{math.Cos(pitch) * math.Sin(yaw), math.Sin(pitch), math.Cos(pitch) * math.Cos(yaw)}
	eye := r3add(target, r3scale(dir, dist))
	up := r3vec{0, 1, 0}
	view = r3lookAt(eye, target, up)
	vp = r3mul(r3perspective(52, float64(sw)/float64(sh), 0.1, 200), view)

	visFactor := sys.visFactorFunc(world)
	quads = sys.collectTiles(world, pcx, pcz, visFactor)
	right := r3norm(r3cross(r3norm(r3sub(target, eye)), up))
	quads = sys.collectBillboards(world, quads, pcx, pcz, right, visFactor)
	return quads, vp, view
}

// DrawList はこのフレームで描くクアッドの命令列を返す。VRTで命令列を安定に差分比較する。
// Draw と同じ buildScene・ソート・投影を通すので、画像とズレない。画面外へ落ちるクアッドは
// Draw と同じく除外する。座標と色は整数へ丸めて float ノイズを排除する。
func (sys *Render3DSystem) DrawList(world w.World, sw, sh int) []R3DrawCommand {
	quads, vp, view := sys.buildScene(world, sw, sh)
	sortQuadsByDepth(quads, view)

	cmds := make([]R3DrawCommand, 0, len(quads))
	for i := range quads {
		q := &quads[i]
		var scr [4][2]int
		ok := true
		for k, p := range q.p {
			x, y, vis := projectToScreen(vp, p, sw, sh)
			if !vis {
				ok = false
				break
			}
			scr[k] = [2]int{int(math.Round(x)), int(math.Round(y))}
		}
		if !ok {
			continue
		}
		cmds = append(cmds, R3DrawCommand{
			Kind:   q.kind,
			Tile:   q.tile,
			Atlas:  q.atlasName,
			Screen: scr,
			Color:  [3]int{clamp255(q.col[0]), clamp255(q.col[1]), clamp255(q.col[2])},
		})
	}
	return cmds
}

// clamp255 は 0..1 の色成分を 0..255 の整数へ丸めて収める。
func clamp255(v float64) int {
	n := int(math.Round(v * 255))
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return n
}

// playerCenter はプレイヤーのタイル座標を返す。いなければ既定値。
func (sys *Render3DSystem) playerCenter(world w.World) (float64, float64) {
	if pe, err := query.GetPlayerEntity(world); err == nil && world.Components.GridElement.Has(pe) {
		g := world.Components.GridElement.Get(pe)
		return float64(g.X), float64(g.Y)
	}
	return 25, 25
}

// visFactorFunc は視界に応じた減光係数を返す関数を作る。隠れタイルは ok=false。
func (sys *Render3DSystem) visFactorFunc(world w.World) visFunc {
	white := [3]float64{1, 1, 1}
	if !sys.UseFOV {
		return func(*gc.GridElement) (float64, bool, bool, [3]float64) { return 1, true, true, white }
	}
	renderMap := computeTileRenderMap(world, query.GetVisionState(world).LightSourceCache)
	return func(g *gc.GridElement) (float64, bool, bool, [3]float64) {
		switch info := renderMap[*g].(type) {
		case TileRenderVisible:
			// Darkness を明るさへ、LightColor を色味へ反映する。松明の淡い照らしと暖色が2Dと同じく出る
			return 1 - float64(info.Darkness), true, true, normalizeLight(info.LightColor)
		case TileRenderRemembered:
			return 1 - float64(info.Darkness), true, false, white
		default:
			return 0, false, false, white
		}
	}
}

// collectTiles は床と壁のクアッドを集める。
func (sys *Render3DSystem) collectTiles(world w.World, pcx, pcz float64, visFactor visFunc) []r3quad {
	var quads []r3quad
	walls := map[[2]int]bool{}
	wallQ := query.ActiveFilter2[gc.GridElement, gc.BlockPass](world).With(ecs.C[gc.Tile]()).Query()
	for wallQ.Next() {
		g := world.Components.GridElement.Get(wallQ.Entity())
		walls[[2]int{int(g.X), int(g.Y)}] = true
	}
	tileQ := query.ActiveFilter3[gc.SpriteRender, gc.GridElement, gc.Tile](world).Query()
	for tileQ.Next() {
		e := tileQ.Entity()
		g := world.Components.GridElement.Get(e)
		fx, fz := float64(g.X), float64(g.Y)
		if math.Abs(fx-pcx) > r3cullRadius || math.Abs(fz-pcz) > r3cullRadius {
			continue
		}
		sr := world.Components.SpriteRender.Get(e)
		atlas, ux, uy, uw, uh, ok := sys.spriteRect(world, sr)
		if !ok {
			continue
		}
		vf, vok, _, light := visFactor(g)
		if !vok {
			continue
		}
		tint := scaleCol(light, vf) // 明るさと光源色を合わせた乗算色
		meta := r3meta{tile: [2]int{int(g.X), int(g.Y)}, atlas: sr.SpriteSheetName}
		if world.Components.BlockPass.Has(e) {
			sys.addWall(&quads, walls, int(g.X), int(g.Y), fx, fz, atlas, ux, uy, uw, uh, tint, meta)
			continue
		}
		meta.kind = "floor"
		sys.addQuad(&quads, r3vec{fx, 0, fz}, r3vec{fx + 1, 0, fz}, r3vec{fx + 1, 0, fz + 1}, r3vec{fx, 0, fz + 1}, atlas, ux, uy, uw, uh, tint, meta)
	}
	return quads
}

// addWall は壁1マスの天面と、隣が壁でない側だけの側面を積む。
func (sys *Render3DSystem) addWall(out *[]r3quad, walls map[[2]int]bool, ix, iy int, fx, fz float64, atlas *ebiten.Image, ux, uy, uw, uh float64, tint [3]float64, meta r3meta) {
	// 天面は真上から見るので既存テクスチャをそのまま貼る。側面はフラット単色にする。
	// tint に面ごとのシェードを掛けて立体感を出す
	top := meta
	top.kind = "wallTop"
	sys.addQuad(out, r3vec{fx, r3wallHeight, fz}, r3vec{fx + 1, r3wallHeight, fz}, r3vec{fx + 1, r3wallHeight, fz + 1}, r3vec{fx, r3wallHeight, fz + 1}, atlas, ux, uy, uw, uh, scaleCol(tint, 0.95), top)
	// 側面はフラット塗りなのでアトラス識別子を flat にする
	side := meta
	side.kind = "wallSide"
	side.atlas = "flat"
	if !walls[[2]int{ix, iy - 1}] {
		sys.addFlatQuad(out, r3vec{fx, 0, fz}, r3vec{fx + 1, 0, fz}, r3vec{fx + 1, r3wallHeight, fz}, r3vec{fx, r3wallHeight, fz}, atlas, ux, uy, uw, uh, scaleCol(tint, 0.6), side)
	}
	if !walls[[2]int{ix, iy + 1}] {
		sys.addFlatQuad(out, r3vec{fx + 1, 0, fz + 1}, r3vec{fx, 0, fz + 1}, r3vec{fx, r3wallHeight, fz + 1}, r3vec{fx + 1, r3wallHeight, fz + 1}, atlas, ux, uy, uw, uh, scaleCol(tint, 0.6), side)
	}
	if !walls[[2]int{ix - 1, iy}] {
		sys.addFlatQuad(out, r3vec{fx, 0, fz + 1}, r3vec{fx, 0, fz}, r3vec{fx, r3wallHeight, fz}, r3vec{fx, r3wallHeight, fz + 1}, atlas, ux, uy, uw, uh, scaleCol(tint, 0.78), side)
	}
	if !walls[[2]int{ix + 1, iy}] {
		sys.addFlatQuad(out, r3vec{fx + 1, 0, fz}, r3vec{fx + 1, 0, fz + 1}, r3vec{fx + 1, r3wallHeight, fz + 1}, r3vec{fx + 1, r3wallHeight, fz}, atlas, ux, uy, uw, uh, scaleCol(tint, 0.78), side)
	}
}

// collectBillboards はタイル以外のエンティティをカメラ向きの立て板として積む。
func (sys *Render3DSystem) collectBillboards(world w.World, quads []r3quad, pcx, pcz float64, right r3vec, visFactor visFunc) []r3quad {
	objQ := query.ActiveFilter2[gc.SpriteRender, gc.GridElement](world).Without(ecs.C[gc.Tile]()).Query()
	for objQ.Next() {
		e := objQ.Entity()
		g := world.Components.GridElement.Get(e)
		fx, fz := float64(g.X), float64(g.Y)
		if math.Abs(fx-pcx) > r3cullRadius || math.Abs(fz-pcz) > r3cullRadius {
			continue
		}
		sr := world.Components.SpriteRender.Get(e)
		atlas, ux, uy, uw, uh, ok := sys.spriteRect(world, sr)
		if !ok {
			continue
		}
		// 動体は今見えているタイルにだけ描く。記憶エリアや隠れエリアには描かない。
		// 2Dの renderObjectLayer と同じで、フォグ内の敵やアイテムは位置を見せない
		b, vok, vis, light := visFactor(g)
		if !vok || !vis {
			continue
		}
		base := r3vec{fx + 0.5, 0, fz + 0.5}
		const bw, bh = 0.45, 1.0
		b0 := r3add(base, r3scale(right, -bw))
		b1 := r3add(base, r3scale(right, bw))
		meta := r3meta{kind: "billboard", tile: [2]int{int(g.X), int(g.Y)}, atlas: sr.SpriteSheetName}
		sys.addQuad(&quads, r3add(b0, r3vec{0, bh, 0}), r3add(b1, r3vec{0, bh, 0}), b1, b0, atlas, ux, uy, uw, uh, scaleCol(light, b), meta)
	}
	return quads
}

// sortQuadsByDepth はカメラ空間の奥行きでクアッドを安定ソートする。画家アルゴリズムの前段。
// Draw の emit と DrawList が同じ並びを得るため共有する。
func sortQuadsByDepth(quads []r3quad, view r3mat) {
	for i := range quads {
		c := r3vec{}
		for _, p := range quads[i].p {
			c = r3add(c, p)
		}
		c = r3scale(c, 0.25)
		_, _, vz, _ := r3apply(view, c)
		quads[i].key = vz
	}
	sort.SliceStable(quads, func(i, j int) bool { return quads[i].key < quads[j].key })
}

// projectToScreen は点をスクリーン座標へ投影する。カメラ後方や w<=0 の点は ok=false。
func projectToScreen(vp r3mat, p r3vec, sw, sh int) (float64, float64, bool) {
	cx, cy, _, cw := r3apply(vp, p)
	if cw <= 0.001 {
		return 0, 0, false
	}
	return (cx/cw*0.5 + 0.5) * float64(sw), (1 - (cy/cw*0.5 + 0.5)) * float64(sh), true
}

// emit はクアッドを奥行きでソートし、アトラスが変わる境目でバッチに分けて描く。
func (sys *Render3DSystem) emit(screen *ebiten.Image, quads []r3quad, vp, view r3mat, sw, sh int) {
	sortQuadsByDepth(quads, view)

	var verts []ebiten.Vertex
	var inds []uint16
	var curAtlas *ebiten.Image
	flush := func() {
		if len(inds) == 0 || curAtlas == nil {
			return
		}
		screen.DrawTriangles(verts, inds, curAtlas, &ebiten.DrawTrianglesOptions{})
		verts = verts[:0]
		inds = inds[:0]
	}
	project := func(p r3vec) (float64, float64, bool) {
		return projectToScreen(vp, p, sw, sh)
	}
	for i := range quads {
		q := &quads[i]
		if q.atlas != curAtlas {
			flush()
			curAtlas = q.atlas
		}
		sys.emitQuad(&verts, &inds, q, project)
	}
	flush()
}

// emitQuad は1クアッドを三角形2枚として頂点バッファへ積む。画面外の頂点があれば捨てる。
func (sys *Render3DSystem) emitQuad(verts *[]ebiten.Vertex, inds *[]uint16, q *r3quad, project func(r3vec) (float64, float64, bool)) {
	var sp [4][2]float64
	for k, p := range q.p {
		sx, sy, ok := project(p)
		if !ok {
			return
		}
		sp[k] = [2]float64{sx, sy}
	}
	b := uint16(len(*verts))
	cr, cg, cb := float32(q.col[0]), float32(q.col[1]), float32(q.col[2])
	for k := range 4 {
		*verts = append(*verts, ebiten.Vertex{
			DstX: float32(sp[k][0]), DstY: float32(sp[k][1]),
			SrcX: float32(q.uv[k][0]), SrcY: float32(q.uv[k][1]),
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: 1,
		})
	}
	*inds = append(*inds, b, b+1, b+2, b, b+2, b+3)
}
