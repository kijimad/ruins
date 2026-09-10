package systems

import (
	"image"
	"image/color"
	"math"
	"slices"
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
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

// Draw は w.Renderer を満たす。3Dシーンを screen へ描く。
func (sys *Render3DSystem) Draw(world w.World, screen *ebiten.Image) error {
	quads, projector, err := sys.buildScene(world)
	if err != nil {
		return err
	}
	sys.emit(screen, quads, projector)
	return nil
}

// itemMarkerProximity はマーカーを出すプレイヤーからの近接距離。2マス以内。
// 少し離れていても、漁る価値のある升が見えるようにする。
const itemMarkerProximity = 2

// collectItemMarkers はプレイヤー近接かつ視界内の升に、開ける前には見えない中身があることを
// 示す記号マーカーのクアッドを quads へ足す。対象は「中身のある収納」か「拾えるアイテムが2個以上
// 重なった升」。単品で見えているアイテムは自前スプライトで分かるので出さない。重なって隠れた分だけを指す。
// マーカーは 2D オーバーレイでなくシーンのクアッドとして積み、emit の深度ソートで手前の壁に隠させる。
func (sys *Render3DSystem) collectItemMarkers(world w.World, quads []r3quad, projector render3d.Projector) []r3quad {
	player, err := query.GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return quads
	}
	pc := world.Components.GridElement.Get(player).Coord
	near := func(c consts.Coord[consts.Tile]) bool {
		return geometry.ChebyshevDistance(pc, c) <= itemMarkerProximity
	}

	// 拾えるフィールドアイテムを升ごとに数える。近接分だけでよい。
	// Fixed でないフィールド物が拾える物なので、フィルタで Fixed を除く
	itemCount := map[consts.Coord[consts.Tile]]int{}
	itemQuery := query.ActiveFilter2[gc.LocationOnField, gc.GridElement](world).Without(ecs.C[gc.Fixed]()).Query()
	for itemQuery.Next() {
		e := itemQuery.Entity()
		c := world.Components.GridElement.Get(e).Coord
		if near(c) {
			itemCount[c]++
		}
	}

	// 中身のある収納の升を集める
	storageHas := map[consts.Coord[consts.Tile]]bool{}
	stQuery := query.ActiveFilter2[gc.Interactable, gc.GridElement](world).Query()
	for stQuery.Next() {
		e := stQuery.Entity()
		c := world.Components.GridElement.Get(e).Coord
		if !near(c) {
			continue
		}
		if !slices.Contains(world.Components.Interactable.Get(e).Interactions, gc.InteractionStorage) {
			continue
		}
		if query.HasStorageItems(world, e) {
			storageHas[c] = true
		}
	}

	// マーカー画像は実際に1枚でも要るときだけ焼く。近くに対象が無い升配置では
	// フォントリソースに触れずに済み、リソースを持たない最小 world でも通る
	var img *ebiten.Image
	var iw, ih int
	for dy := -itemMarkerProximity; dy <= itemMarkerProximity; dy++ {
		for dx := -itemMarkerProximity; dx <= itemMarkerProximity; dx++ {
			c := consts.Coord[consts.Tile]{X: pc.X + consts.Tile(dx), Y: pc.Y + consts.Tile(dy)}
			if !storageHas[c] && itemCount[c] < 2 {
				continue
			}
			if !query.IsInVision(world, pc, c) {
				continue
			}
			if img == nil {
				img = itemMarkerImage(world.Resources.UIResources.Text.SplashFontFace)
				iw, ih = img.Bounds().Dx(), img.Bounds().Dy()
				if iw == 0 || ih == 0 {
					return quads
				}
			}
			quads = sys.appendMarkerQuad(quads, projector, c, img, iw, ih)
		}
	}
	return quads
}

// itemMarkerGlyph はマーカーに出す汎用記号。中身や重なりがあることの合図。
const itemMarkerGlyph = "!"

// itemMarkerHeightRatio はマーカーの高さをビルボード高の何割にするか。ズームや奥行きで
// ビルボードが伸縮しても比率を保ち、常に同じ大きさに見せる。
const itemMarkerHeightRatio = 0.65

var (
	itemMarkerImg     *ebiten.Image
	itemMarkerImgOnce sync.Once
)

// itemMarkerImage は縁取り済みの記号を1枚の画像へ焼いて返す。text 描画は共有グリフキャッシュを
// 触るので一度だけ行い、以降はこの画像を拡大して重ねる。拡大縮小しても縁取りごと比率が保たれる。
func itemMarkerImage(face text.Face) *ebiten.Image {
	itemMarkerImgOnce.Do(func() {
		gw, gh := uicore.MeasureText(itemMarkerGlyph, face)
		// 縁取りを太めに敷いて、細い記号でも太く見えるようにする
		const outline = 3
		const pad = outline + 1
		// 緑のアイテムとも黒縁で分離される蛍光緑にして、どの升にマーカーが付くかを目立たせる
		markerColor := color.RGBA{R: 60, G: 255, B: 90, A: 255}
		img := ebiten.NewImage(gw+pad*2, gh+pad*2)
		cv := uicore.NewEbitenCanvas(img)
		// 暗色を周囲 outline 半径へずらして重ね、太い縁取りにする。中央に本体を重ねる
		for dy := -outline; dy <= outline; dy++ {
			for dx := -outline; dx <= outline; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				cv.DrawText(image.Pt(pad+dx, pad+dy), itemMarkerGlyph, face, theme.HUDTextOutline)
			}
		}
		cv.DrawText(image.Pt(pad, pad), itemMarkerGlyph, face, markerColor)
		itemMarkerImg = img
	})
	return itemMarkerImg
}

// appendMarkerQuad はビルボード右上隅に張り付く記号マーカーのクアッドを1枚 quads へ足す。
// ワールド空間の板としてカメラ正面へ立てるので、emit の深度ソートで手前の壁に隠れる。
// 高さはビルボード高に比例させ、ズームや奥行きでも見かけの比率を一定に保つ。
func (sys *Render3DSystem) appendMarkerQuad(quads []r3quad, projector render3d.Projector, c consts.Coord[consts.Tile], img *ebiten.Image, iw, ih int) []r3quad {
	// collectBillboards と同じ幾何でビルボード右上隅の world 座標を組む
	const bw = 0.45
	right := projector.Right()
	up := render3d.At(0, 1, 0)
	corner := render3d.Add(
		render3d.Add(render3d.At(float64(c.X)+0.5, 0, float64(c.Y)+0.5), render3d.Scale(right, bw)),
		render3d.At(0, render3d.BillboardHeight, 0),
	)
	// 右上隅を基準に、左と下へ板を広げる。高さはビルボード高の比率、幅は画像縦横比で決める
	fh := render3d.BillboardHeight * itemMarkerHeightRatio
	fw := fh * float64(iw) / float64(ih)
	tr := corner
	tl := render3d.Add(corner, render3d.Scale(right, -fw))
	br := render3d.Add(corner, render3d.Scale(up, -fh))
	bl := render3d.Add(tl, render3d.Scale(up, -fh))
	// tint は満照。視界内の升にしか出さないので減光せず浮かせる
	sys.addQuad(&quads, tl, tr, br, bl, img, 0, 0, float64(iw), float64(ih), [3]float64{1, 1, 1})
	return quads
}

// buildScene は投影とクアッド列を組み立てる。Draw の幾何を1箇所に集約する。
func (sys *Render3DSystem) buildScene(world w.World) ([]r3quad, render3d.Projector, error) {
	projector, err := render3d.WorldProjector(world)
	if err != nil {
		return nil, render3d.Projector{}, err
	}
	center, err := render3d.PlayerTile(world)
	if err != nil {
		return nil, render3d.Projector{}, err
	}
	pcx, pcz := float64(center.X), float64(center.Y)

	visTint := sys.visTintFunc(world)
	quads := sys.collectTiles(world, pcx, pcz, visTint)
	quads = sys.collectBillboards(world, quads, pcx, pcz, projector.Right(), visTint)
	// 隠れた中身を示すマーカーもクアッドとして積み、深度ソートで手前の壁に隠させる
	quads = sys.collectItemMarkers(world, quads, projector)
	return quads, projector, nil
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
