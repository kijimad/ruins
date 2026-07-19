package systems

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

var (
	wallShadowImage  *ebiten.Image // 壁が落とす影
	moverShadowImage *ebiten.Image // 動く物体が落とす影
)

// spriteImageCacheKey はスプライト画像キャッシュのキー
// SpriteRenderには比較不能なフィールドが含まれていて直接使えないので定義する
type spriteImageCacheKey struct {
	SpriteSheetName string
	SpriteKey       string
}

// coloredDarknessCacheKey は光源色ごとの暗闇画像のキャッシュキー
type coloredDarknessCacheKey struct {
	R             uint8
	G             uint8
	B             uint8
	DarknessLevel int
}

// RenderSpriteSystem はスプライト描画システム
// キャッシュを保持し、描画処理を行う
type RenderSpriteSystem struct {
	spriteImageCache     map[spriteImageCacheKey]*ebiten.Image
	darknessCacheImages  []*ebiten.Image
	coloredDarknessCache map[coloredDarknessCacheKey]*ebiten.Image
}

// NewRenderSpriteSystem はRenderSpriteSystemを初期化する
func NewRenderSpriteSystem() *RenderSpriteSystem {
	return &RenderSpriteSystem{
		spriteImageCache:     make(map[spriteImageCacheKey]*ebiten.Image),
		coloredDarknessCache: make(map[coloredDarknessCacheKey]*ebiten.Image),
	}
}

// SetTranslate はカメラを考慮した画像配置オプションをセットする。
// 単発描画向けの公開エントリで、内部でカメラを取得する。
// 描画ループ内で繰り返し呼ぶ場合は取得済みカメラを渡す setTranslate を使う
func SetTranslate(world w.World, op *ebiten.DrawImageOptions) {
	setTranslate(world, op, getCamera(world))
}

// setTranslate は取得済みカメラを使って画像配置オプションをセットする。
// per-sprite/per-shadow のホットループから呼ばれ、カメラ取得を1フレーム1回に抑える
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

// Draw は (下) タイル -> 暗闇 -> 影 -> スプライト (上) の順に表示する
// w.Renderer interfaceを実装
func (sys *RenderSpriteSystem) Draw(world w.World, screen *ebiten.Image) error {
	// VisionSystemが計算した光源情報を取得する
	tileRenderMap := computeTileRenderMap(world, query.GetDungeon(world).LightSourceCache)

	initializeShadowImages()

	// カメラはフレーム内で不変。ここで1回だけ取得し各描画関数へ渡す。
	// 描画するスプライト/影の数だけフィルタ生成が走るのを防ぐ
	camera := getCamera(world)

	if err := sys.renderFloorLayer(world, screen, tileRenderMap, camera); err != nil {
		return err
	}
	sys.renderDarkness(world, screen, tileRenderMap, camera)
	sys.renderShadows(world, screen, tileRenderMap, camera)
	if err := sys.renderObjectLayer(world, screen, tileRenderMap, camera); err != nil {
		return err
	}

	return nil
}

// initializeShadowImages は影画像を初期化する
func initializeShadowImages() {
	if wallShadowImage == nil {
		wallWidth := int(consts.TileSize)
		wallHeight := int(consts.TileSize / 2)
		if wallWidth > 0 && wallHeight > 0 {
			wallShadowImage = ebiten.NewImage(wallWidth, wallHeight)
			wallShadowImage.Fill(color.RGBA{0, 0, 0, 80})
		}
	}
	if moverShadowImage == nil {
		moverWidth := int(consts.TileSize - 6 - 2)
		moverHeight := int(consts.TileSize / 2)
		if moverWidth > 0 && moverHeight > 0 {
			moverShadowImage = ebiten.NewImage(moverWidth, moverHeight)
			moverShadowImage.Fill(color.RGBA{0, 0, 0, 120})
		}
	}
}

// renderFloorLayer は床レイヤー（タイル）を描画する
func (sys *RenderSpriteSystem) renderFloorLayer(world w.World, screen *ebiten.Image, tileRenderMap map[gc.GridElement]TileRenderInfo, camera *gc.Camera) error {
	iSprite := 0
	minX, maxX, minY, maxY := viewportTileBounds(world, viewportCullMargin, camera)
	// タイル総数を上限に確保する。viewport カリングで実際に詰めるのは一部だけ
	countQuery := ecs.NewFilter3[gc.SpriteRender, gc.GridElement, gc.Tile](world.ECS).Query()
	entities := make([]ecs.Entity, countQuery.Count())
	countQuery.Close()
	tileQuery := ecs.NewFilter3[gc.SpriteRender, gc.GridElement, gc.Tile](world.ECS).Query()
	for tileQuery.Next() {
		entity := tileQuery.Entity()
		// 画面外のタイルはソートも描画もしない
		if !inViewport(world.Components.GridElement.Get(entity), minX, maxX, minY, maxY) {
			continue
		}
		entities[iSprite] = entity
		iSprite++
	}

	sort.Slice(entities[:iSprite], func(i, j int) bool {
		spriteRender1 := world.Components.SpriteRender.Get(entities[i])
		spriteRender2 := world.Components.SpriteRender.Get(entities[j])
		return spriteRender1.Depth < spriteRender2.Depth
	})

	for i := range iSprite {
		entity := entities[i]
		gridElement := world.Components.GridElement.Get(entity)

		_, exists := tileRenderMap[*gridElement]
		if !exists {
			continue
		}

		spriteRender := world.Components.SpriteRender.Get(entity)
		pos := &gc.Position{Coord: consts.Coord[consts.Pixel]{X: consts.Pixel(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize/2)), Y: consts.Pixel(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize/2))}}
		if err := sys.drawImage(world, screen, spriteRender, pos, 0, camera); err != nil {
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
	objectQuery := ecs.NewFilter2[gc.SpriteRender, gc.GridElement](world.ECS).
		Without(ecs.C[gc.Tile]()).Query()
	for objectQuery.Next() {
		entity := objectQuery.Entity()
		// 画面外は描画しない
		if !inViewport(world.Components.GridElement.Get(entity), minX, maxX, minY, maxY) {
			continue
		}
		entities = append(entities, entity)
	}

	sort.Slice(entities, func(i, j int) bool {
		spriteRender1 := world.Components.SpriteRender.Get(entities[i])
		spriteRender2 := world.Components.SpriteRender.Get(entities[j])
		return spriteRender1.Depth < spriteRender2.Depth
	})

	for _, entity := range entities {
		gridElement := world.Components.GridElement.Get(entity)

		if _, ok := tileRenderMap[*gridElement].(TileRenderVisible); !ok {
			continue
		}

		spriteRender := world.Components.SpriteRender.Get(entity)
		pos := &gc.Position{Coord: consts.Coord[consts.Pixel]{X: consts.Pixel(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2), Y: consts.Pixel(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)}}
		if err := sys.drawImage(world, screen, spriteRender, pos, 0, camera); err != nil {
			return err
		}
	}
	return nil
}

// renderShadows は物体と壁の影を描画する
func (sys *RenderSpriteSystem) renderShadows(world w.World, screen *ebiten.Image, tileRenderMap map[gc.GridElement]TileRenderInfo, camera *gc.Camera) {
	minX, maxX, minY, maxY := viewportTileBounds(world, viewportCullMargin, camera)

	// 物体の影
	moverShadowQuery := ecs.NewFilter2[gc.SpriteRender, gc.GridElement](world.ECS).Query()
	for moverShadowQuery.Next() {
		entity := moverShadowQuery.Entity()
		// TurnBased または Prop を持つエンティティのみ
		if !world.Components.TurnBased.Has(entity) && !world.Components.Prop.Has(entity) {
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

		// グリッド座標をピクセル座標に変換
		pixelX := float64(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2 - 12)
		pixelY := float64(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)

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
	// 下タイル参照は床タイルのみが対象。gc.Tile で絞りキャラ/Prop を走査から除く
	tileMapQuery := ecs.NewFilter3[gc.GridElement, gc.SpriteRender, gc.Tile](world.ECS).Query()
	for tileMapQuery.Next() {
		e := tileMapQuery.Entity()
		ge := world.Components.GridElement.Get(e)
		if !inViewport(ge, minX, maxX, minY, maxY) {
			continue
		}
		tileMap[*ge] = e
	}

	wallShadowQuery := ecs.NewFilter4[gc.SpriteRender, gc.GridElement, gc.BlockView, gc.BlockPass](world.ECS).Query()
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
	var result *ebiten.Image
	key := spriteImageCacheKey{
		SpriteSheetName: spriteRender.SpriteSheetName,
		SpriteKey:       spriteRender.SpriteKey,
	}
	if v, ok := sys.spriteImageCache[key]; ok {
		result = v
	} else {
		// Resourcesからスプライトシートを取得
		if world.Resources.SpriteSheets == nil {
			return nil, fmt.Errorf("SpriteSheets が nil です")
		}
		spriteSheet, exists := world.Resources.SpriteSheets[spriteRender.SpriteSheetName]
		if !exists {
			return nil, fmt.Errorf("スプライトシート '%s' が見つかりません", spriteRender.SpriteSheetName)
		}

		// スプライトキーからスプライトを取得
		sprite, exists := spriteSheet.Sprites[spriteRender.SpriteKey]
		if !exists {
			return nil, fmt.Errorf("スプライトキー '%s' がスプライトシート '%s' に存在しません", spriteRender.SpriteKey, spriteRender.SpriteSheetName)
		}

		texture := spriteSheet.Texture
		textureWidth := texture.Image.Bounds().Dx()
		textureHeight := texture.Image.Bounds().Dy()

		left := max(0, sprite.X)
		right := min(textureWidth, sprite.X+sprite.Width)
		top := max(0, sprite.Y)
		bottom := min(textureHeight, sprite.Y+sprite.Height)

		result = gc.SubImage(texture.Image, image.Rect(left, top, right, bottom))
		sys.spriteImageCache[key] = result
	}

	return result, nil
}

func (sys *RenderSpriteSystem) drawImage(world w.World, screen *ebiten.Image, spriteRender *gc.SpriteRender, pos *gc.Position, angle float64, camera *gc.Camera) error {
	// Resourcesからスプライトシートを取得
	if world.Resources.SpriteSheets == nil {
		return fmt.Errorf("SpriteSheets が nil です")
	}
	spriteSheet, exists := world.Resources.SpriteSheets[spriteRender.SpriteSheetName]
	if !exists {
		return fmt.Errorf("スプライトシート '%s' が見つかりません", spriteRender.SpriteSheetName)
	}

	sprite, exists := spriteSheet.Sprites[spriteRender.SpriteKey]
	if !exists {
		return fmt.Errorf("スプライトキー '%s' がスプライトシート '%s' に存在しません", spriteRender.SpriteKey, spriteRender.SpriteSheetName)
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
	screen.DrawImage(img, op)

	if world.Config.ShowMapDebug {
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

// DarknessLevels は暗闇の段階数を定義する。少ない段階数のほうが見た目が自然になる
const DarknessLevels = 4

// renderDarkness はタイルごとの暗闇オーバーレイを描画する
func (sys *RenderSpriteSystem) renderDarkness(world w.World, screen *ebiten.Image, tileRenderMap map[gc.GridElement]TileRenderInfo, camera *gc.Camera) {
	var cameraX, cameraY float64
	cameraScale := 1.0
	if camera != nil {
		cameraX = float64(camera.Pos.X)
		cameraY = float64(camera.Pos.Y)
		cameraScale = camera.Scale
	}

	if len(sys.darknessCacheImages) == 0 {
		sys.initializeDarknessCache(int(consts.TileSize))
	}

	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height
	// 暗闇は可視範囲のタイルだけに描く。境界は viewportTileBounds に集約する
	startTileX, endTileX, startTileY, endTileY := viewportTileBounds(world, 1, camera)

	for tileX := startTileX; tileX <= endTileX; tileX++ {
		for tileY := startTileY; tileY <= endTileY; tileY++ {
			grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(tileX), Y: consts.Tile(tileY)}}

			var darkness float64
			var lightColor color.RGBA
			info, exists := tileRenderMap[grid]
			if !exists {
				// tileRenderMapにないタイルは完全に黒くする。
				// マップ外・未探索タイルの両方が該当する
				darkness = 1.0
			} else {
				switch v := info.(type) {
				case TileRenderVisible:
					darkness = float64(v.Darkness)
					lightColor = v.LightColor
				case TileRenderRemembered:
					darkness = float64(v.Darkness)
				}
			}

			worldX := float64(tileX * int(consts.TileSize))
			worldY := float64(tileY * int(consts.TileSize))
			screenX := (worldX-cameraX)*cameraScale + float64(screenWidth)/2
			screenY := (worldY-cameraY)*cameraScale + float64(screenHeight)/2
			sys.drawDarknessAtLevelWithColor(screen, screenX, screenY, darkness, lightColor, cameraScale, int(consts.TileSize))
		}
	}
}

// initializeDarknessCache は段階的暗闇用の画像キャッシュを初期化する
func (sys *RenderSpriteSystem) initializeDarknessCache(tileSize int) {
	if tileSize <= 0 {
		return
	}

	sys.darknessCacheImages = make([]*ebiten.Image, DarknessLevels+1)
	sys.darknessCacheImages[0] = nil // 0: 暗闇なし

	for i := 1; i <= DarknessLevels; i++ {
		darkness := float64(i) / float64(DarknessLevels)
		alpha := uint8(darkness * 255)

		sys.darknessCacheImages[i] = ebiten.NewImage(tileSize, tileSize)
		sys.darknessCacheImages[i].Fill(color.RGBA{0, 0, 0, alpha})
	}
}

// drawDarknessAtLevelWithColor は光源の色を考慮した暗闇を描画する
func (sys *RenderSpriteSystem) drawDarknessAtLevelWithColor(screen *ebiten.Image, x, y, darkness float64, lightColor color.RGBA, scale float64, tileSize int) {
	if darkness <= 0.0 {
		return
	}

	darknessLevel := max(min(int(math.Ceil(darkness*float64(DarknessLevels))), DarknessLevels), 1)

	quantizedDarkness := float64(darknessLevel) / float64(DarknessLevels)

	cacheKey := coloredDarknessCacheKey{
		R:             lightColor.R,
		G:             lightColor.G,
		B:             lightColor.B,
		DarknessLevel: darknessLevel,
	}

	darknessImg, exists := sys.coloredDarknessCache[cacheKey]
	if !exists {
		alpha := uint8(quantizedDarkness * 255)

		colorStrength := 0.1
		darknessColor := color.RGBA{
			R: uint8(float64(lightColor.R) * colorStrength),
			G: uint8(float64(lightColor.G) * colorStrength),
			B: uint8(float64(lightColor.B) * colorStrength),
			A: alpha,
		}

		darknessImg = ebiten.NewImage(tileSize, tileSize)
		darknessImg.Fill(darknessColor)

		if len(sys.coloredDarknessCache) < 1000 {
			sys.coloredDarknessCache[cacheKey] = darknessImg
		}
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	screen.DrawImage(darknessImg, op)
}
