package systems

import (
	"cmp"
	"fmt"
	"image/color"
	"slices"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

var (
	wallShadowImage  *ebiten.Image // 壁が落とす影
	moverShadowImage *ebiten.Image // 動く物体が落とす影
	shadowImageOnce  sync.Once     // 影画像の初期化を1度だけにし、並列描画のデータレースを防ぐ
)

// 記憶タイルの退色パラメータ。彩度を大きく落として明度を少し下げ、記憶らしい見た目にする
const (
	memorySaturation = 0.18 // 記憶タイルの彩度。0で完全グレー
	memoryValue      = 0.9  // 記憶タイルの明度。わずかに暗くする
)

// spriteImageCacheKey はスプライト画像キャッシュのキー
// SpriteRenderには比較不能なフィールドが含まれていて直接使えないので定義する
type spriteImageCacheKey struct {
	SpriteSheetName string
	SpriteKey       string
}

// RenderSpriteSystem はスプライト描画システム
// キャッシュを保持し、描画処理を行う
type RenderSpriteSystem struct {
	spriteImageCache map[spriteImageCacheKey]*ebiten.Image
	// darknessMap は1タイル1テクセルの暗さマップ。FilterNearest でタイルサイズへ拡大し、タイル整列の暗闇を描く
	darknessMap *ebiten.Image
	// darknessPix は darknessMap へ書き込む画素バッファ。毎フレームの確保を避けて使い回す
	darknessPix []byte
}

// NewRenderSpriteSystem はRenderSpriteSystemを初期化する
func NewRenderSpriteSystem() *RenderSpriteSystem {
	return &RenderSpriteSystem{
		spriteImageCache: make(map[spriteImageCacheKey]*ebiten.Image),
	}
}

// setTranslate は真上から見下ろす2D描画の画像配置オプションをセットする。
// 取得済みカメラを受け取り、per-sprite/per-shadow のホットループでもカメラ取得を1フレーム1回に抑える
func setTranslate(world w.World, op *ebiten.DrawImageOptions, camera *gc.Camera) {
	cx, cy := float64(world.Resources.ScreenDimensions.Width/2), float64(world.Resources.ScreenDimensions.Height/2)

	// カメラ位置の設定
	if camera != nil {
		op.GeoM.Translate(-float64(camera.Pos.X), -float64(camera.Pos.Y))
		op.GeoM.Scale(camera.Scale, camera.Scale)
	}
	// 画面の中央
	op.GeoM.Translate(cx, cy)
}

// viewportTileBounds はカメラの可視範囲をタイル座標の矩形で返す。margin タイル分だけ外側に広げる。
// 画面外のタイル/スプライト描画をスキップするための可視カリングに使う。
func viewportTileBounds(world w.World, margin consts.Tile, camera *gc.Camera) (minX, maxX, minY, maxY int) {
	var cameraX, cameraY float64
	cameraScale := 1.0
	if camera != nil {
		cameraX, cameraY, cameraScale = float64(camera.Pos.X), float64(camera.Pos.Y), camera.Scale
	}
	if cameraScale <= 0 {
		cameraScale = 1.0
	}
	ts := int(consts.TileSize)
	m := int(margin)
	halfW := int(float64(world.Resources.ScreenDimensions.Width)/cameraScale) / 2
	halfH := int(float64(world.Resources.ScreenDimensions.Height)/cameraScale) / 2
	minX = (int(cameraX)-halfW)/ts - m
	maxX = (int(cameraX)+halfW)/ts + m
	minY = (int(cameraY)-halfH)/ts - m
	maxY = (int(cameraY)+halfH)/ts + m
	return minX, maxX, minY, maxY
}

// inViewport は指定タイルが可視範囲矩形内かを返す
func inViewport(grid *gc.GridElement, minX, maxX, minY, maxY int) bool {
	x, y := int(grid.X), int(grid.Y)
	return x >= minX && x <= maxX && y >= minY && y <= maxY
}

// viewportCullMargin は可視カリングの外側マージン。単位はタイル。
// スプライト/影が画面端を跨いでも欠けないよう余裕を持たせる
const viewportCullMargin consts.Tile = 2

// String はシステム名を返す
// w.Renderer interfaceを実装
func (sys RenderSpriteSystem) String() string {
	return "RenderSpriteSystem"
}

// Draw は (下) タイル -> 影 -> スプライト -> 暗闇 (上) の順に表示する。
// 暗闇を最後に重ねることで、床だけでなく影もスプライトも同じ暗さで減光する。
// w.Renderer interfaceを実装
func (sys *RenderSpriteSystem) Draw(world w.World, screen *ebiten.Image) error {
	// VisionSystemが計算した光源情報を取得する
	tileRenderMap := computeTileRenderMap(world, query.GetVisionState(world).LightSourceCache)

	initializeShadowImages()

	// カメラはフレーム内で不変。ここで1回だけ取得し各描画関数へ渡す。
	// 描画するスプライト/影の数だけフィルタ生成が走るのを防ぐ
	camera := query.GetPlayerCamera(world)

	if err := sys.renderFloorLayer(world, screen, tileRenderMap, camera); err != nil {
		return err
	}
	sys.renderShadows(world, screen, tileRenderMap, camera)
	if err := sys.renderObjectLayer(world, screen, tileRenderMap, camera); err != nil {
		return err
	}
	// 暗闇は最後に重ねる。床だけでなく影・スプライトも同じ暗さで沈み、光源から離れた
	// オブジェクトも暗くなる。per-tile の暗さを重ねるのでオブジェクトもタイル単位で減光する
	sys.renderDarkness(world, screen, tileRenderMap, camera)

	return nil
}

// initializeShadowImages は影画像を初期化する
func initializeShadowImages() {
	shadowImageOnce.Do(func() {
		wallWidth := int(consts.TileSize)
		wallHeight := int(consts.TileSize / 2)
		if wallWidth > 0 && wallHeight > 0 {
			wallShadowImage = ebiten.NewImage(wallWidth, wallHeight)
			wallShadowImage.Fill(color.RGBA{0, 0, 0, 80})
		}
		moverWidth := int(consts.TileSize - 6 - 2)
		moverHeight := int(consts.TileSize / 2)
		if moverWidth > 0 && moverHeight > 0 {
			moverShadowImage = ebiten.NewImage(moverWidth, moverHeight)
			moverShadowImage.Fill(color.RGBA{0, 0, 0, 120})
		}
	})
}

// renderFloorLayer は床レイヤー（タイル）を描画する
func (sys *RenderSpriteSystem) renderFloorLayer(world w.World, screen *ebiten.Image, tileRenderMap map[gc.GridElement]TileRenderInfo, camera *gc.Camera) error {
	iSprite := 0
	minX, maxX, minY, maxY := viewportTileBounds(world, viewportCullMargin, camera)
	// タイル総数を上限に確保する。viewport カリングで実際に詰めるのは一部だけ
	countQuery := query.ActiveFilter3[gc.SpriteRender, gc.GridElement, gc.Tile](world).Query()
	entities := make([]ecs.Entity, countQuery.Count())
	countQuery.Close()
	tileQuery := query.ActiveFilter3[gc.SpriteRender, gc.GridElement, gc.Tile](world).Query()
	for tileQuery.Next() {
		entity := tileQuery.Entity()
		// 画面外のタイルはソートも描画もしない
		if !inViewport(world.Components.GridElement.Get(entity), minX, maxX, minY, maxY) {
			continue
		}
		entities[iSprite] = entity
		iSprite++
	}

	slices.SortStableFunc(entities[:iSprite], func(a, b ecs.Entity) int {
		return cmp.Compare(world.Components.SpriteRender.Get(a).Depth, world.Components.SpriteRender.Get(b).Depth)
	})

	for i := range iSprite {
		entity := entities[i]
		gridElement := world.Components.GridElement.Get(entity)

		info, exists := tileRenderMap[*gridElement]
		if !exists {
			continue
		}
		// 記憶タイルの床は退色させて描く。今見えているタイルと区別して「記憶している」と
		// 分かる見た目にする。フルカラーのまま暗くすると平坦な暗い部屋に見えるのを避ける
		_, remembered := info.(TileRenderRemembered)

		spriteRender := world.Components.SpriteRender.Get(entity)
		pos := &gc.Position{Coord: consts.TileCenterToWorld(gridElement.Coord)}
		if err := sys.drawImage(world, screen, spriteRender, pos, 0, camera, remembered); err != nil {
			// エンティティ情報を追加してエラーを詳細化
			var entityInfo string
			if world.Components.Name.Has(entity) {
				name := world.Components.Name.Get(entity)
				entityInfo = fmt.Sprintf("Name: %s", name.Name)
			}
			return fmt.Errorf("entity %d at (%d,%d), SpriteSheet: '%s', SpriteKey: '%s', %s: %w",
				entity, gridElement.X, gridElement.Y, spriteRender.SpriteSheetName, spriteRender.SpriteKey, entityInfo, err)
		}
	}
	return nil
}

// renderObjectLayer はタイル以外のオブジェクトレイヤーを描画する
func (sys *RenderSpriteSystem) renderObjectLayer(world w.World, screen *ebiten.Image, tileRenderMap map[gc.GridElement]TileRenderInfo, camera *gc.Camera) error {
	var entities []ecs.Entity
	minX, maxX, minY, maxY := viewportTileBounds(world, viewportCullMargin, camera)

	// タイル以外のスプライトを収集する。フィールド上のオブジェクトとMoversを含む
	objectQuery := query.ActiveFilter2[gc.SpriteRender, gc.GridElement](world).Without(ecs.C[gc.Tile]()).Query()
	for objectQuery.Next() {
		entity := objectQuery.Entity()
		// 画面外は描画しない
		if !inViewport(world.Components.GridElement.Get(entity), minX, maxX, minY, maxY) {
			continue
		}
		entities = append(entities, entity)
	}

	slices.SortStableFunc(entities, func(a, b ecs.Entity) int {
		return cmp.Compare(world.Components.SpriteRender.Get(a).Depth, world.Components.SpriteRender.Get(b).Depth)
	})

	for _, entity := range entities {
		gridElement := world.Components.GridElement.Get(entity)

		if _, ok := tileRenderMap[*gridElement].(TileRenderVisible); !ok {
			continue
		}

		spriteRender := world.Components.SpriteRender.Get(entity)
		pos := &gc.Position{Coord: consts.TileCenterToWorld(gridElement.Coord)}
		if err := sys.drawImage(world, screen, spriteRender, pos, 0, camera, false); err != nil {
			return err
		}
	}
	return nil
}

// renderShadows は物体と壁の影を描画する
func (sys *RenderSpriteSystem) renderShadows(world w.World, screen *ebiten.Image, tileRenderMap map[gc.GridElement]TileRenderInfo, camera *gc.Camera) {
	minX, maxX, minY, maxY := viewportTileBounds(world, viewportCullMargin, camera)

	// 物体の影
	moverShadowQuery := query.ActiveFilter2[gc.SpriteRender, gc.GridElement](world).Query()
	for moverShadowQuery.Next() {
		entity := moverShadowQuery.Entity()
		// TurnBased または Fixed を持つエンティティのみ
		if !world.Components.TurnBased.Has(entity) && !world.Components.Fixed.Has(entity) {
			continue
		}

		spriteRender := world.Components.SpriteRender.Get(entity)

		// 高さのあるものだけが影を落とす
		if spriteRender.Depth <= gc.DepthNumRug {
			continue
		}

		gridElement := world.Components.GridElement.Get(entity)

		if !inViewport(gridElement, minX, maxX, minY, maxY) {
			continue
		}
		if _, ok := tileRenderMap[*gridElement].(TileRenderVisible); !ok {
			continue
		}

		// グリッド座標をタイル中心のピクセル座標に変換。X はスプライト幅ぶん左へずらす
		center := consts.TileCenterToWorld(gridElement.Coord)
		pixelX := float64(center.X) - 12
		pixelY := float64(center.Y)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pixelX, pixelY)
		setTranslate(world, op, camera)
		if moverShadowImage != nil {
			screen.DrawImage(moverShadowImage, op)
		}
	}

	// 下タイルが床の場合のみ壁の影。
	// 下タイル参照用のマップは viewport 内（+margin）だけ構築する。大マップで全タイルを
	// 毎フレーム map 化するのを避ける
	tileMap := make(map[gc.GridElement]ecs.Entity)
	// 下タイル参照は床タイルのみが対象。gc.Tile で絞りキャラ/固定物を走査から除く
	tileMapQuery := query.ActiveFilter3[gc.GridElement, gc.SpriteRender, gc.Tile](world).Query()
	for tileMapQuery.Next() {
		e := tileMapQuery.Entity()
		ge := world.Components.GridElement.Get(e)
		if !inViewport(ge, minX, maxX, minY, maxY) {
			continue
		}
		tileMap[*ge] = e
	}

	wallShadowQuery := query.ActiveFilter4[gc.SpriteRender, gc.GridElement, gc.BlockView, gc.BlockPass](world).Query()
	for wallShadowQuery.Next() {
		entity := wallShadowQuery.Entity()
		grid := world.Components.GridElement.Get(entity)

		if !inViewport(grid, minX, maxX, minY, maxY) {
			continue
		}
		if _, ok := tileRenderMap[*grid].(TileRenderVisible); !ok {
			continue
		}

		spriteRender := world.Components.SpriteRender.Get(entity)

		// 高さのあるものだけが影を落とす
		if spriteRender.Depth <= gc.DepthNumRug {
			continue
		}

		// 下のタイルを検索
		belowPos := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: grid.X, Y: grid.Y + 1}}
		belowTileEntity, foundBelow := tileMap[belowPos]

		if !foundBelow {
			continue
		}

		if !world.Components.SpriteRender.Has(belowTileEntity) {
			continue // 下が床でなければ影を描画しない
		}
		belowSpriteRender := world.Components.SpriteRender.Get(belowTileEntity)
		if belowSpriteRender.Depth != gc.DepthNumFloor {
			continue // 下が床でなければ影を描画しない
		}

		// 下のタイルが壁でないことも確認（壁->床->壁の場合は影を描画しない）
		if world.Components.BlockView.Has(belowTileEntity) && world.Components.BlockPass.Has(belowTileEntity) {
			continue
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(int(grid.X)*int(consts.TileSize)), float64(int(grid.Y)*int(consts.TileSize)+int(consts.TileSize)))
		setTranslate(world, op, camera)
		if wallShadowImage != nil {
			screen.DrawImage(wallShadowImage, op)
		}
	}
}

func (sys *RenderSpriteSystem) getImage(world w.World, spriteRender *gc.SpriteRender) (*ebiten.Image, error) {
	key := spriteImageCacheKey{
		SpriteSheetName: spriteRender.SpriteSheetName,
		SpriteKey:       spriteRender.SpriteKey,
	}
	if v, ok := sys.spriteImageCache[key]; ok {
		return v, nil
	}
	// 解決は resources.SpriteImage に集約する。ここは毎フレームのホットパスなので結果をキャッシュする
	img, err := resources.SpriteImage(world.Resources.SpriteSheets, spriteRender)
	if err != nil {
		return nil, err
	}
	sys.spriteImageCache[key] = img
	return img, nil
}

func (sys *RenderSpriteSystem) drawImage(world w.World, screen *ebiten.Image, spriteRender *gc.SpriteRender, pos *gc.Position, angle float64, camera *gc.Camera, desaturate bool) error {
	// Resourcesからスプライトシートを取得
	if world.Resources.SpriteSheets == nil {
		return fmt.Errorf("sprite sheets are nil")
	}
	spriteSheet, exists := world.Resources.SpriteSheets[spriteRender.SpriteSheetName]
	if !exists {
		return fmt.Errorf("sprite sheet '%s' not found", spriteRender.SpriteSheetName)
	}

	sprite, exists := spriteSheet.Sprites[spriteRender.SpriteKey]
	if !exists {
		return fmt.Errorf("sprite key '%s' does not exist in sprite sheet '%s'", spriteRender.SpriteKey, spriteRender.SpriteSheetName)
	}

	op := &spriteRender.Options
	op.GeoM.Reset()                                                       // FIXME: Resetがないと非表示になる。なぜ?
	op.GeoM.Translate(float64(-sprite.Width/2), float64(-sprite.Width/2)) // 回転軸を画像の中心にする
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	setTranslate(world, op, camera)

	img, err := sys.getImage(world, spriteRender)
	if err != nil {
		return err
	}
	if desaturate {
		// 記憶タイルは彩度を落として退色させる。色行列で彩度を下げ明度を少し落とし、
		// 「今見ている」タイルと区別する。減光は後段の暗闇オーバーレイが担う
		var cm colorm.ColorM
		cm.ChangeHSV(0, memorySaturation, memoryValue)
		dop := &colorm.DrawImageOptions{GeoM: op.GeoM, Blend: op.Blend, Filter: op.Filter}
		colorm.DrawImage(screen, img, cm, dop)
	} else {
		screen.DrawImage(img, op)
	}

	if world.Resources.Config.ShowMapDebug {
		// デバッグ用：スプライト番号表示(dirt, dwall)
		if spriteRender.SpriteSheetName == "tile" {
			var number string
			var prefix string
			if after, ok := strings.CutPrefix(spriteRender.SpriteKey, "dirt_"); ok {
				number = after
				prefix = "d"
			} else if after, ok := strings.CutPrefix(spriteRender.SpriteKey, "dwall_"); ok {
				number = after
				prefix = "w"
			}

			if number != "" {
				// カメラ変換を考慮したテキスト位置を計算
				textOp := &ebiten.DrawImageOptions{}
				textOp.GeoM.Translate(float64(pos.X-8), float64(pos.Y-8)) // タイルの左上付近に表示
				setTranslate(world, textOp, camera)

				// テキスト表示位置を逆変換で求める
				screenX, screenY := textOp.GeoM.Apply(0, 0)
				ebitenutil.DebugPrintAt(screen, prefix+number, int(screenX), int(screenY))
			}
		}
	}

	return nil
}

// renderDarkness は per-tile の暗さをタイル整列でそのまま描く暗闇オーバーレイ。
// vision が壁遮蔽込みで計算した per-tile の暗さを1タイル1テクセルの小画像へ詰め、
// FilterNearest でタイルサイズへ拡大する。補間しないのでタイル単位の一様な明るさになり、
// ドット絵と質感が揃う。暗さは連続値なので段差ジャンプ(チカチカ)は出ない。
func (sys *RenderSpriteSystem) renderDarkness(world w.World, screen *ebiten.Image, tileRenderMap map[gc.GridElement]TileRenderInfo, camera *gc.Camera) {
	var cameraX, cameraY float64
	cameraScale := 1.0
	if camera != nil {
		cameraX = float64(camera.Pos.X)
		cameraY = float64(camera.Pos.Y)
		cameraScale = camera.Scale
	}

	// 床レイヤのカリング余白に合わせてタイルを拾う。描き漏れを防ぐ
	minX, maxX, minY, maxY := viewportTileBounds(world, viewportCullMargin, camera)
	mw := maxX - minX + 1
	mh := maxY - minY + 1
	if mw <= 0 || mh <= 0 {
		return
	}
	sys.ensureDarknessMap(mw, mh)

	// 1タイル1テクセル。黒 rgb=0、アルファに暗さを連続値で詰める。バッファは使い回す。
	// rgb は常に0のままで、各テクセルのアルファは毎フレーム全て上書きするのでクリア不要。
	// 3状態を tileRenderAt で網羅する。未探索は完全な闇、可視/記憶はそれぞれの暗さ
	pix := sys.darknessPix
	for j := range mh {
		for i := range mw {
			grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(minX + i), Y: consts.Tile(minY + j)}}
			var darkness float64
			switch v := tileRenderAt(tileRenderMap, grid).(type) {
			case TileRenderVisible:
				darkness = float64(v.Darkness)
			case TileRenderRemembered:
				darkness = float64(v.Darkness)
			case TileRenderUnexplored:
				darkness = 1.0
			default:
				panic(fmt.Sprintf("unknown TileRenderInfo: %T", v))
			}
			pix[(j*mw+i)*4+3] = byte(max(0.0, min(1.0, darkness)) * 255)
		}
	}
	sys.darknessMap.WritePixels(pix)

	// タイルグリッドへ合わせて拡大する。texel(0,0) の左上をタイル(minX,minY)の左上へ置く。
	// FilterNearest なので各テクセルは1タイルの一様な四角として描かれる
	ts := float64(consts.TileSize)
	offsetX := (float64(minX)*ts-cameraX)*cameraScale + float64(world.Resources.ScreenDimensions.Width)/2
	offsetY := (float64(minY)*ts-cameraY)*cameraScale + float64(world.Resources.ScreenDimensions.Height)/2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(ts*cameraScale, ts*cameraScale)
	op.GeoM.Translate(offsetX, offsetY)
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(sys.darknessMap, op)
}

// ensureDarknessMap は per-tile 暗さマップの小画像を用意する。サイズが変わったら作り直す。
func (sys *RenderSpriteSystem) ensureDarknessMap(width, height int) {
	if sys.darknessMap != nil {
		b := sys.darknessMap.Bounds()
		if b.Dx() == width && b.Dy() == height {
			return
		}
		sys.darknessMap.Deallocate()
	}
	sys.darknessMap = ebiten.NewImage(width, height)
	sys.darknessPix = make([]byte, width*height*4)
}
