package systems

import (
	"image/color"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/turns"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ExtractHUDData はworldから全てのHUDデータを抽出する
func ExtractHUDData(world w.World) hud.Data {
	return hud.Data{
		GameInfo:        extractGameInfo(world),
		MinimapData:     extractMinimapData(world),
		DebugOverlay:    extractDebugOverlay(world),
		MessageData:     extractMessageData(world, gamelog.FieldLog),
		CurrencyData:    extractCurrencyData(world),
		WeaponSlotsData: extractWeaponSlotsData(world),
	}
}

// extractGameInfo はゲーム基本情報を抽出する
func extractGameInfo(world w.World) hud.GameInfoData {
	floorNumber := world.Resources.Dungeon.Depth

	var turnNumber int
	var playerMoves int
	if world.Resources.TurnManager != nil {
		if turnManager, ok := world.Resources.TurnManager.(*turns.TurnManager); ok {
			turnNumber = turnManager.TurnNumber
			playerMoves = turnManager.PlayerMoves
		}
	}

	// プレイヤー情報を抽出する
	var playerHP, playerMaxHP, playerSP, playerMaxSP, playerEP, playerMaxEP int
	var playerWeight, playerMaxWeight float64
	world.Manager.Join(
		world.Components.Player,
		world.Components.Pools,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if poolsComponent := world.Components.Pools.Get(entity); poolsComponent != nil {
			pools := poolsComponent.(*gc.Pools)
			playerHP = pools.HP.Current
			playerMaxHP = pools.HP.Max
			playerSP = pools.SP.Current
			playerMaxSP = pools.SP.Max
			playerEP = pools.EP.Current
			playerMaxEP = pools.EP.Max
			playerWeight = pools.Weight.Current
			playerMaxWeight = pools.Weight.Max
		}
	}))

	// プレイヤーの空腹度情報を抽出
	var playerHunger int
	hungerLevel := gc.HungerNormal // デフォルト値
	world.Manager.Join(
		world.Components.Player,
		world.Components.Hunger,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if hungerComponent := world.Components.Hunger.Get(entity); hungerComponent != nil {
			hunger := hungerComponent.(*gc.Hunger)
			playerHunger = hunger.Current
			hungerLevel = hunger.GetLevel()
		}
	}))

	// 画面サイズを取得
	screenWidth, screenHeight := world.Resources.GetScreenDimensions()

	// メッセージエリアの高さを計算（message_area.goのDefaultMessageAreaConfigと同じ）
	messageAreaConfig := hud.DefaultMessageAreaConfig
	messageAreaHeight := messageAreaConfig.LogAreaMargin*2 + messageAreaConfig.MaxLogLines*messageAreaConfig.LineHeight + messageAreaConfig.YPadding*2

	return hud.GameInfoData{
		FloorNumber:       floorNumber,
		TurnNumber:        turnNumber,
		PlayerMoves:       playerMoves,
		PlayerHP:          playerHP,
		PlayerMaxHP:       playerMaxHP,
		PlayerSP:          playerSP,
		PlayerMaxSP:       playerMaxSP,
		PlayerEP:          playerEP,
		PlayerMaxEP:       playerMaxEP,
		PlayerWeight:      playerWeight,
		PlayerMaxWeight:   playerMaxWeight,
		PlayerHunger:      playerHunger,
		HungerLevel:       hungerLevel,
		MessageAreaHeight: messageAreaHeight,
		ScreenDimensions: hud.ScreenDimensions{
			Width:  screenWidth,
			Height: screenHeight,
		},
	}
}

// extractMinimapData はミニマップデータを抽出する
func extractMinimapData(world w.World) hud.MinimapData {
	// プレイヤー位置を取得
	var playerGridElement *gc.GridElement
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Player,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		playerGridElement = world.Components.GridElement.Get(entity).(*gc.GridElement)
	}))

	if playerGridElement == nil {
		return hud.MinimapData{} // プレイヤーが見つからない場合は空データ
	}

	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}

	// プレイヤーのタイル座標
	playerTileX := int(playerGridElement.X)
	playerTileY := int(playerGridElement.Y)

	// タイル色情報を抽出
	tileColors := buildTileColors(world)

	return hud.MinimapData{
		PlayerTileX:   playerTileX,
		PlayerTileY:   playerTileY,
		ExploredTiles: world.Resources.Dungeon.ExploredTiles,
		TileColors:    tileColors,
		MinimapConfig: hud.MinimapConfig{
			Width:  world.Resources.Dungeon.MinimapSettings.Width,
			Height: world.Resources.Dungeon.MinimapSettings.Height,
			Scale:  world.Resources.Dungeon.MinimapSettings.Scale,
		},
		ScreenDimensions: screenDimensions,
	}
}

// TileColorInfo はタイル色情報の内部型
type TileColorInfo = hud.TileColorInfo

// extractDebugOverlay はデバッグオーバーレイデータを抽出する
func extractDebugOverlay(world w.World) hud.DebugOverlayData {
	cfg := config.Get()
	if !cfg.ShowAIDebug {
		return hud.DebugOverlayData{Enabled: false}
	}

	// カメラ情報を取得
	var cameraPos gc.Position
	var cameraScale float64
	world.Manager.Join(
		world.Components.Camera,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(camEntity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(camEntity).(*gc.GridElement)
		// GridElementからピクセル座標に変換
		cameraPos = gc.Position{
			X: gc.Pixel(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2),
			Y: gc.Pixel(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2),
		}
		camera := world.Components.Camera.Get(camEntity).(*gc.Camera)
		cameraScale = camera.Scale
	}))

	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}

	// AI状態情報を抽出
	var aiStates []hud.AIStateInfo
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.AIMoveFSM,
		world.Components.AIRoaming,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		roaming := world.Components.AIRoaming.Get(entity).(*gc.AIRoaming)

		// AIの現在の状態を判定
		var stateText string
		if entity.HasComponent(world.Components.AIChasing) {
			stateText = "CHASING"
		} else {
			switch roaming.SubState {
			case gc.AIRoamingWaiting:
				stateText = "WAITING"
			case gc.AIRoamingDriving:
				stateText = "ROAMING"
			case gc.AIRoamingChasing:
				stateText = "CHASING"
			default:
				stateText = "UNKNOWN"
			}
		}

		// グリッド座標をピクセル座標に変換
		pixelX := float64(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2)
		pixelY := float64(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)

		// 画面座標に変換
		screenX := (pixelX-float64(cameraPos.X))*cameraScale + float64(screenDimensions.Width)/2
		screenY := (pixelY-float64(cameraPos.Y))*cameraScale + float64(screenDimensions.Height)/2

		aiStates = append(aiStates, hud.AIStateInfo{
			ScreenX:   screenX,
			ScreenY:   screenY,
			StateText: stateText,
		})
	}))

	// 視界範囲情報を抽出
	var visionRanges []hud.VisionRangeInfo
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.AIVision,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		vision := world.Components.AIVision.Get(entity).(*gc.AIVision)

		// グリッド座標をピクセル座標に変換
		pixelX := float64(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2)
		pixelY := float64(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)

		screenX := (pixelX-float64(cameraPos.X))*cameraScale + float64(screenDimensions.Width)/2
		screenY := (pixelY-float64(cameraPos.Y))*cameraScale + float64(screenDimensions.Height)/2
		scaledRadius := float32(float64(vision.ViewDistance) * cameraScale)

		visionRanges = append(visionRanges, hud.VisionRangeInfo{
			ScreenX:      screenX,
			ScreenY:      screenY,
			ScaledRadius: scaledRadius,
		})
	}))

	// HP表示情報を抽出（プレイヤー以外のPoolsを持つエンティティ）
	var hpDisplays []hud.HPDisplayInfo
	world.Manager.Join(
		world.Components.GridElement,
		world.Components.Pools,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		// プレイヤーは除外
		if entity.HasComponent(world.Components.Player) {
			return
		}

		gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)
		pools := world.Components.Pools.Get(entity).(*gc.Pools)

		// エンティティ名を取得（デバッグ用）
		var entityName string
		if nameComp := world.Components.Name.Get(entity); nameComp != nil {
			entityName = nameComp.(*gc.Name).Name
		} else {
			entityName = "Unknown"
		}

		// グリッド座標をピクセル座標に変換
		pixelX := float64(int(gridElement.X)*int(consts.TileSize) + int(consts.TileSize)/2)
		pixelY := float64(int(gridElement.Y)*int(consts.TileSize) + int(consts.TileSize)/2)

		// 画面座標に変換
		screenX := (pixelX-float64(cameraPos.X))*cameraScale + float64(screenDimensions.Width)/2
		screenY := (pixelY-float64(cameraPos.Y))*cameraScale + float64(screenDimensions.Height)/2

		hpDisplays = append(hpDisplays, hud.HPDisplayInfo{
			ScreenX:    screenX,
			ScreenY:    screenY,
			CurrentHP:  pools.HP.Current,
			MaxHP:      pools.HP.Max,
			EntityName: entityName,
		})
	}))

	return hud.DebugOverlayData{
		Enabled:          true,
		AIStates:         aiStates,
		VisionRanges:     visionRanges,
		HPDisplays:       hpDisplays,
		ScreenDimensions: screenDimensions,
	}
}

// extractMessageData はメッセージデータを抽出する
func extractMessageData(world w.World, store *gamelog.SafeSlice) hud.MessageData {
	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}

	// デフォルト設定を使用
	config := hud.DefaultMessageAreaConfig

	return hud.MessageData{
		Messages:         store.GetHistory(),
		ScreenDimensions: screenDimensions,
		Config:           config,
	}
}

// extractCurrencyData は通貨データを抽出する
func extractCurrencyData(world w.World) hud.CurrencyData {
	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}

	// デフォルト設定を使用
	config := hud.DefaultMessageAreaConfig

	// プレイヤーの地髄を取得
	currency := 0
	worldhelper.QueryPlayer(world, func(entity ecs.Entity) {
		currency = worldhelper.GetCurrency(world, entity)
	})

	return hud.CurrencyData{
		Currency:         currency,
		ScreenDimensions: screenDimensions,
		Config:           config,
	}
}

// buildTileColors はタイル色マップを構築する
func buildTileColors(world w.World) map[gc.GridElement]TileColorInfo {
	// 全エンティティをスキャンしてタイル情報をマップに格納
	tileTypeMap := make(map[gc.GridElement]bool) // true=壁, false=床

	world.Manager.Join(
		world.Components.GridElement,
		world.Components.SpriteRender,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		gridElement := gc.GridElement{X: grid.X, Y: grid.Y}
		tileTypeMap[gridElement] = entity.HasComponent(world.Components.BlockView)
	}))

	// 探索済みタイルの色情報を一括生成
	tileColors := make(map[gc.GridElement]TileColorInfo)
	for gridElement := range world.Resources.Dungeon.ExploredTiles {
		var tileColor color.RGBA
		if isWall, exists := tileTypeMap[gridElement]; exists {
			if isWall {
				tileColor = color.RGBA{100, 100, 100, 255} // 壁は灰色
			} else {
				tileColor = color.RGBA{200, 200, 200, 128} // 床は薄い灰色
			}
		} else {
			tileColor = color.RGBA{0, 0, 0, 0} // 透明
		}

		tileColors[gridElement] = TileColorInfo{
			R: tileColor.R,
			G: tileColor.G,
			B: tileColor.B,
			A: tileColor.A,
		}
	}

	return tileColors
}

// extractWeaponSlotsData は武器スロットデータを抽出する
func extractWeaponSlotsData(world w.World) hud.WeaponSlotsData {
	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}

	var slots []hud.WeaponSlotInfo
	var selectedSlot int

	// プレイヤーの武器スロット情報を取得
	worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
		weapons := worldhelper.GetWeapons(world, playerEntity)

		// 5つの武器スロット情報を作成
		for i := 0; i < 5; i++ {
			slotNumber := gc.EquipmentSlotNumber(int(gc.SlotWeapon1) + i)
			weapon := weapons[i]

			var weaponName string
			var spriteSheet, spriteName string
			hasWeapon := weapon != nil

			if hasWeapon {
				// 武器名を取得
				if nameComp := world.Components.Name.Get(*weapon); nameComp != nil {
					weaponName = nameComp.(*gc.Name).Name
				} else {
					weaponName = "???"
				}

				// スプライト情報を取得
				if spriteRender := world.Components.SpriteRender.Get(*weapon); spriteRender != nil {
					sprite := spriteRender.(*gc.SpriteRender)
					spriteSheet = sprite.SpriteSheetName
					spriteName = sprite.SpriteKey
				}
			}

			slots = append(slots, hud.WeaponSlotInfo{
				SlotNumber:  slotNumber,
				WeaponName:  weaponName,
				HasWeapon:   hasWeapon,
				SpriteSheet: spriteSheet,
				SpriteName:  spriteName,
			})
		}

		// 現在選択中のスロット（1-5）を0ベース配列インデックスに変換
		selectedSlot = world.Resources.SelectedWeaponSlot - 1
	})

	return hud.WeaponSlotsData{
		Slots:            slots,
		SelectedSlot:     selectedSlot,
		ScreenDimensions: screenDimensions,
	}
}
