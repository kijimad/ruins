package systems

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/maptemplate"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/consts"
	ecs "github.com/x-hgg-x/goecs/v2"
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

// RenderSpriteSystem はスプライト描画システム
// キャッシュを保持し、描画処理を行う
type RenderSpriteSystem struct {
	spriteImageCache map[spriteImageCacheKey]*ebiten.Image
}

// NewRenderSpriteSystem はRenderSpriteSystemを初期化する
func NewRenderSpriteSystem() *RenderSpriteSystem {
	return &RenderSpriteSystem{
		spriteImageCache: make(map[spriteImageCacheKey]*ebiten.Image),
	}
}

// SetTranslate はカメラを考慮した画像配置オプションをセットする
// TODO: ズーム率を追加する
func SetTranslate(world w.World, op *ebiten.DrawImageOptions) {
	var camera *gc.Camera
	var cPos *gc.Position
	var cGridElement *gc.GridElement

	// ピクセル座標のカメラを先に確認
	world.Manager.Join(
		world.Components.Camera,
		world.Components.Position,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		camera = world.Components.Camera.Get(entity).(*gc.Camera)
		cPos = world.Components.Position.Get(entity).(*gc.Position)
	}))

	// グリッド座標のカメラを確認
	world.Manager.Join(
		world.Components.Camera,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		camera = world.Components.Camera.Get(entity).(*gc.Camera)
		cGridElement = world.Components.GridElement.Get(entity).(*gc.GridElement)
	}))

	cx, cy := float64(world.Resources.ScreenDimensions.Width/2), float64(world.Resources.ScreenDimensions.Height/2)

	// カメラ位置の設定
	if cGridElement != nil {
		// グリッド座標をピクセル座標に変換
		tilePixelX := float64(int(cGridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2)
		tilePixelY := float64(int(cGridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)
		op.GeoM.Translate(-tilePixelX, -tilePixelY)
	} else if cPos != nil {
		op.GeoM.Translate(float64(-cPos.X), float64(-cPos.Y))
	}

	if camera != nil {
		op.GeoM.Scale(camera.Scale, camera.Scale)
	}
	// 画面の中央
	op.GeoM.Translate(float64(cx), float64(cy))
}

// String はシステム名を返す
// w.Renderer interfaceを実装
func (sys RenderSpriteSystem) String() string {
	return "RenderSpriteSystem"
}

// Draw は (下) タイル -> 暗闇 -> 光源グロー -> 影 -> スプライト (上) の順に表示する
// w.Renderer interfaceを実装
func (sys *RenderSpriteSystem) Draw(world w.World, screen *ebiten.Image) error {
	// 現在の視界データを取得（全描画で使用）
	visibilityData := GetCurrentVisibilityData()

	// シャドウ画像を初期化
	initializeShadowImages()

	// 各種描画処理
	if err := sys.renderFloorLayer(world, screen, visibilityData); err != nil { // 床レイヤー（タイル・アイテム）を描画
		return err
	}
	renderDistanceBasedDarkness(world, screen, visibilityData)               // 床タイルに暗闇オーバーレイ
	sys.renderShadows(world, screen, visibilityData)                         // 影を描画
	if err := sys.renderSprites(world, screen, visibilityData); err != nil { // 物体を描画
		return err
	}

	cfg := config.Get()
	if cfg.ShowMapDebug {
		// 橋エンティティの位置を色付きで表示
		sys.renderBridgeDebug(world, screen, visibilityData)
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

// renderFloorLayer は床レイヤー（タイル・地面に落ちているアイテム）を描画する
func (sys *RenderSpriteSystem) renderFloorLayer(world w.World, screen *ebiten.Image, visibilityData map[string]TileVisibility) error {
	iSprite := 0
	entities := make([]ecs.Entity, world.Manager.Join(world.Components.SpriteRender, world.Components.GridElement).Size())
	world.Manager.Join(
		world.Components.SpriteRender,
		world.Components.GridElement,
		world.Components.Prop.Not(),
		world.Components.TurnBased.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entities[iSprite] = entity
		iSprite++
	}))

	sort.Slice(entities[:iSprite], func(i, j int) bool {
		spriteRender1 := world.Components.SpriteRender.Get(entities[i]).(*gc.SpriteRender)
		spriteRender2 := world.Components.SpriteRender.Get(entities[j]).(*gc.SpriteRender)
		return spriteRender1.Depth < spriteRender2.Depth
	})

	for i := 0; i < iSprite; i++ {
		entity := entities[i]
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// 視界チェック - 視界内または探索済みのタイルのみ描画
		if visibilityData != nil {
			tileKey := fmt.Sprintf("%d,%d", gridElement.X, gridElement.Y)
			if tileData, exists := visibilityData[tileKey]; !exists || !tileData.Visible {
				// 探索済みかどうかもチェック
				if !world.Resources.Dungeon.ExploredTiles[*gridElement] {
					continue // 未探索かつ視界外のタイルは描画しない
				}
			}
		}

		spriteRender := world.Components.SpriteRender.Get(entity).(*gc.SpriteRender)
		pos := &gc.Position{
			X: gc.Pixel(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize/2)),
			Y: gc.Pixel(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize/2)),
		}
		if err := sys.drawImage(world, screen, spriteRender, pos, 0); err != nil {
			// エンティティ情報を追加してエラーを詳細化
			var entityInfo string
			if entity.HasComponent(world.Components.Name) {
				name := world.Components.Name.Get(entity).(*gc.Name)
				entityInfo = fmt.Sprintf("Name: %s", name.Name)
			}
			return fmt.Errorf("entity %d at (%d,%d), SpriteSheet: '%s', SpriteKey: '%s', %s: %w",
				entity, gridElement.X, gridElement.Y, spriteRender.SpriteSheetName, spriteRender.SpriteKey, entityInfo, err)
		}
	}
	return nil
}

// renderSprites はスプライトを描画する
func (sys *RenderSpriteSystem) renderSprites(world w.World, screen *ebiten.Image, visibilityData map[string]TileVisibility) error {
	var entities []ecs.Entity

	// Props を収集
	world.Manager.Join(
		world.Components.SpriteRender,
		world.Components.GridElement,
		world.Components.Prop,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entities = append(entities, entity)
	}))

	// Movers を収集
	world.Manager.Join(
		world.Components.SpriteRender,
		world.Components.GridElement,
		world.Components.TurnBased,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		entities = append(entities, entity)
	}))

	sort.Slice(entities, func(i, j int) bool {
		spriteRender1 := world.Components.SpriteRender.Get(entities[i]).(*gc.SpriteRender)
		spriteRender2 := world.Components.SpriteRender.Get(entities[j]).(*gc.SpriteRender)
		return spriteRender1.Depth < spriteRender2.Depth
	})

	// 描画
	for _, entity := range entities {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// 視界チェック - 視界内のもののみ描画
		if visibilityData != nil {
			tileKey := fmt.Sprintf("%d,%d", gridElement.X, gridElement.Y)
			if tileData, exists := visibilityData[tileKey]; !exists || !tileData.Visible {
				continue
			}
		}

		// 光源チェック - 光がある場所のみ描画（完全に暗い場所は描画しない）
		lightInfo := getCachedLightInfo(world, int(gridElement.X), int(gridElement.Y))
		if lightInfo.Darkness >= 1.0 {
			continue // 完全に暗い場所は描画しない
		}

		spriteRender := world.Components.SpriteRender.Get(entity).(*gc.SpriteRender)
		pos := &gc.Position{
			X: gc.Pixel(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2),
			Y: gc.Pixel(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2),
		}
		if err := sys.drawImage(world, screen, spriteRender, pos, 0); err != nil {
			return err
		}
	}
	return nil
}

// renderShadows は物体と壁の影を描画する
func (sys *RenderSpriteSystem) renderShadows(world w.World, screen *ebiten.Image, visibilityData map[string]TileVisibility) {
	// 物体の影
	world.Manager.Join(
		world.Components.SpriteRender,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		// TurnBased または Prop を持つエンティティのみ
		if !entity.HasComponent(world.Components.TurnBased) && !entity.HasComponent(world.Components.Prop) {
			return
		}

		spriteRender := world.Components.SpriteRender.Get(entity).(*gc.SpriteRender)

		// 高さのあるものだけが影を落とす
		if spriteRender.Depth <= gc.DepthNumRug {
			return
		}

		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// 視界チェック
		if visibilityData != nil {
			tileKey := fmt.Sprintf("%d,%d", gridElement.X, gridElement.Y)
			if tileData, exists := visibilityData[tileKey]; !exists || !tileData.Visible {
				return
			}
		}

		// グリッド座標をピクセル座標に変換
		pixelX := float64(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2 - 12)
		pixelY := float64(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pixelX, pixelY)
		SetTranslate(world, op)
		if moverShadowImage != nil {
			screen.DrawImage(moverShadowImage, op)
		}
	}))

	// 壁の影（下タイルが床の場合のみ）
	tileMap := make(map[gc.GridElement]ecs.Entity)
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.SpriteRender,
	).Visit(ecs.Visit(func(e ecs.Entity) {
		ge := world.Components.GridElement.Get(e).(*gc.GridElement)
		tileMap[*ge] = e
	}))

	world.Manager.Join(
		world.Components.SpriteRender,
		world.Components.GridElement,
		world.Components.BlockView,
		world.Components.BlockPass,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// 視界チェック
		if visibilityData != nil {
			tileKey := fmt.Sprintf("%d,%d", grid.X, grid.Y)
			if tileData, exists := visibilityData[tileKey]; !exists || !tileData.Visible {
				return
			}
		}

		spriteRender := world.Components.SpriteRender.Get(entity).(*gc.SpriteRender)

		// 高さのあるものだけが影を落とす
		if spriteRender.Depth <= gc.DepthNumRug {
			return
		}

		// 下のタイルを検索
		belowPos := gc.GridElement{X: grid.X, Y: grid.Y + 1}
		belowTileEntity, foundBelow := tileMap[belowPos]

		if !foundBelow {
			return
		}

		belowSpriteRender, ok := world.Components.SpriteRender.Get(belowTileEntity).(*gc.SpriteRender)
		if !ok || belowSpriteRender.Depth != gc.DepthNumFloor {
			return // 下が床でなければ影を描画しない
		}

		// 下のタイルが壁でないことも確認（壁->床->壁の場合は影を描画しない）
		if belowTileEntity.HasComponent(world.Components.BlockView) && belowTileEntity.HasComponent(world.Components.BlockPass) {
			return
		}

		// 影が落ちる位置（下のタイル）の光源情報を取得
		belowX := int(belowPos.X)
		belowY := int(belowPos.Y)
		lightInfo := getCachedLightInfo(world, belowX, belowY)

		// 光源の明るさに応じて影の透明度を調整
		// Darkness: 0.0(明るい) → 影薄い、1.0(暗い) → 影濃い
		baseShadowAlpha := 80.0
		shadowAlpha := uint8(baseShadowAlpha * lightInfo.Darkness)

		// 影の透明度が非常に小さい場合は描画しない（最適化）
		if shadowAlpha < 5 {
			return
		}

		// 影画像を生成（キャッシュあり）
		wallShadow := sys.getWallShadowImage(shadowAlpha)
		if wallShadow == nil {
			return
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(int(grid.X)*int(consts.TileSize)), float64(int(grid.Y)*int(consts.TileSize)+int(consts.TileSize)))
		SetTranslate(world, op)
		screen.DrawImage(wallShadow, op)
	}))
}

// getWallShadowImage は指定した透明度の壁の影画像を取得する
func (sys *RenderSpriteSystem) getWallShadowImage(alpha uint8) *ebiten.Image {
	// キャッシュキーを生成
	cacheKey := spriteImageCacheKey{
		SpriteSheetName: "wall_shadow",
		SpriteKey:       fmt.Sprintf("alpha_%d", alpha),
	}

	// キャッシュから取得
	if img, exists := sys.spriteImageCache[cacheKey]; exists {
		return img
	}

	// 新規作成
	wallWidth := int(consts.TileSize)
	wallHeight := int(consts.TileSize / 2)
	if wallWidth <= 0 || wallHeight <= 0 {
		return nil
	}

	img := ebiten.NewImage(wallWidth, wallHeight)
	img.Fill(color.RGBA{0, 0, 0, alpha})

	sys.spriteImageCache[cacheKey] = img

	return img
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
		spriteSheet, exists := (*world.Resources.SpriteSheets)[spriteRender.SpriteSheetName]
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

		result = texture.Image.SubImage(image.Rect(left, top, right, bottom)).(*ebiten.Image)
		sys.spriteImageCache[key] = result
	}

	return result, nil
}

func (sys *RenderSpriteSystem) drawImage(world w.World, screen *ebiten.Image, spriteRender *gc.SpriteRender, pos *gc.Position, angle float64) error {
	// Resourcesからスプライトシートを取得
	if world.Resources.SpriteSheets == nil {
		return fmt.Errorf("SpriteSheets が nil です")
	}
	spriteSheet, exists := (*world.Resources.SpriteSheets)[spriteRender.SpriteSheetName]
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
	SetTranslate(world, op)

	img, err := sys.getImage(world, spriteRender)
	if err != nil {
		return err
	}
	screen.DrawImage(img, op)

	// デバッグ用：スプライト番号表示(土だけ)
	cfg := config.Get()
	if cfg.ShowMonitor && spriteRender.SpriteSheetName == "tile" && strings.HasPrefix(spriteRender.SpriteKey, "dirt_") {
		// dirt_X から番号を抽出
		number := strings.TrimPrefix(spriteRender.SpriteKey, "dirt_")

		// カメラ変換を考慮したテキスト位置を計算
		textOp := &ebiten.DrawImageOptions{}
		textOp.GeoM.Translate(float64(pos.X-8), float64(pos.Y-8)) // タイルの左上付近に表示
		SetTranslate(world, textOp)

		// テキスト表示位置を逆変換で求める
		screenX, screenY := textOp.GeoM.Apply(0, 0)
		ebitenutil.DebugPrintAt(screen, number, int(screenX), int(screenY))
	}

	return nil
}

// renderBridgeDebug はデバッグモードで橋エンティティの位置を色付きで表示する
func (sys *RenderSpriteSystem) renderBridgeDebug(world w.World, screen *ebiten.Image, visibilityData map[string]TileVisibility) {
	// 橋エンティティを検索
	world.Manager.Join(
		world.Components.Interactable,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		interactable := world.Components.Interactable.Get(entity).(*gc.Interactable)
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// 出口の色とラベルを決定する
		var debugColor color.RGBA
		var label string
		switch v := interactable.Data.(type) {
		case gc.BridgeInteraction:
			label = string(v.BridgeID)
			switch v.BridgeID {
			case maptemplate.ExitIDLeft:
				debugColor = color.RGBA{255, 0, 0, 128} // 赤（半透明）
			case maptemplate.ExitIDCenter:
				debugColor = color.RGBA{0, 255, 0, 128} // 緑（半透明）
			case maptemplate.ExitIDRight:
				debugColor = color.RGBA{0, 0, 255, 128} // 青（半透明）
			case maptemplate.ExitIDMain:
				debugColor = color.RGBA{255, 165, 0, 128} // オレンジ（半透明）
			default:
				return
			}
		default:
			return
		}

		// 視界チェック
		if visibilityData != nil {
			tileKey := fmt.Sprintf("%d,%d", gridElement.X, gridElement.Y)
			if tileData, exists := visibilityData[tileKey]; !exists || !tileData.Visible {
				return
			}
		}

		// グリッド座標をピクセル座標に変換
		pixelX := float64(int(gridElement.X) * int(consts.TileSize))
		pixelY := float64(int(gridElement.Y) * int(consts.TileSize))

		// タイルサイズの色付き矩形を描画
		debugImage := ebiten.NewImage(int(consts.TileSize), int(consts.TileSize))
		debugImage.Fill(debugColor)

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(pixelX, pixelY)
		SetTranslate(world, op)
		screen.DrawImage(debugImage, op)

		// 出口ラベルをテキストで表示
		textOp := &ebiten.DrawImageOptions{}
		textOp.GeoM.Translate(pixelX+float64(consts.TileSize)/2-4, pixelY+float64(consts.TileSize)/2-4)
		SetTranslate(world, textOp)
		screenX, screenY := textOp.GeoM.Apply(0, 0)
		ebitenutil.DebugPrintAt(screen, label, int(screenX), int(screenY))
	}))
}
