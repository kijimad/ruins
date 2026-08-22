package systems

import (
	"image/color"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// exploredTiles は現ステージの探索済みタイルを返す。現ステージの StageField が未生成なら nil を返す。
// 探索履歴は StageField が持つため、HUD 抽出は StageField 経由で読む
func exploredTiles(world w.World) map[gc.GridElement]bool {
	if field := query.GetCurrentStageField(world); field != nil {
		return field.ExploredTiles
	}
	return nil
}

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
	}
}

// extractGameInfo はゲーム基本情報を抽出する
func extractGameInfo(world w.World) hud.GameInfoData {
	floorNumber := query.GetDungeon(world).CurrentStage.Depth

	// プレイヤー情報を抽出する
	var playerHP, playerMaxHP int
	var playerWeight, playerMaxWeight consts.Milligram
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

	return hud.MinimapData{
		PlayerTile:    consts.Coord[consts.Tile]{X: playerTileX, Y: playerTileY},
		ExploredTiles: exploredTiles(world),
		TileColors:    tileColors,
		MinimapConfig: hud.MinimapConfig{
			Width:  consts.MinimapWidth,
			Height: consts.MinimapHeight,
			Scale:  consts.MinimapScale,
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

	screenDimensions := hud.ScreenDimensions{
		Width:  world.Resources.ScreenDimensions.Width,
		Height: world.Resources.ScreenDimensions.Height,
	}
	// 世界を描くのと同じ投影を使う。デバッグ表示だけ別の変換に取り残すと、
	// それを手本にして古い変換が新しい箇所へ広がる。
	// このデバッグ抽出は error を返せないので、投影が組めなければ表示を諦める
	projector, err := render3d.ProjectorFor(world)
	if err != nil {
		return hud.DebugOverlayData{Enabled: false}
	}

	// AI状態情報と視界範囲情報を抽出
	var aiStates []hud.AIStateInfo
	var visionRanges []hud.VisionRangeInfo
	soloAIQuery := query.ActiveFilter2[gc.GridElement, gc.SoloAI](world).Query()
	for soloAIQuery.Next() {
		entity := soloAIQuery.Entity()
		gridElement := world.Components.GridElement.Get(entity)
		solo := world.Components.SoloAI.Get(entity)

		screen, ok := projector.BillboardTop(gridElement.Coord)
		if !ok {
			continue
		}

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

		// 視界円の半径は、足元から視界距離だけ離れたタイルまでの画面上の距離で表す
		scaledRadius := float32(0)
		if base, okBase := projector.TileCenter(gridElement.Coord, 0); okBase {
			edge := gridElement.Add(consts.Coord[consts.Tile]{X: solo.ViewDistance})
			if far, okFar := projector.TileCenter(edge, 0); okFar {
				scaledRadius = float32(math.Abs(float64(far.X - base.X)))
			}
		}
		visionRanges = append(visionRanges, hud.VisionRangeInfo{
			Screen:       screen,
			ScaledRadius: scaledRadius,
		})
	}

	// HP表示情報を抽出（プレイヤー以外のHPを持つエンティティ）
	var hpDisplays []hud.HPDisplayInfo
	hpDisplayQuery := query.ActiveFilter2[gc.GridElement, gc.HP](world).Query()
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

		screen, ok := projector.BillboardTop(gridElement.Coord)
		if !ok {
			continue
		}

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
	var currency consts.Currency
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

	tileQuery := query.ActiveFilter2[gc.GridElement, gc.SpriteRender](world).Query()
	for tileQuery.Next() {
		entity := tileQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		gridElement := gc.GridElement{Coord: grid.Coord}
		tileTypeMap[gridElement] = world.Components.BlockView.Has(entity)
	}

	// 探索済みタイルの色情報を一括生成
	tileColors := make(map[gc.GridElement]TileColorInfo)
	for gridElement := range exploredTiles(world) {
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
					weaponName = query.T(world, nameComp.Name)
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
		selectedSlot = query.GetWeaponSelection(world).Slot - 1
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
