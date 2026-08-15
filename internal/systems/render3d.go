package systems

import (
	"math"
	"sort"

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
	shade float64
	key   float64
}

const (
	r3wallHeight = 1.0  // 壁の高さ。タイル1個分
	r3cullRadius = 18.0 // プレイヤーからこのタイル数だけ描く
)

type visFunc func(*gc.GridElement) (float64, bool)

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

func (sys *Render3DSystem) addQuad(out *[]r3quad, p0, p1, p2, p3 r3vec, atlas *ebiten.Image, x, y, ww, hh, shade float64) {
	*out = append(*out, r3quad{
		p:     [4]r3vec{p0, p1, p2, p3},
		uv:    [4][2]float64{{x, y}, {x + ww, y}, {x + ww, y + hh}, {x, y + hh}},
		atlas: atlas,
		shade: shade,
	})
}

// Draw は w.Renderer を満たす。3Dシーンを screen へ描く。
func (sys *Render3DSystem) Draw(world w.World, screen *ebiten.Image) error {
	pcx, pcz := sys.playerCenter(world)

	target := r3vec{pcx + 0.5, 0.4, pcz + 0.5}
	dist, pitch, yaw := sys.Dist, sys.Pitch, sys.Yaw
	if dist <= 0 {
		dist, pitch, yaw = 9, 0.62, 0 // ゼロ値の保険
	}
	dir := r3vec{math.Cos(pitch) * math.Sin(yaw), math.Sin(pitch), -math.Cos(pitch) * math.Cos(yaw)}
	eye := r3add(target, r3scale(dir, dist))
	up := r3vec{0, 1, 0}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	view := r3lookAt(eye, target, up)
	vp := r3mul(r3perspective(52, float64(sw)/float64(sh), 0.1, 200), view)

	visFactor := sys.visFactorFunc(world)
	quads := sys.collectTiles(world, pcx, pcz, visFactor)
	right := r3norm(r3cross(r3norm(r3sub(target, eye)), up))
	quads = sys.collectBillboards(world, quads, pcx, pcz, right, visFactor)
	sys.emit(screen, quads, vp, view, sw, sh)
	return nil
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
	if !sys.UseFOV {
		return func(*gc.GridElement) (float64, bool) { return 1, true }
	}
	renderMap := computeTileRenderMap(world, query.GetVisionState(world).LightSourceCache)
	return func(g *gc.GridElement) (float64, bool) {
		switch renderMap[*g].(type) {
		case TileRenderVisible:
			return 1, true
		case TileRenderRemembered:
			return 0.35, true
		default:
			return 0, false
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
		atlas, ux, uy, uw, uh, ok := sys.spriteRect(world, world.Components.SpriteRender.Get(e))
		if !ok {
			continue
		}
		vf, vok := visFactor(g)
		if !vok {
			continue
		}
		if world.Components.BlockPass.Has(e) {
			sys.addWall(&quads, walls, int(g.X), int(g.Y), fx, fz, atlas, ux, uy, uw, uh, vf)
			continue
		}
		sys.addQuad(&quads, r3vec{fx, 0, fz}, r3vec{fx + 1, 0, fz}, r3vec{fx + 1, 0, fz + 1}, r3vec{fx, 0, fz + 1}, atlas, ux, uy, uw, uh, vf)
	}
	return quads
}

// addWall は壁1マスの天面と、隣が壁でない側だけの側面を積む。
func (sys *Render3DSystem) addWall(out *[]r3quad, walls map[[2]int]bool, ix, iy int, fx, fz float64, atlas *ebiten.Image, ux, uy, uw, uh, vf float64) {
	sys.addQuad(out, r3vec{fx, r3wallHeight, fz}, r3vec{fx + 1, r3wallHeight, fz}, r3vec{fx + 1, r3wallHeight, fz + 1}, r3vec{fx, r3wallHeight, fz + 1}, atlas, ux, uy, uw, uh, 0.9*vf)
	if !walls[[2]int{ix, iy - 1}] {
		sys.addQuad(out, r3vec{fx, 0, fz}, r3vec{fx + 1, 0, fz}, r3vec{fx + 1, r3wallHeight, fz}, r3vec{fx, r3wallHeight, fz}, atlas, ux, uy, uw, uh, 0.62*vf)
	}
	if !walls[[2]int{ix, iy + 1}] {
		sys.addQuad(out, r3vec{fx + 1, 0, fz + 1}, r3vec{fx, 0, fz + 1}, r3vec{fx, r3wallHeight, fz + 1}, r3vec{fx + 1, r3wallHeight, fz + 1}, atlas, ux, uy, uw, uh, 0.62*vf)
	}
	if !walls[[2]int{ix - 1, iy}] {
		sys.addQuad(out, r3vec{fx, 0, fz + 1}, r3vec{fx, 0, fz}, r3vec{fx, r3wallHeight, fz}, r3vec{fx, r3wallHeight, fz + 1}, atlas, ux, uy, uw, uh, 0.78*vf)
	}
	if !walls[[2]int{ix + 1, iy}] {
		sys.addQuad(out, r3vec{fx + 1, 0, fz}, r3vec{fx + 1, 0, fz + 1}, r3vec{fx + 1, r3wallHeight, fz + 1}, r3vec{fx + 1, r3wallHeight, fz}, atlas, ux, uy, uw, uh, 0.78*vf)
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
		atlas, ux, uy, uw, uh, ok := sys.spriteRect(world, world.Components.SpriteRender.Get(e))
		if !ok {
			continue
		}
		if _, vok := visFactor(g); !vok {
			continue
		}
		base := r3vec{fx + 0.5, 0, fz + 0.5}
		const bw, bh = 0.45, 1.0
		b0 := r3add(base, r3scale(right, -bw))
		b1 := r3add(base, r3scale(right, bw))
		sys.addQuad(&quads, r3add(b0, r3vec{0, bh, 0}), r3add(b1, r3vec{0, bh, 0}), b1, b0, atlas, ux, uy, uw, uh, 1)
	}
	return quads
}

// emit はクアッドを奥行きでソートし、アトラスが変わる境目でバッチに分けて描く。
func (sys *Render3DSystem) emit(screen *ebiten.Image, quads []r3quad, vp, view r3mat, sw, sh int) {
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
		cx, cy, _, cw := r3apply(vp, p)
		if cw <= 0.001 {
			return 0, 0, false
		}
		return (cx/cw*0.5 + 0.5) * float64(sw), (1 - (cy/cw*0.5 + 0.5)) * float64(sh), true
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
	sc := float32(q.shade)
	for k := range 4 {
		*verts = append(*verts, ebiten.Vertex{
			DstX: float32(sp[k][0]), DstY: float32(sp[k][1]),
			SrcX: float32(q.uv[k][0]), SrcY: float32(q.uv[k][1]),
			ColorR: sc, ColorG: sc, ColorB: sc, ColorA: 1,
		})
	}
	*inds = append(*inds, b, b+1, b+2, b, b+2, b+3)
}
