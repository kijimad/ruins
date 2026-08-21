package systems

import (
	"image/color"
	"math"
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/render3d"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// Render3DSystem は壁と床をローポリの3Dで描くレンダラである。
// 既存の RenderSpriteSystem と同じ ECS ワールドを読み、床タイルを水平クアッド、
// 壁タイルを箱、それ以外のエンティティをビルボードとして透視投影する。
// テクスチャは既存スプライトシートをそのまま流用する。Ebiten は深度バッファを持たないため、
// 各クアッドをカメラ空間の奥行きで並べ替える画家アルゴリズムで隠面を解く。
type Render3DSystem struct {
	// UseFOV は視界データに従い、隠れたタイルを描かず記憶タイルを減光する。
	// 本番のダンジョンでは true、部屋全体を見せたいデモでは false にする
	UseFOV bool
}

// NewRender3DSystem は視界を反映する本番の設定で初期化する。
// オービットカメラの向きと距離は ECS の Camera が持ち、Projector 経由で読む。
func NewRender3DSystem() *Render3DSystem {
	return &Render3DSystem{UseFOV: true}
}

// String は w.Renderer を満たす。
func (sys *Render3DSystem) String() string { return "Render3DSystem" }

// --- クアッド ---

type r3quad struct {
	p     [4]render3d.Vec
	uv    [4][2]float64
	atlas *ebiten.Image
	col   [3]float64 // 頂点色の乗算。テクスチャ面は灰色の減光、フラット面は平均色
	alpha float64    // 頂点アルファ。既定は不透明の1。霜など半透明の重ねだけ1未満にする
	key   float64
	// depth は画家ソートの副キー。quad は元エンティティを持たないので、立て板を作るときにSpriteRender.Depth をここへ焼き込む
	depth int
}

// r3cullRadius はプレイヤーからこのタイル数だけ描く。カメラの視錐台より広めに取る
const r3cullRadius = 60.0

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

func (sys *Render3DSystem) addQuad(out *[]r3quad, p0, p1, p2, p3 render3d.Vec, atlas *ebiten.Image, x, y, ww, hh float64, col [3]float64) {
	*out = append(*out, r3quad{
		p:     [4]render3d.Vec{p0, p1, p2, p3},
		uv:    [4][2]float64{{x, y}, {x + ww, y}, {x + ww, y + hh}, {x, y + hh}},
		atlas: atlas,
		col:   col,
		alpha: 1,
	})
}

// addFlatQuad は面をスプライトの平均色でフラットに塗る。壁の側面に使う。真上視点用テクスチャを
// 垂直面へ引き伸ばす違和感を避け、新規アートなしでローポリらしい平板シェードにする。
func (sys *Render3DSystem) addFlatQuad(out *[]r3quad, p0, p1, p2, p3 render3d.Vec, atlas *ebiten.Image, x, y, ww, hh float64, col [3]float64) {
	c := avgSpriteColor(atlas, x, y, ww, hh)
	*out = append(*out, r3quad{
		p:     [4]render3d.Vec{p0, p1, p2, p3},
		uv:    [4][2]float64{{0, 0}, {0, 0}, {0, 0}, {0, 0}},
		atlas: whitePixel(),
		col:   [3]float64{c[0] * col[0], c[1] * col[1], c[2] * col[2]},
		alpha: 1,
	})
}

// frostColorNorm は氷オーバーレイの色を 0..1 で表す。2Dの FrostRenderSystem と同じ寒色。
var frostColorNorm = [3]float64{130.0 / 255, 205.0 / 255, 240.0 / 255}

// addFrostQuad は床や壁天面へ氷を重ねる。タイルは一様な四角なので白1pxで塗りつぶす。
func (sys *Render3DSystem) addFrostQuad(out *[]r3quad, fx, fz, topY, alpha float64) {
	p := [4]render3d.Vec{render3d.At(fx, topY, fz), render3d.At(fx+1, topY, fz), render3d.At(fx+1, topY, fz+1), render3d.At(fx, topY, fz+1)}
	zeroUV := [4][2]float64{{0, 0}, {0, 0}, {0, 0}, {0, 0}}
	appendFrostQuad(out, p, zeroUV, whitePixel(), alpha)
}

// appendFrostQuad は氷クアッドを1枚積む。頂点色を氷色×alpha にプリマルチプライしアルファを渡すことで、
// ソースオーバー合成が 2D の氷オーバーレイと同じになる。atlas と uv を受け、床・壁は白1px、ビルボードは
// スプライトのテクスチャで貼る。スプライトで貼るとアルファでマスクされ透明な余白に氷が乗らない。
func appendFrostQuad(out *[]r3quad, p [4]render3d.Vec, uv [4][2]float64, atlas *ebiten.Image, alpha float64) {
	*out = append(*out, r3quad{
		p:     p,
		uv:    uv,
		atlas: atlas,
		col:   [3]float64{frostColorNorm[0] * alpha, frostColorNorm[1] * alpha, frostColorNorm[2] * alpha},
		alpha: alpha,
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
	quads, projector := sys.buildScene(world)
	sys.emit(screen, quads, projector)
	return nil
}

// buildScene は投影とクアッド列を組み立てる。Draw の幾何を1箇所に集約する。
func (sys *Render3DSystem) buildScene(world w.World) ([]r3quad, render3d.Projector) {
	projector := render3d.For(world)
	center := render3d.PlayerTile(world)
	pcx, pcz := float64(center.X), float64(center.Y)

	visFactor := sys.visFactorFunc(world)
	frost := sys.frostFunc(world)
	quads := sys.collectTiles(world, pcx, pcz, visFactor, frost)
	quads = sys.collectBillboards(world, quads, pcx, pcz, projector.Right(), visFactor, frost)
	return quads, projector
}

// visFactorFunc は視界に応じた減光係数を返す関数を作る。隠れタイルは ok=false。
func (sys *Render3DSystem) visFactorFunc(world w.World) visFunc {
	if !sys.UseFOV {
		return func(*gc.GridElement) (float64, bool, bool, [3]float64) { return 1, true, true, [3]float64{1, 1, 1} }
	}
	renderMap := computeTileRenderMap(world, query.GetVisionState(world).LightSourceCache)
	return func(g *gc.GridElement) (float64, bool, bool, [3]float64) {
		return tileVisFactor(renderMap[*g])
	}
}

// tileVisFactor はタイルの描画情報から明るさ・描画可否・可視・光源色を導く純関数。
// 可視は Darkness を 1-d の明るさへ、LightColor を色味へ反映する。記憶は visible=false、未探索は描かない。
func tileVisFactor(info TileRenderInfo) (bright float64, drawable, visible bool, light [3]float64) {
	white := [3]float64{1, 1, 1}
	switch v := info.(type) {
	case TileRenderVisible:
		return 1 - float64(v.Darkness), true, true, normalizeLight(v.LightColor)
	case TileRenderRemembered:
		return 1 - float64(v.Darkness), true, false, white
	default:
		return 0, false, false, white
	}
}

// frostFunc は寒波前線の氷を載せる判定を作る。オーバーワールドでなければ常に載せない。
// タイル列の絶対Xから frostAlpha でアルファと可否を返す。frostAlpha は2Dの霜と共有する純関数。
func (sys *Render3DSystem) frostFunc(world w.World) func(tileX int) (alpha float64, draw bool) {
	if !query.IsOnOverworld(world) {
		return func(int) (float64, bool) { return 0, false }
	}
	sb := *query.GetSeamlessBand(world)
	frontEast := int(sb.Front.EastAbsX)
	coldZoneWest := int(sb.Front.ColdZoneWest())
	return func(tileX int) (float64, bool) {
		a, d := frostAlpha(frontEast, coldZoneWest, int(sb.LocalToAbsX(consts.Tile(tileX))))
		return float64(a), d
	}
}

// collectTiles は床と壁のクアッドを集める。可視タイルには寒波前線の氷を重ねる。
func (sys *Render3DSystem) collectTiles(world w.World, pcx, pcz float64, visFactor visFunc, frost func(int) (float64, bool)) []r3quad {
	var quads []r3quad
	walls := render3d.WallTileSet(world)
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
		vf, vok, vis, light := visFactor(g)
		if !vok {
			continue
		}
		tint := scaleCol(light, vf) // 明るさと光源色を合わせた乗算色
		isWall := render3d.IsWallTileEntity(world, e)
		if isWall {
			sys.addWall(&quads, walls, g.Coord, fx, fz, atlas, ux, uy, uw, uh, tint)
		} else {
			sys.addQuad(&quads, render3d.At(fx, 0, fz), render3d.At(fx+1, 0, fz), render3d.At(fx+1, 0, fz+1), render3d.At(fx, 0, fz+1), atlas, ux, uy, uw, uh, tint)
		}
		// 霜は今見えているタイルにだけ載せる。床は地面、壁は天面の高さへ、z-fight を避け少し上へ重ねる
		if vis {
			if a, d := frost(int(g.X)); d {
				topY := 0.02
				if isWall {
					topY = render3d.WallHeight + 0.02
				}
				sys.addFrostQuad(&quads, fx, fz, topY, a)
			}
		}
	}
	return quads
}

// addWall は壁1マスの天面と、隣が壁でない側だけの側面を積む。
func (sys *Render3DSystem) addWall(out *[]r3quad, walls map[consts.Coord[consts.Tile]]bool, c consts.Coord[consts.Tile], fx, fz float64, atlas *ebiten.Image, ux, uy, uw, uh float64, tint [3]float64) {
	// 天面は真上視点なので既存テクスチャをそのまま貼り、側面はフラット単色にする。
	// 面ごとのシェード 天面0.95・南北0.6・東西0.78 は疑似方向光源で、平板な側面に陰影を付けて立体に見せる
	sys.addQuad(out, render3d.At(fx, render3d.WallHeight, fz), render3d.At(fx+1, render3d.WallHeight, fz), render3d.At(fx+1, render3d.WallHeight, fz+1), render3d.At(fx, render3d.WallHeight, fz+1), atlas, ux, uy, uw, uh, scaleCol(tint, 0.95))
	if !walls[c.Add(consts.Coord[consts.Tile]{X: 0, Y: -1})] {
		sys.addFlatQuad(out, render3d.At(fx, 0, fz), render3d.At(fx+1, 0, fz), render3d.At(fx+1, render3d.WallHeight, fz), render3d.At(fx, render3d.WallHeight, fz), atlas, ux, uy, uw, uh, scaleCol(tint, 0.6))
	}
	if !walls[c.Add(consts.Coord[consts.Tile]{X: 0, Y: 1})] {
		sys.addFlatQuad(out, render3d.At(fx+1, 0, fz+1), render3d.At(fx, 0, fz+1), render3d.At(fx, render3d.WallHeight, fz+1), render3d.At(fx+1, render3d.WallHeight, fz+1), atlas, ux, uy, uw, uh, scaleCol(tint, 0.6))
	}
	if !walls[c.Add(consts.Coord[consts.Tile]{X: -1, Y: 0})] {
		sys.addFlatQuad(out, render3d.At(fx, 0, fz+1), render3d.At(fx, 0, fz), render3d.At(fx, render3d.WallHeight, fz), render3d.At(fx, render3d.WallHeight, fz+1), atlas, ux, uy, uw, uh, scaleCol(tint, 0.78))
	}
	if !walls[c.Add(consts.Coord[consts.Tile]{X: 1, Y: 0})] {
		sys.addFlatQuad(out, render3d.At(fx+1, 0, fz), render3d.At(fx+1, 0, fz+1), render3d.At(fx+1, render3d.WallHeight, fz+1), render3d.At(fx+1, render3d.WallHeight, fz), atlas, ux, uy, uw, uh, scaleCol(tint, 0.78))
	}
}

// collectBillboards はタイル以外のエンティティをカメラ向きの立て板として積む。可視かつ凍結タイルの
// エンティティには、同じ立て板の位置へ氷を重ねる。2Dが霜をスプライトの上に載せるのと同じにする。
func (sys *Render3DSystem) collectBillboards(world w.World, quads []r3quad, pcx, pcz float64, right render3d.Vec, visFactor visFunc, frost func(int) (float64, bool)) []r3quad {
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
		// 動体は今見えているタイルにだけ描く。フォグ内や記憶エリアの敵・アイテムは位置を見せない
		b, vok, vis, light := visFactor(g)
		if !vok || !vis {
			continue
		}
		base := render3d.At(fx+0.5, 0, fz+0.5)
		const bw = 0.45
		const bh = render3d.BillboardHeight
		b0 := render3d.Add(base, render3d.Scale(right, -bw))
		b1 := render3d.Add(base, render3d.Scale(right, bw))
		top := render3d.At(0, bh, 0)
		tl, tr := render3d.Add(b0, top), render3d.Add(b1, top)
		// 立て板の上下は SpriteRender.Depth で決める。同一タイルのプレイヤーとアイテムは4隅が
		// 一致して奥行きが同値になるので、この副キーが無いと走査順で前後がばらつく
		depth := int(sr.Depth)
		sys.addQuad(&quads, tl, tr, b1, b0, atlas, ux, uy, uw, uh, scaleCol(light, b))
		quads[len(quads)-1].depth = depth
		// エンティティの立て板へ氷を重ねる。スプライトのテクスチャとUVで貼り、透明な余白は氷が乗らない。
		// 立て板と同じ depth を与え、同値時は挿入順で氷を後に、つまり手前に描く
		if a, d := frost(int(g.X)); d {
			uv := [4][2]float64{{ux, uy}, {ux + uw, uy}, {ux + uw, uy + uh}, {ux, uy + uh}}
			appendFrostQuad(&quads, [4]render3d.Vec{tl, tr, b1, b0}, uv, atlas, a)
			quads[len(quads)-1].depth = depth
		}
	}
	return quads
}

// sortQuadsByDepth はカメラ空間の奥行きでクアッドを安定ソートする。画家アルゴリズムの前段。
func sortQuadsByDepth(quads []r3quad, depth func(render3d.Vec) float64) {
	for i := range quads {
		c := render3d.Vec{}
		for _, p := range quads[i].p {
			c = render3d.Add(c, p)
		}
		quads[i].key = depth(render3d.Scale(c, 0.25))
	}
	// 奥行きが同値のときは depth で割る。同一タイルの立て板はここで前後が確定する。
	// 大きい depth を後に描いて手前にし、プレイヤーを足元のアイテムより必ず上へ出す
	sort.SliceStable(quads, func(i, j int) bool {
		if quads[i].key != quads[j].key {
			return quads[i].key < quads[j].key
		}
		return quads[i].depth < quads[j].depth
	})
}

// emit はクアッドを奥行きでソートし、アトラスが変わる境目でバッチに分けて描く。
func (sys *Render3DSystem) emit(screen *ebiten.Image, quads []r3quad, projector render3d.Projector) {
	sortQuadsByDepth(quads, projector.Depth)

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
	for i := range quads {
		q := &quads[i]
		// アトラス切り替え、または uint16 の頂点インデックス上限を跨ぐ前に flush する。
		// 同一スプライトシートのタイルが大量に積まれても 65535 を超えると黙って描画化けするため
		if q.atlas != curAtlas || len(verts)+4 > maxVertsPerBatch {
			flush()
			curAtlas = q.atlas
		}
		sys.emitQuad(&verts, &inds, q, projector.Point)
	}
	flush()
}

// maxVertsPerBatch は1回の DrawTriangles へ積む頂点数の上限。インデックスが uint16 なので 65535 まで。
const maxVertsPerBatch = 65535

// emitQuad は1クアッドを三角形2枚として頂点バッファへ積む。画面外の頂点があれば捨てる。
func (sys *Render3DSystem) emitQuad(verts *[]ebiten.Vertex, inds *[]uint16, q *r3quad, project func(render3d.Vec) (consts.Coord[consts.ScreenPixel], bool)) {
	var sp [4]consts.Coord[consts.ScreenPixel]
	for k, p := range q.p {
		screenPos, ok := project(p)
		if !ok {
			return
		}
		sp[k] = screenPos
	}
	b := uint16(len(*verts))
	cr, cg, cb, ca := float32(q.col[0]), float32(q.col[1]), float32(q.col[2]), float32(q.alpha)
	for k := range 4 {
		*verts = append(*verts, ebiten.Vertex{
			DstX: float32(sp[k].X), DstY: float32(sp[k].Y),
			SrcX: float32(q.uv[k][0]), SrcY: float32(q.uv[k][1]),
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		})
	}
	*inds = append(*inds, b, b+1, b+2, b, b+2, b+3)
}
