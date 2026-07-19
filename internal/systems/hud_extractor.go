package systems

import (
	"image/color"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ExtractHUDData はworldから全てのHUDデータを抽出する
func ExtractHUDData(world w.World) hud.Data {
	return hud.Data{
		GameInfo:         extractGameInfo(world),
		MinimapData:      extractMinimapData(world),
		DebugOverlay:     extractDebugOverlay(world),
		MessageData:      extractMessageData(world, query.GetGameLog(world)),
		CurrencyData:     extractCurrencyData(world),
		WeaponSlotsData:  extractWeaponSlotsData(world),
		StatusBadgesData: extractStatusBadgesData(world),
		SquadHUDData:     extractSquadHUDData(world),
	}
}

// extractGameInfo はゲーム基本情報を抽出する
func extractGameInfo(world w.World) hud.GameInfoData {
	floorNumber := query.GetDungeon(world).Depth

	// プレイヤー情報を抽出する
	var playerHP, playerMaxHP int
	var playerWeight, playerMaxWeight float64
	playerQuery := ecs.NewFilter3[gc.Player, gc.HP, gc.WeightCapacity](world.ECS).Query()
	for playerQuery.Next() {
		entity := playerQuery.Entity()
		hp := world.Components.HP.Get(entity)
		cw := world.Components.WeightCapacity.Get(entity)
		playerHP = hp.Current
		playerMaxHP = hp.Max
		playerWeight = cw.Current
		playerMaxWeight = cw.Max
	}

	// 画面サイズを取得
	screenWidth, screenHeight := world.Resources.GetScreenDimensions()

	// メッセージエリアの高さを計算（message_area.goのDefaultMessageAreaConfigと同じ）
	messageAreaConfig := hud.DefaultMessageAreaConfig
	messageAreaHeight := messageAreaConfig.LogAreaMargin*2 + messageAreaConfig.MaxLogLines*messageAreaConfig.LineHeight + messageAreaConfig.YPadding*2

	return hud.GameInfoData{
		FloorNumber:       floorNumber,
		PlayerHP:          playerHP,
		PlayerMaxHP:       playerMaxHP,
		PlayerWeight:      playerWeight,
		PlayerMaxWeight:   playerMaxWeight,
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
	playerQuery := ecs.NewFilter2[gc.GridElement, gc.Player](world.ECS).Query()
	for playerQuery.Next() {
		entity := playerQuery.Entity()
		playerGridElement = world.Components.GridElement.Get(entity)
	}

	if playerGridElement == nil {
		return hud.MinimapData{} // プレイヤーが見つからない場合は空データ
	}

	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}

	// プレイヤーのタイル座標
	playerTileX := playerGridElement.X
	playerTileY := playerGridElement.Y

	// タイル色情報を抽出
	tileColors := buildTileColors(world)

	// 隊員位置を抽出する
	var squadPositions []hud.MinimapMarker
	_, err := query.GetPlayerEntity(world)
	if err == nil {
		for _, member := range query.SquadMembers(world) {
			if world.Components.GridElement.Has(member) {
				grid := world.Components.GridElement.Get(member)
				squadPositions = append(squadPositions, hud.MinimapMarker{
					Tile: consts.Coord[consts.Tile]{X: grid.X, Y: grid.Y},
				})
			}
		}
	}

	return hud.MinimapData{
		PlayerTile:     consts.Coord[consts.Tile]{X: playerTileX, Y: playerTileY},
		ExploredTiles:  query.GetDungeon(world).ExploredTiles,
		TileColors:     tileColors,
		SquadPositions: squadPositions,
		MinimapConfig: hud.MinimapConfig{
			Width:  query.GetDungeon(world).MinimapSettings.Width,
			Height: query.GetDungeon(world).MinimapSettings.Height,
			Scale:  query.GetDungeon(world).MinimapSettings.Scale,
		},
		ScreenDimensions: screenDimensions,
	}
}

// TileColorInfo はタイル色情報の内部型
type TileColorInfo = hud.TileColorInfo

// extractDebugOverlay はデバッグオーバーレイデータを抽出する
func extractDebugOverlay(world w.World) hud.DebugOverlayData {
	if !world.Config.ShowAIDebug {
		return hud.DebugOverlayData{Enabled: false}
	}

	// カメラ情報を取得
	var cameraPos gc.Position
	var cameraScale float64
	cameraQuery := ecs.NewFilter2[gc.Camera, gc.GridElement](world.ECS).Query()
	for cameraQuery.Next() {
		camEntity := cameraQuery.Entity()
		gridElement := world.Components.GridElement.Get(camEntity)
		// GridElementからワールドピクセル座標に変換
		cameraPos = gc.Position{Coord: consts.TileCenterToWorld(gridElement.Coord)}
		camera := world.Components.Camera.Get(camEntity)
		cameraScale = camera.Scale
	}

	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}

	// AI状態情報と視界範囲情報を抽出
	var aiStates []hud.AIStateInfo
	var visionRanges []hud.VisionRangeInfo
	soloAIQuery := ecs.NewFilter2[gc.GridElement, gc.SoloAI](world.ECS).Query()
	for soloAIQuery.Next() {
		entity := soloAIQuery.Entity()
		gridElement := world.Components.GridElement.Get(entity)
		solo := world.Components.SoloAI.Get(entity)

		// グリッド座標をワールドピクセルへ、さらにスクリーン座標へ変換
		screen := consts.WorldToScreen(consts.TileCenterToWorld(gridElement.Coord), cameraPos.Coord, cameraScale, screenDimensions.Width, screenDimensions.Height)

		var stateText string
		switch solo.SubState {
		case gc.AIStateWaiting:
			stateText = "WAITING"
		case gc.AIStateDriving:
			stateText = "ROAMING"
		case gc.AIStateChasing:
			stateText = "CHASING"
		case gc.AIStateFleeing:
			stateText = "FLEEING"
		default:
			stateText = "UNKNOWN"
		}
		aiStates = append(aiStates, hud.AIStateInfo{
			Screen:    screen,
			StateText: stateText,
		})

		scaledRadius := float32(float64(solo.ViewDistance) * float64(consts.TileSize) * cameraScale)
		visionRanges = append(visionRanges, hud.VisionRangeInfo{
			Screen:       screen,
			ScaledRadius: scaledRadius,
		})
	}

	// HP表示情報を抽出（プレイヤー以外のHPを持つエンティティ）
	var hpDisplays []hud.HPDisplayInfo
	hpDisplayQuery := ecs.NewFilter2[gc.GridElement, gc.HP](world.ECS).Query()
	for hpDisplayQuery.Next() {
		entity := hpDisplayQuery.Entity()
		// プレイヤーは除外
		if world.Components.Player.Has(entity) {
			continue
		}

		gridElement := world.Components.GridElement.Get(entity)
		hp := world.Components.HP.Get(entity)

		// エンティティ名を取得（デバッグ用）
		var entityName string
		if nameComp := world.Components.Name.Get(entity); nameComp != nil {
			entityName = nameComp.Name
		} else {
			entityName = "Unknown"
		}

		// グリッド座標をワールドピクセルへ、さらにスクリーン座標へ変換
		screen := consts.WorldToScreen(consts.TileCenterToWorld(gridElement.Coord), cameraPos.Coord, cameraScale, screenDimensions.Width, screenDimensions.Height)

		hpDisplays = append(hpDisplays, hud.HPDisplayInfo{
			Screen:     screen,
			CurrentHP:  hp.Current,
			MaxHP:      hp.Max,
			EntityName: entityName,
		})
	}

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
	query.Player(world, func(entity ecs.Entity) {
		currency = query.GetCurrency(world, entity)
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

	tileQuery := ecs.NewFilter2[gc.GridElement, gc.SpriteRender](world.ECS).Query()
	for tileQuery.Next() {
		entity := tileQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		gridElement := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: grid.X, Y: grid.Y}}
		tileTypeMap[gridElement] = world.Components.BlockView.Has(entity)
	}

	// 探索済みタイルの色情報を一括生成
	tileColors := make(map[gc.GridElement]TileColorInfo)
	for gridElement := range query.GetDungeon(world).ExploredTiles {
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
	query.Player(world, func(playerEntity ecs.Entity) {
		weapons := query.GetWeapons(world, playerEntity)

		// 5つの武器スロット情報を作成
		for i := range 5 {
			slotNumber := gc.EquipmentSlotNumber(int(gc.SlotWeapon1) + i)
			weapon := weapons[i]

			var weaponName string
			var spriteSheet, spriteName string

			if weapon != nil {
				// 武器名を取得
				if nameComp := world.Components.Name.Get(*weapon); nameComp != nil {
					weaponName = nameComp.Name
				} else {
					weaponName = "???"
				}

				// スプライト情報を取得
				if spriteRender := world.Components.SpriteRender.Get(*weapon); spriteRender != nil {
					spriteSheet = spriteRender.SpriteSheetName
					spriteName = spriteRender.SpriteKey
				}
			}

			slots = append(slots, hud.WeaponSlotInfo{
				SlotNumber:  slotNumber,
				WeaponName:  weaponName,
				SpriteSheet: spriteSheet,
				SpriteName:  spriteName,
			})
		}

		// 現在選択中のスロット（1-5）を0ベース配列インデックスに変換
		selectedSlot = query.GetDungeon(world).SelectedWeaponSlot - 1
	})

	return hud.WeaponSlotsData{
		Slots:            slots,
		SelectedSlot:     selectedSlot,
		ScreenDimensions: screenDimensions,
	}
}

// extractStatusBadgesData はステータスバッジデータを抽出する
func extractStatusBadgesData(world w.World) hud.StatusBadgesData {
	var badges []hud.StatusBadge

	// プレイヤーの空腹度を取得
	hungerQuery := ecs.NewFilter2[gc.Player, gc.Hunger](world.ECS).Query()
	for hungerQuery.Next() {
		entity := hungerQuery.Entity()
		if hunger := world.Components.Hunger.Get(entity); hunger != nil {
			level := hunger.GetLevel()
			if level != gc.HungerNormal {
				badges = append(badges, hud.StatusBadge{
					Text:  level.String(),
					Color: getHungerBadgeColor(level),
				})
			}
		}
	}

	// 画面サイズを取得
	screenWidth, screenHeight := world.Resources.GetScreenDimensions()

	// メッセージエリアの高さを計算
	messageAreaConfig := hud.DefaultMessageAreaConfig
	messageAreaHeight := messageAreaConfig.LogAreaMargin*2 + messageAreaConfig.MaxLogLines*messageAreaConfig.LineHeight + messageAreaConfig.YPadding*2

	return hud.StatusBadgesData{
		Badges:            badges,
		MessageAreaHeight: messageAreaHeight,
		ScreenDimensions: hud.ScreenDimensions{
			Width:  screenWidth,
			Height: screenHeight,
		},
	}
}

// extractSquadHUDData は隊員HP一覧データを抽出する
func extractSquadHUDData(world w.World) hud.SquadHUDData {
	screenWidth, screenHeight := world.Resources.GetScreenDimensions()

	_, err := query.GetPlayerEntity(world)
	if err != nil {
		return hud.SquadHUDData{
			ScreenDimensions: hud.ScreenDimensions{Width: screenWidth, Height: screenHeight},
		}
	}

	var members []hud.SquadHUDMember
	for _, member := range query.SquadMembers(world) {
		name := query.GetEntityName(member, world)
		hp := world.Components.HP.Get(member)
		members = append(members, hud.SquadHUDMember{
			Name:      name,
			CurrentHP: hp.Current,
			MaxHP:     hp.Max,
		})
	}

	return hud.SquadHUDData{
		Members: members,
		ScreenDimensions: hud.ScreenDimensions{
			Width:  screenWidth,
			Height: screenHeight,
		},
	}
}

// getHungerBadgeColor は空腹度に応じたバッジ色を返す
func getHungerBadgeColor(level gc.HungerLevel) color.RGBA {
	switch level {
	case gc.HungerSatiated:
		return color.RGBA{100, 200, 100, 255} // 緑（満腹）
	case gc.HungerHungry:
		return color.RGBA{255, 200, 0, 255} // 黄色（空腹）
	case gc.HungerStarving:
		return color.RGBA{255, 50, 50, 255} // 赤（飢餓）
	default:
		return color.RGBA{255, 255, 255, 255}
	}
}
