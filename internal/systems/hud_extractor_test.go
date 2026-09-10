package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/gamelog"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractMacroMapData_帯なしはHasBandを偽にする は、オーバーワールド外では帯が無く HasBand が
// 偽になり、パネル寸法だけが設定されることを固定する。ウィジェットはこれを見て地図を描かない。
func TestExtractMacroMapData_帯なしはHasBandを偽にする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(800, 600)

	data := extractMacroMapData(world)

	assert.False(t, data.HasBand, "SeamlessBand が無ければ帯なし")
	assert.Empty(t, data.View.Cells, "帯が無ければセルは空")
	assert.Equal(t, consts.MacroMapWidth, data.Config.Width, "パネル幅は設定される")
	assert.Equal(t, consts.MacroMapHeight, data.Config.Height, "パネル高さは設定される")
	assert.Equal(t, consts.MacroMapMinGlyphPx, data.Config.MinGlyphPx, "glyph 閾値は設定される")
	assert.Equal(t, 800, data.Screen.Width, "画面幅を持つ")
}

// TestExtractMacroMapData_オーバーワールドは近傍をフォグ付きで開く は、オーバーワールドにいると
// プレイヤー中心の近傍窓が組まれ、探索済みチャンクだけが開放され未探索は伏せられることを固定する。
func TestExtractMacroMapData_オーバーワールドは近傍をフォグ付きで開く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(800, 600)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world)) // プレイヤーとキューブをスポーンする

	// 探索フォグを1タイルぶん開ける。VisionSystem が書くのと同じ ExploredTiles を直接埋める。
	// Start 直後は未探索なので、そのままだと全チャンクがフォグで伏せられる
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	pg := world.Components.GridElement.Get(player)
	query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: pg.Coord}] = true

	data := extractMacroMapData(world)

	assert.True(t, data.HasBand, "オーバーワールドでは帯がある")
	require.NotEmpty(t, data.View.Cells, "窓のセルが並ぶ")
	assert.Len(t, data.View.Cells[0], 2*consts.MacroMapChunkRadius+1, "プレイヤー中心の近傍窓ぶんの列数")

	var discovered, hidden int
	for _, row := range data.View.Cells {
		for _, cell := range row {
			if cell.Discovered {
				discovered++
			} else {
				hidden++
			}
		}
	}
	assert.Positive(t, discovered, "プレイヤー周辺は開放される")
	assert.Positive(t, hidden, "未探索の近傍はフォグで伏せる")
}

func TestGetHungerBadgeColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		level    gc.HungerLevel
		expected color.RGBA
	}{
		{
			name:     "満腹状態は緑",
			level:    gc.HungerSatiated,
			expected: color.RGBA{100, 200, 100, 255},
		},
		{
			name:     "空腹状態は黄色",
			level:    gc.HungerHungry,
			expected: color.RGBA{255, 200, 0, 255},
		},
		{
			name:     "飢餓状態は赤",
			level:    gc.HungerStarving,
			expected: color.RGBA{255, 50, 50, 255},
		},
		{
			name:     "普通状態はデフォルトの白",
			level:    gc.HungerNormal,
			expected: color.RGBA{255, 255, 255, 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getHungerBadgeColor(tt.level))
		})
	}
}

func TestExtractGameInfo(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤー情報とフロア番号を反映する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t, testutil.WithCurrentStage(gc.NewDungeonStage("test", 3)))
		world.Resources.SetScreenDimensions(800, 600)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.HP.Add(player, &gc.HP{Current: 30, Max: 50})
		world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{Current: 1000, Max: 5000})

		info := extractGameInfo(world)

		assert.Equal(t, 3, info.FloorNumber)
		assert.Equal(t, 30, info.PlayerHP)
		assert.Equal(t, 50, info.PlayerMaxHP)
		assert.Equal(t, consts.Milligram(1000), info.PlayerWeight)
		assert.Equal(t, consts.Milligram(5000), info.PlayerMaxWeight)
		assert.Equal(t, 800, info.ScreenDimensions.Width)
		assert.Equal(t, 600, info.ScreenDimensions.Height)

		config := hud.DefaultMessageAreaConfig
		expectedHeight := config.Height()
		assert.Equal(t, expectedHeight, info.MessageAreaHeight)
	})

	t.Run("プレイヤー不在時はゼロ値になる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		info := extractGameInfo(world)

		assert.Equal(t, 0, info.PlayerHP)
		assert.Equal(t, 0, info.PlayerMaxHP)
		assert.Equal(t, consts.Milligram(0), info.PlayerWeight)
		assert.Equal(t, consts.Milligram(0), info.PlayerMaxWeight)
	})
}

func TestExtractCurrencyData(t *testing.T) {
	t.Parallel()

	t.Run("プレイヤーの所持金を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.SetScreenDimensions(320, 240)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.FactionAlly.Add(player, &gc.FactionAlly{})
		world.Components.Wallet.Add(player, &gc.Wallet{Currency: 12345})

		data := extractCurrencyData(world)

		assert.Equal(t, consts.Currency(12345), data.Currency)
		assert.Equal(t, 320, data.ScreenDimensions.Width)
		assert.Equal(t, 240, data.ScreenDimensions.Height)
	})

	t.Run("プレイヤー不在時は0を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		data := extractCurrencyData(world)

		assert.Equal(t, consts.Currency(0), data.Currency)
	})
}

func TestExtractWeaponSlotsData(t *testing.T) {
	t.Parallel()

	t.Run("武器未装備なら空スロットを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.FactionAlly.Add(player, &gc.FactionAlly{})

		query.GetWeaponSelection(world).Slot = 1

		data := extractWeaponSlotsData(world)

		require.Len(t, data.Slots, 5)
		for _, slot := range data.Slots {
			assert.Empty(t, slot.WeaponName)
			assert.Empty(t, slot.SpriteSheet)
			assert.Empty(t, slot.SpriteName)
		}
		// Slot=1は0ベースで0になる
		assert.Equal(t, 0, data.SelectedSlot)
	})

	t.Run("装備した武器のスロットに情報を反映する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.FactionAlly.Add(player, &gc.FactionAlly{})

		weapon := world.ECS.NewEntity()
		world.Components.Melee.Add(weapon, &gc.Melee{})
		world.Components.Name.Add(weapon, &gc.Name{Name: "レイピア"})
		world.Components.SpriteRender.Add(weapon, &gc.SpriteRender{SpriteSheetName: "weapons", SpriteKey: "rapier"})
		world.Components.LocationEquipped.Add(weapon, &gc.LocationEquipped{Owner: player, EquipmentSlot: gc.SlotWeapon1})

		query.GetWeaponSelection(world).Slot = 3

		data := extractWeaponSlotsData(world)

		require.Len(t, data.Slots, 5)
		assert.Equal(t, "レイピア", data.Slots[0].WeaponName)
		assert.Equal(t, "weapons", data.Slots[0].SpriteSheet)
		assert.Equal(t, "rapier", data.Slots[0].SpriteName)
		for i := 1; i < 5; i++ {
			assert.Empty(t, data.Slots[i].WeaponName, "他のスロットは空のまま")
		}
		assert.Equal(t, 2, data.SelectedSlot, "Slot=3は0ベースで2になる")
	})
}

func TestExtractStatusBadgesData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		hunger     gc.Hunger
		wantBadges int
	}{
		{
			name:       "普通状態ではバッジが付かない",
			hunger:     gc.Hunger{Current: 400, Max: 500},
			wantBadges: 0,
		},
		{
			name:       "空腹状態ではバッジが付く",
			hunger:     gc.Hunger{Current: 100, Max: 500},
			wantBadges: 1,
		},
		{
			name:       "飢餓状態ではバッジが付く",
			hunger:     gc.Hunger{Current: 10, Max: 500},
			wantBadges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			world.Resources.SetScreenDimensions(640, 480)

			player := world.ECS.NewEntity()
			world.Components.Player.Add(player, &gc.Player{})
			hunger := tt.hunger
			world.Components.Hunger.Add(player, &hunger)

			data := extractStatusBadgesData(world)

			require.Len(t, data.Badges, tt.wantBadges)
			if tt.wantBadges > 0 {
				assert.Equal(t, hunger.GetLevel().String(), data.Badges[0].Text)
				assert.Equal(t, getHungerBadgeColor(hunger.GetLevel()), data.Badges[0].Color)
			}
			assert.Equal(t, 640, data.ScreenDimensions.Width)
			assert.Equal(t, 480, data.ScreenDimensions.Height)
		})
	}
}

func TestExtractStatusBadgesData_不調バッジ(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(640, 480)
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	hs := &gc.HealthStatus{}
	hs.Parts[gc.BodyPartArms].SetCondition(gc.HealthCondition{Type: gc.ConditionFracture, Timer: 60, Severity: gc.TimerToSeverity(60)})
	hs.Parts[gc.BodyPartWholeBody].SetCondition(gc.HealthCondition{Type: gc.ConditionHypothermia, Timer: 60, Severity: gc.TimerToSeverity(60)})
	world.Components.HealthStatus.Add(player, hs)

	data := extractStatusBadgesData(world)

	// 骨折は不調バッジ、低体温は体温バッジ。低体温を不調バッジに二重で出さないので合計2つ
	texts := make([]string, 0, len(data.Badges))
	for _, b := range data.Badges {
		texts = append(texts, b.Text)
	}
	require.Len(t, data.Badges, 2, "低体温は体温バッジの1つだけで二重に出さない")
	assert.Contains(t, texts, "Fracture Medium", "骨折がバッジに出る")
	assert.Contains(t, texts, "Hypothermia Medium", "低体温は体温バッジで出る")
}

func TestExtractMessageData_メッセージ履歴と画面情報を反映する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(1024, 768)

	store := gamelog.NewSafeSlice(10)
	store.Push("メッセージ1")
	store.Push("メッセージ2")

	data := extractMessageData(world, store)

	assert.Equal(t, []string{"メッセージ1", "メッセージ2"}, data.Messages)
	assert.Equal(t, 1024, data.ScreenDimensions.Width)
	assert.Equal(t, 768, data.ScreenDimensions.Height)
	assert.Equal(t, hud.DefaultMessageAreaConfig, data.Config)
}

func TestExtractDebugOverlay(t *testing.T) {
	t.Parallel()

	t.Run("無効時はEnabledがfalseになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Config.ShowAIDebug = false

		data := extractDebugOverlay(world)

		assert.False(t, data.Enabled)
		assert.Empty(t, data.AIStates)
		assert.Empty(t, data.VisionRanges)
		assert.Empty(t, data.HPDisplays)
	})

	t.Run("有効時はAI状態と視界範囲とHP表示を反映する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Config.ShowAIDebug = true
		world.Resources.SetScreenDimensions(800, 600)

		camera := world.ECS.NewEntity()
		world.Components.Camera.Add(camera, &gc.Camera{Scale: 1.0, Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist})
		world.Components.GridElement.Add(camera, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})

		soloAI := world.ECS.NewEntity()
		world.Components.GridElement.Add(soloAI, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 2, Y: 0}})
		world.Components.SoloAI.Add(soloAI, &gc.SoloAI{SubState: gc.AIStateChasing, ViewDistance: 5})

		enemy := world.ECS.NewEntity()
		world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 0}})
		world.Components.HP.Add(enemy, &gc.HP{Current: 3, Max: 10})
		world.Components.Name.Add(enemy, &gc.Name{Name: "敵"})

		player := world.ECS.NewEntity()
		world.Components.Player.Add(player, &gc.Player{})
		world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}})
		world.Components.HP.Add(player, &gc.HP{Current: 40, Max: 40})

		data := extractDebugOverlay(world)

		require.True(t, data.Enabled)
		require.Len(t, data.AIStates, 1)
		assert.Equal(t, "CHASING", data.AIStates[0].StateText)

		// 視界円の半径は、足元から ViewDistance タイルだけ離れた位置までの画面上の距離になる。
		// 透視投影なので、タイル数にタイルサイズを掛けた一定値にはならず既定カメラの見え方で決まる
		require.Len(t, data.VisionRanges, 1)
		assert.InDelta(t, 189.46, data.VisionRanges[0].ScaledRadius, 0.01)

		// プレイヤーはHP表示対象から除外され、敵のみ残る
		require.Len(t, data.HPDisplays, 1)
		assert.Equal(t, 3, data.HPDisplays[0].CurrentHP)
		assert.Equal(t, 10, data.HPDisplays[0].MaxHP)
		assert.Equal(t, "敵", data.HPDisplays[0].EntityName)
	})
}

func TestExtractHUDData_全カテゴリのデータを集約する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.SetScreenDimensions(800, 600)

	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})
	world.Components.FactionAlly.Add(player, &gc.FactionAlly{})
	world.Components.HP.Add(player, &gc.HP{Current: 10, Max: 10})
	world.Components.WeightCapacity.Add(player, &gc.WeightCapacity{Current: 0, Max: 100})
	world.Components.Wallet.Add(player, &gc.Wallet{Currency: 500})
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}})

	data := ExtractHUDData(world)

	assert.Equal(t, 10, data.GameInfo.PlayerHP)
	assert.Equal(t, consts.Currency(500), data.CurrencyData.Currency)
	require.Len(t, data.WeaponSlotsData.Slots, 5)
	assert.Equal(t, 800, data.MacroMap.Screen.Width)
}

// newColdPlayer は基本気温0度のダンジョンに体が冷えた低体温状態のプレイヤーを作る。
// 矢印の向きは環境から導出されるので、この時点では冷える向きになる
func newColdPlayer(t *testing.T) (w.World, ecs.Entity) {
	t.Helper()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage("Dead forest", 1)
	e, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	hs := world.Components.HealthStatus.Get(e)
	hs.BodyTempOffset = -3.0
	hs.Parts[gc.BodyPartWholeBody].UpdateConditionTimer(gc.ConditionHypothermia, 40)
	return world, e
}

func TestExtractGameInfo_体温ゲージの割合を返す(t *testing.T) {
	t.Parallel()
	world, _ := newColdPlayer(t)

	info := extractGameInfo(world)

	require.True(t, info.BodyTempVisible)
	assert.InDelta(t, 0.4, info.BodyTempRatio, 1e-9, "オフセット-3はクランプ幅-5..0の 0.4 に写る")
}

func TestExtractGameInfo_異常な体温オフセットでも割合を0から1に収める(t *testing.T) {
	t.Parallel()
	world, e := newColdPlayer(t)

	// セーブの編集などでクランプ幅の外の値が入っても、ゲージ描画が枠外へ出ない
	world.Components.HealthStatus.Get(e).BodyTempOffset = -100
	assert.InDelta(t, 0.0, extractGameInfo(world).BodyTempRatio, 1e-9)

	world.Components.HealthStatus.Get(e).BodyTempOffset = 100
	assert.InDelta(t, 1.0, extractGameInfo(world).BodyTempRatio, 1e-9)
}

func TestTemperatureArrow_冷えると青の下向き(t *testing.T) {
	t.Parallel()
	world, e := newColdPlayer(t)

	arrow := temperatureArrow(world, e)
	require.True(t, arrow.Visible, "体温状態があれば矢印を出す")
	assert.Equal(t, hud.TempDirectionDown, arrow.Direction, "冷えるので下向き")
	assert.Greater(t, arrow.Color.B, arrow.Color.R, "冷える向きは青が強い")
}

func TestTemperatureArrow_熱源のそばでは赤の上向き(t *testing.T) {
	t.Parallel()
	world, e := newColdPlayer(t)

	// 環境の冷えを上回る暖かさになるよう焚き火を2つ隣接させる
	_, err := lifecycle.SpawnProp(world, "fire", 6, 5)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "fire", 4, 5)
	require.NoError(t, err)

	arrow := temperatureArrow(world, e)
	require.True(t, arrow.Visible)
	assert.Equal(t, hud.TempDirectionUp, arrow.Direction, "温まるので上向き")
	assert.Greater(t, arrow.Color.R, arrow.Color.B, "温まる向きは赤が強い")
}

func TestTemperatureArrow_快適時は黄色の右向きを常時出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.Components.Player.Add(e, &gc.Player{})
	world.Components.HealthStatus.Add(e, &gc.HealthStatus{})

	arrow := temperatureArrow(world, e)
	require.True(t, arrow.Visible, "矢印は常時出す")
	assert.Equal(t, hud.TempDirectionSteady, arrow.Direction, "変化が無ければ一定")
	assert.Equal(t, color.RGBA{255, 200, 0, 255}, arrow.Color, "一定は黄色")
}

func TestTemperatureArrow_状態が無くても寒い環境なら早期警告を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage("Dead forest", 1)
	e, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	arrow := temperatureArrow(world, e)
	require.True(t, arrow.Visible, "環境が冷やす向きなら状態の確定前から矢印を出す")
	assert.Equal(t, hud.TempDirectionDown, arrow.Direction, "冷える向き")
}

func TestTemperatureArrow_境界の変化量でも横矢印は黄色(t *testing.T) {
	t.Parallel()
	// 恒常性の最終ステップなどで一定判定の境界値ちょうどになっても、横矢印が青や赤にならない
	yellow := color.RGBA{255, 200, 0, 255}
	assert.Equal(t, yellow, temperatureDirectionColor(hud.TempDirectionSteady, 0))
	assert.Equal(t, yellow, temperatureDirectionColor(hud.TempDirectionSteady, -temperatureSteadyThreshold))
	assert.Equal(t, yellow, temperatureDirectionColor(hud.TempDirectionSteady, temperatureSteadyThreshold))
}

func TestTemperatureDirectionColor_変化が速いほど濃くなる(t *testing.T) {
	t.Parallel()
	slow := temperatureDirectionColor(hud.TempDirectionUp, 0.25)
	fast := temperatureDirectionColor(hud.TempDirectionUp, 1.0)
	assert.Greater(t, slow.G, fast.G, "温まる向きは速いほど濃い赤になる")

	slowCool := temperatureDirectionColor(hud.TempDirectionDown, -0.25)
	fastCool := temperatureDirectionColor(hud.TempDirectionDown, -1.0)
	assert.Greater(t, slowCool.R, fastCool.R, "冷える向きは速いほど濃い青になる")
}

func TestTemperatureStateBadge_低体温は寒色バッジ(t *testing.T) {
	t.Parallel()
	world, e := newColdPlayer(t)

	badge, ok := temperatureStateBadge(world, e)
	require.True(t, ok, "体温状態があればバッジを出す")
	assert.Greater(t, badge.Color.B, badge.Color.R, "低体温は寒色")
}

func TestTemperatureStateBadge_快適時は出さない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	e := world.ECS.NewEntity()
	world.Components.Player.Add(e, &gc.Player{})
	world.Components.HealthStatus.Add(e, &gc.HealthStatus{})

	_, ok := temperatureStateBadge(world, e)
	assert.False(t, ok, "体温状態が無ければバッジを出さない")
}

func TestShelterMsgid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		shelter  gc.ShelterType
		expected string
	}{
		{
			name:     "屋内はIndoor",
			shelter:  gc.ShelterFull,
			expected: "Indoor",
		},
		{
			name:     "半屋外はSemi-outdoor",
			shelter:  gc.ShelterPartial,
			expected: "Semi-outdoor",
		},
		{
			name:     "屋外はOutdoor",
			shelter:  gc.ShelterNone,
			expected: "Outdoor",
		},
		{
			name:     "未知の値はOutdoorへ落とす",
			shelter:  gc.ShelterType(99),
			expected: "Outdoor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, shelterMsgid(tt.shelter))
		})
	}
}

func TestGetFatigueBadgeColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		level    gc.FatigueLevel
		expected color.RGBA
	}{
		{
			name:     "過労は赤",
			level:    gc.FatigueExhausted,
			expected: color.RGBA{255, 50, 50, 255},
		},
		{
			name:     "疲労は黄",
			level:    gc.FatigueTired,
			expected: color.RGBA{255, 200, 0, 255},
		},
		{
			name:     "通常段階も黄のデフォルト",
			level:    gc.FatigueNormal,
			expected: color.RGBA{255, 200, 0, 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getFatigueBadgeColor(tt.level))
		})
	}
}

func TestAmbientTempDisplayColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		temp     int
		expected color.RGBA
	}{
		{
			name:     "快適帯の下限を下回ると青",
			temp:     query.ComfortableTempLower - 1,
			expected: color.RGBA{150, 190, 255, 255},
		},
		{
			name:     "快適帯の上限を上回ると赤",
			temp:     query.ComfortableTempUpper + 1,
			expected: color.RGBA{255, 170, 120, 255},
		},
		{
			name:     "快適帯の下限ちょうどは白",
			temp:     query.ComfortableTempLower,
			expected: color.RGBA{255, 255, 255, 255},
		},
		{
			name:     "快適帯の上限ちょうどは白",
			temp:     query.ComfortableTempUpper,
			expected: color.RGBA{255, 255, 255, 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ambientTempDisplayColor(tt.temp))
		})
	}
}
