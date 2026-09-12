package systems

import (
	"image"
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
	// 本番のダンジョンでは true、部屋全体を見せたいデモでは false にする。
	// TODO: この切り替えは system でなく設定で保持する。設計は docs/design/260822112145.md
	UseFOV bool
	// geoLow は幾何を直接ラスタライズする低解像度バッファ。sprBuf はスプライトを原寸で描き、
	// maskBuf は原寸の遮蔽マスク。毎フレーム作り直さず使い回す
	geoLow, sprBuf, maskBuf *ebiten.Image
}

// pixelScale は幾何を 1/pixelScale の解像度へラスタライズして最近傍拡大する倍率。輪郭もテクスチャも
// 一緒に粗くなり、スーファミ風の一貫した低ポリになる。スプライトは原寸で描き、壁の手前だけマスクで
// 消して遮蔽を保つ。値はスプライトのドット粒度に合うよう選ぶ。
const pixelScale = 3

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
	// occluder は遮蔽マスク専用の種別。幾何は true で後ろを削り、スプライトは false で塗る
	occluder bool
}

// r3cullRadius はプレイヤーからこのタイル数だけ描く。カメラの視錐台より広めに取る
const r3cullRadius = 60.0

// dayOverbright* は屋外の日照が強いほどタイルの明るさを乗算の天井 1.0 超へ持ち上げる帯。
// テクスチャ本来の明るさを越えて晴天らしく明るく見せる。dayOverbrightLow 以下では持ち上げない。
// vision.go の lightFadeHigh と同じ overworldDaylight スケールの値で、役割ごとに段階が分かれる。
// まず日照 lightFadeHigh までに光源の寄与を消し、そこからさらに明るい dayOverbrightHigh に向けて日中の底上げが最大になる。
const (
	dayOverbrightMax  = 0.3
	dayOverbrightLow  = 0.6
	dayOverbrightHigh = 0.95
)

// tintFunc はタイルへ乗せる乗算色 tint と描画状態を返す。tint は明るさと光源色を合成済み。
// drawable はタイルを描くか、visible は今まさに見えているか。動体は visible のときだけ描く。
type tintFunc func(*gc.GridElement) (tint [3]float64, drawable, visible bool)

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
	tex, rect, ok := world.Resources.Sprites.Rect(sr)
	if !ok {
		return nil, 0, 0, 0, 0, false
	}
	return tex.Image, float64(rect.Min.X), float64(rect.Min.Y), float64(rect.Dx()), float64(rect.Dy()), true
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
// アトラス配置に画素が左右されないよう unmanaged で作る。機構は components.Texture を参照
func whitePixel() *ebiten.Image {
	r3whiteOnce.Do(func() {
		r3whiteImg = ebiten.NewImageWithOptions(image.Rect(0, 0, 1, 1), &ebiten.NewImageOptions{Unmanaged: true})
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

// Draw は w.Renderer を満たす。幾何(床・壁)を低解像度で直接ラスタライズして最近傍拡大し、スーファミ風の
// くっきりした低ポリにする。スプライトは原寸で鮮明に描く。深度バッファが無いので、幾何とスプライトを
// 深度順に焼いた遮蔽マスクで、壁の手前のスプライトだけを削って遮蔽を保つ。
func (sys *Render3DSystem) Draw(world w.World, screen *ebiten.Image) error {
	sw, sh := world.Resources.GetScreenDimensions()
	fullProj, err := render3d.WorldProjectorSized(world, sw, sh)
	if err != nil {
		return err
	}
	center, err := render3d.PlayerTile(world)
	if err != nil {
		return err
	}
	pcx, pcz := float64(center.X), float64(center.Y)
	visTint := sys.visTintFunc(world)

	geo := sys.collectTiles(world, pcx, pcz, visTint)
	var spr []r3quad
	spr = sys.collectBillboards(world, spr, pcx, pcz, fullProj.Right(), visTint)
	// 状態従属の装飾もスプライトとして積み、同じ遮蔽で手前の壁に隠させる
	spr = sys.collectDecorations(world, spr, fullProj)

	if err := sys.drawGeometry(world, screen, geo, sw, sh); err != nil {
		return err
	}
	sys.compositeSprites(screen, geo, spr, fullProj, sw, sh)
	return nil
}

// drawGeometry は幾何を低解像度バッファへ直接ラスタライズし、最近傍で screen へ拡大する。原寸で描いて
// から縮小せず最初から低解像度で描くので、平均化のぼやけも縮小モアレも出ず、くっきりしたドットになる。
func (sys *Render3DSystem) drawGeometry(world w.World, screen *ebiten.Image, geo []r3quad, sw, sh int) error {
	lw, lh := sw/pixelScale, sh/pixelScale
	lowProj, err := render3d.WorldProjectorSized(world, lw, lh)
	if err != nil {
		return err
	}
	sys.geoLow = ensureBuffer(sys.geoLow, lw, lh)
	sys.geoLow.Clear()
	// 低解像度の幾何は線形標本化。後退する床のテクセルの拾いがなめらかになり、移動時のちらつきを抑える。
	// 拡大は最近傍のままなので大きなドット感は保たれる
	sys.emit(sys.geoLow, geo, lowProj, ebiten.Blend{}, ebiten.FilterLinear)
	up := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	up.GeoM.Scale(float64(pixelScale), float64(pixelScale))
	screen.DrawImage(sys.geoLow, up)
	return nil
}

// compositeSprites は原寸スプライトを描き、遮蔽マスクで壁の手前のぶんだけ削って幾何の上へ重ねる。
// マスクは深度順に、スプライトを source-over で塗り、幾何を destination-out で削る。手前の幾何だけが
// スプライトを切り、スプライト同士は透明部分で欠けない。
func (sys *Render3DSystem) compositeSprites(screen *ebiten.Image, geo, spr []r3quad, projector render3d.Projector, sw, sh int) {
	sys.sprBuf = ensureBuffer(sys.sprBuf, sw, sh)
	sys.maskBuf = ensureBuffer(sys.maskBuf, sw, sh)

	sys.sprBuf.Clear()
	sys.emit(sys.sprBuf, spr, projector, ebiten.Blend{}, ebiten.FilterNearest)

	mask := make([]r3quad, 0, len(geo)+len(spr))
	for i := range geo {
		q := geo[i]
		q.occluder = true
		mask = append(mask, q)
	}
	for i := range spr {
		q := spr[i]
		q.occluder = false
		mask = append(mask, q)
	}
	sys.maskBuf.Clear()
	sys.emitMask(sys.maskBuf, mask, projector)

	// マスクの α で原寸スプライトを削る。壁の手前のスプライトは α0 で消える。等倍なので縁は保たれる
	cut := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn}
	sys.sprBuf.DrawImage(sys.maskBuf, cut)

	screen.DrawImage(sys.sprBuf, nil)
}

// ensureBuffer は指定サイズの描画バッファを返す。サイズが違えば作り直す
func ensureBuffer(buf *ebiten.Image, w, h int) *ebiten.Image {
	if buf == nil || buf.Bounds().Dx() != w || buf.Bounds().Dy() != h {
		return ebiten.NewImage(w, h)
	}
	return buf
}

// lightSample は解決済みの明るさと色。フィルタ列で乗算色まで変換する。
type lightSample struct {
	brightness float64    // 0..1 の明るさ。overbright で 1.0 を超えうる
	color      [3]float64 // 光源色の乗算色。無色なら {1,1,1}
}

// sampleOf はタイルの描画情報から明るさと色のサンプルを起こす。可視は Darkness を 1-d の
// 明るさへ、LightColor を色味へ反映する。記憶は visible=false、未探索は drawable=false。
func sampleOf(info TileRenderInfo) (s lightSample, drawable, visible bool) {
	switch v := info.(type) {
	case TileRenderVisible:
		return lightSample{1 - float64(v.Darkness), normalizeLight(v.LightColor)}, true, true
	case TileRenderRemembered:
		return lightSample{1 - float64(v.Darkness), [3]float64{1, 1, 1}}, true, false
	default:
		return lightSample{}, false, false
	}
}

// overbright は明るさを boost 倍に持ち上げる。
func (s lightSample) overbright(boost float64) lightSample {
	s.brightness *= boost
	return s
}

// tint はサンプルを乗算色にする。
func (s lightSample) tint() [3]float64 {
	return scaleCol(s.color, s.brightness)
}

// visTintFunc はタイルへ乗せる乗算色 tint を返す関数を作る。隠れタイルは drawable=false。
func (sys *Render3DSystem) visTintFunc(world w.World) tintFunc {
	if !sys.UseFOV {
		return func(*gc.GridElement) ([3]float64, bool, bool) { return [3]float64{1, 1, 1}, true, true }
	}
	renderMap := computeTileRenderMap(world, query.GetVisionState(world).LightSourceCache)
	// 屋外の日照が強いほど明るさを 1.0 超へ持ち上げ、乗算の天井を越えて晴天を明るく見せる
	boost := 1.0
	if query.IsOnOverworld(world) {
		boost = 1 + dayOverbrightMax*smoothstep(dayOverbrightLow, dayOverbrightHigh, overworldDaylight(query.GetGameTime(world)))
	}
	return func(g *gc.GridElement) ([3]float64, bool, bool) {
		s, drawable, visible := sampleOf(renderMap[*g])
		if visible {
			s = s.overbright(boost)
		}
		return s.tint(), drawable, visible
	}
}

// collectTiles は床と壁のクアッドを集める。
func (sys *Render3DSystem) collectTiles(world w.World, pcx, pcz float64, visTint tintFunc) []r3quad {
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
		tint, vok, _ := visTint(g)
		if !vok {
			continue
		}
		if render3d.IsWallTileEntity(world, e) {
			sys.addWall(&quads, walls, g.Coord, fx, fz, atlas, ux, uy, uw, uh, tint)
		} else {
			sys.addQuad(&quads, render3d.At(fx, 0, fz), render3d.At(fx+1, 0, fz), render3d.At(fx+1, 0, fz+1), render3d.At(fx, 0, fz+1), atlas, ux, uy, uw, uh, tint)
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

// collectBillboards はタイル以外のエンティティをカメラ向きの立て板として積む。
func (sys *Render3DSystem) collectBillboards(world w.World, quads []r3quad, pcx, pcz float64, right render3d.Vec, visTint tintFunc) []r3quad {
	// 運転中プレイヤーは Driving を持つので描画クエリから外す。entity は残り被弾対象のまま
	objQ := query.ActiveFilter2[gc.SpriteRender, gc.GridElement](world).Without(ecs.C[gc.Tile](), ecs.C[gc.Driving]()).Query()
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
		tint, vok, vis := visTint(g)
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
		sys.addQuad(&quads, tl, tr, b1, b0, atlas, ux, uy, uw, uh, tint)
		quads[len(quads)-1].depth = depth
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
// blend は合成方法。通常描画は既定の source-over、遮蔽マスクは前面で上書きする Copy を渡す。
// filter はテクスチャ標本化。スプライトとマスクは最近傍で鮮明さと硬いエッジを保つ。低解像度の幾何は
// 線形にすると、後退する床のテクセルの拾いがフレーム間でなめらかになり、動いたときのちらつきが減る。
func (sys *Render3DSystem) emit(screen *ebiten.Image, quads []r3quad, projector render3d.Projector, blend ebiten.Blend, filter ebiten.Filter) {
	sortQuadsByDepth(quads, projector.Depth)

	var verts []ebiten.Vertex
	var inds []uint16
	var curAtlas *ebiten.Image
	flush := func() {
		if len(inds) == 0 || curAtlas == nil {
			return
		}
		screen.DrawTriangles(verts, inds, curAtlas, &ebiten.DrawTrianglesOptions{Blend: blend, Filter: filter})
		verts = verts[:0]
		inds = inds[:0]
	}
	// 線形標本化のときだけ UV をハーフテクセル内側へ寄せ、共有アトラスの隣タイルを拾って
	// 境界に黒線が出るのを防ぐ。最近傍では拾う texel が矩形内に収まるので不要
	insetUV := filter == ebiten.FilterLinear
	for i := range quads {
		q := &quads[i]
		// アトラス切り替え、または uint16 の頂点インデックス上限を跨ぐ前に flush する。
		// 同一スプライトシートのタイルが大量に積まれても 65535 を超えると黙って描画化けするため
		if q.atlas != curAtlas || len(verts)+4 > maxVertsPerBatch {
			flush()
			curAtlas = q.atlas
		}
		sys.emitQuad(&verts, &inds, q, projector.Point, insetUV)
	}
	flush()
}

// emitMask は遮蔽マスクを焼く。深度順に、スプライト(occluder=false)は source-over で α を塗り重ね、
// 幾何(occluder=true)は destination-out で後ろの α を削る。手前の幾何だけがスプライトを切るので、
// 壁遮蔽は保ちつつ、スプライト同士は透明部分で後ろを消さない。emit と別なのは per-quad で blend が
// 変わり、種別の変わり目でもバッチを flush する点。焼いた α を sprBuf の切り抜きに使う。
func (sys *Render3DSystem) emitMask(screen *ebiten.Image, quads []r3quad, projector render3d.Projector) {
	sortQuadsByDepth(quads, projector.Depth)

	var verts []ebiten.Vertex
	var inds []uint16
	var curAtlas *ebiten.Image
	var curOccluder, started bool
	blendFor := func(occluder bool) ebiten.Blend {
		if occluder {
			return ebiten.BlendDestinationOut
		}
		return ebiten.BlendSourceOver
	}
	flush := func() {
		if len(inds) == 0 || curAtlas == nil {
			return
		}
		screen.DrawTriangles(verts, inds, curAtlas, &ebiten.DrawTrianglesOptions{Blend: blendFor(curOccluder), Filter: ebiten.FilterNearest})
		verts = verts[:0]
		inds = inds[:0]
	}
	for i := range quads {
		q := &quads[i]
		// アトラスまたは種別の変わり目、頂点上限の手前で flush する。種別が変わると blend も変わるため
		if !started || q.atlas != curAtlas || q.occluder != curOccluder || len(verts)+4 > maxVertsPerBatch {
			flush()
			curAtlas = q.atlas
			curOccluder = q.occluder
			started = true
		}
		sys.emitQuad(&verts, &inds, q, projector.Point, false)
	}
	flush()
}

// maxVertsPerBatch は1回の DrawTriangles へ積む頂点数の上限。インデックスが uint16 なので 65535 まで。
const maxVertsPerBatch = 65535

// emitQuad は1クアッドを三角形2枚として頂点バッファへ積む。画面外の頂点があれば捨てる。
// insetUV が真なら UV 矩形を半テクセル内側へ寄せ、線形標本化が隣タイルを拾う滲みを防ぐ。
func (sys *Render3DSystem) emitQuad(verts *[]ebiten.Vertex, inds *[]uint16, q *r3quad, project func(render3d.Vec) (consts.Coord[consts.ScreenPixel], bool), insetUV bool) {
	var sp [4]consts.Coord[consts.ScreenPixel]
	for k, p := range q.p {
		screenPos, ok := project(p)
		if !ok {
			return
		}
		sp[k] = screenPos
	}
	uv := insetUVRect(q.uv, insetUV)
	b := uint16(len(*verts))
	cr, cg, cb, ca := float32(q.col[0]), float32(q.col[1]), float32(q.col[2]), float32(q.alpha)
	for k := range 4 {
		*verts = append(*verts, ebiten.Vertex{
			DstX: float32(sp[k].X), DstY: float32(sp[k].Y),
			SrcX: float32(uv[k][0]), SrcY: float32(uv[k][1]),
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		})
	}
	*inds = append(*inds, b, b+1, b+2, b, b+2, b+3)
}

// insetUVRect は UV 矩形を各辺 0.5 テクセル内側へ寄せる。矩形が半テクセルより狭い、または inset が
// 偽ならそのまま返す。ハーフテクセル・インセットで、線形フィルタが矩形の外へはみ出さないようにする。
func insetUVRect(uv [4][2]float64, inset bool) [4][2]float64 {
	if !inset {
		return uv
	}
	minx := min(uv[0][0], uv[2][0])
	maxx := max(uv[0][0], uv[2][0])
	miny := min(uv[0][1], uv[2][1])
	maxy := max(uv[0][1], uv[2][1])
	const half = 0.5
	if maxx-minx <= 2*half || maxy-miny <= 2*half {
		return uv
	}
	out := uv
	for k := range 4 {
		switch out[k][0] {
		case minx:
			out[k][0] += half
		case maxx:
			out[k][0] -= half
		}
		switch out[k][1] {
		case miny:
			out[k][1] += half
		case maxy:
			out[k][1] -= half
		}
	}
	return out
}
