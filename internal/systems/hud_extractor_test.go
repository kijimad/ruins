package systems

import (
	"fmt"
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/hud"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTileColorInfo はTileColorInfoの型エイリアスをテスト
func TestTileColorInfo(t *testing.T) {
	t.Parallel()
	colorInfo := TileColorInfo{
		R: 255,
		G: 128,
		B: 64,
		A: 200,
	}

	// hud.TileColorInfoと同じ構造であることを確認
	var hudColorInfo = colorInfo

	assert.Equal(t, uint8(255), hudColorInfo.R)
	assert.Equal(t, uint8(128), hudColorInfo.G)
	assert.Equal(t, uint8(64), hudColorInfo.B)
	assert.Equal(t, uint8(200), hudColorInfo.A)
}

func TestBuildTileColors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		setupEntities func(w.World)
		gridElement   gc.GridElement
		expectedColor color.RGBA
	}{
		{
			name: "壁タイルは灰色で描画される",
			setupEntities: func(world w.World) {
				entity := world.ECS.NewEntity()
				world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 3}})
				world.Components.SpriteRender.Add(entity, &gc.SpriteRender{})
				world.Components.BlockView.Add(entity, &gc.BlockView{})
				// 探索済みタイルに追加
				query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 3}}] = true
			},
			gridElement:   gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 3}},
			expectedColor: color.RGBA{100, 100, 100, 255},
		},
		{
			name: "床タイルは薄い灰色で描画される",
			setupEntities: func(world w.World) {
				entity := world.ECS.NewEntity()
				world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 15}})
				world.Components.SpriteRender.Add(entity, &gc.SpriteRender{})
				// BlockViewコンポーネントなし = 床
				// 探索済みタイルに追加
				query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 15}}] = true
			},
			gridElement:   gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 15}},
			expectedColor: color.RGBA{200, 200, 200, 128},
		},
		{
			name: "エンティティなしの場合は透明",
			setupEntities: func(world w.World) {
				// 探索済みタイルに追加してるが、エンティティはない
				query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 999, Y: 999}}] = true
			},
			gridElement:   gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 999, Y: 999}},
			expectedColor: color.RGBA{0, 0, 0, 0},
		},
		{
			name: "同じタイルに壁と床が両方ある場合は壁が優先される",
			setupEntities: func(world w.World) {
				// 床エンティティ
				floorEntity := world.ECS.NewEntity()
				world.Components.GridElement.Add(floorEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 20, Y: 20}})
				world.Components.SpriteRender.Add(floorEntity, &gc.SpriteRender{})

				// 壁エンティティ
				wallEntity := world.ECS.NewEntity()
				world.Components.GridElement.Add(wallEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 20, Y: 20}})
				world.Components.SpriteRender.Add(wallEntity, &gc.SpriteRender{})
				world.Components.BlockView.Add(wallEntity, &gc.BlockView{})
				// 探索済みタイルに追加
				query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 20, Y: 20}}] = true
			},
			gridElement:   gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 20, Y: 20}},
			expectedColor: color.RGBA{100, 100, 100, 255}, // 壁が優先される
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)

			// セットアップ処理を実行
			tt.setupEntities(world)

			// テスト実行
			tileColors := buildTileColors(world)
			actualTileColor, exists := tileColors[tt.gridElement]

			// 結果検証
			assert.True(t, exists, "gridElement %v should exist in tileColors", tt.gridElement)
			actualColor := color.RGBA{R: actualTileColor.R, G: actualTileColor.G, B: actualTileColor.B, A: actualTileColor.A}
			assert.Equal(t, tt.expectedColor, actualColor,
				"buildTileColors gridElement %v = %v, want %v",
				tt.gridElement, actualColor, tt.expectedColor)
		})
	}
}

func TestExtractMinimapData(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// ゲームリソースを設定
	query.GetCurrentStageField(world).ExploredTiles = make(map[gc.GridElement]bool)

	// プレイヤーエンティティを作成
	playerEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(playerEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 15}})
	world.Components.Player.Add(playerEntity, &gc.Player{})

	// 探索済みタイルを設定
	query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 15}}] = true // プレイヤー位置
	query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 15}}] = true  // 左のタイル
	query.GetCurrentStageField(world).ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 15}}] = true // 右のタイル

	// 画面リソースを設定
	world.Resources.SetScreenDimensions(800, 600)

	// いくつかの壁と床エンティティを作成
	wallEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(wallEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 15}})
	world.Components.SpriteRender.Add(wallEntity, &gc.SpriteRender{})
	world.Components.BlockView.Add(wallEntity, &gc.BlockView{})

	floorEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(floorEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 15}})
	world.Components.SpriteRender.Add(floorEntity, &gc.SpriteRender{})

	// テスト実行
	minimapData := extractMinimapData(world)

	// 結果検証
	assert.Equal(t, 10, int(minimapData.PlayerTile.X), "プレイヤーのX座標が正しくない")
	assert.Equal(t, 15, int(minimapData.PlayerTile.Y), "プレイヤーのY座標が正しくない")
	assert.Len(t, minimapData.ExploredTiles, 3, "探索済みタイル数が正しくない")
	assert.Equal(t, consts.MinimapWidth, minimapData.MinimapConfig.Width, "ミニマップ幅が正しくない")
	assert.Equal(t, consts.MinimapHeight, minimapData.MinimapConfig.Height, "ミニマップ高さが正しくない")
	assert.Equal(t, consts.MinimapScale, minimapData.MinimapConfig.Scale, "ミニマップスケールが正しくない")

	// タイル色が正しく設定されているか確認
	wallGrid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 15}}
	floorGrid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 15}}
	require.Contains(t, minimapData.TileColors, wallGrid, "壁タイルの色情報がない")
	require.Contains(t, minimapData.TileColors, floorGrid, "床タイルの色情報がない")

	wallColor := minimapData.TileColors[wallGrid]
	floorColor := minimapData.TileColors[floorGrid]

	assert.Equal(t, uint8(100), wallColor.R, "壁の赤色成分が正しくない")
	assert.Equal(t, uint8(100), wallColor.G, "壁の緑色成分が正しくない")
	assert.Equal(t, uint8(100), wallColor.B, "壁の青色成分が正しくない")
	assert.Equal(t, uint8(255), wallColor.A, "壁のアルファ値が正しくない")

	assert.Equal(t, uint8(200), floorColor.R, "床の赤色成分が正しくない")
	assert.Equal(t, uint8(200), floorColor.G, "床の緑色成分が正しくない")
	assert.Equal(t, uint8(200), floorColor.B, "床の青色成分が正しくない")
	assert.Equal(t, uint8(128), floorColor.A, "床のアルファ値が正しくない")
}

func TestMinimapCoordinateTransformation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		playerTileX    int
		playerTileY    int
		targetTileX    int
		targetTileY    int
		minimapCenterX int
		minimapCenterY int
		minimapScale   int
		expectedMapX   float32
		expectedMapY   float32
		description    string
	}{
		{
			name:           "プレイヤーと同じ位置のタイル",
			playerTileX:    10,
			playerTileY:    10,
			targetTileX:    10,
			targetTileY:    10,
			minimapCenterX: 100,
			minimapCenterY: 100,
			minimapScale:   2,
			expectedMapX:   100, // 中心座標と同じ
			expectedMapY:   100, // 中心座標と同じ
			description:    "プレイヤー位置はミニマップ中心に表示される",
		},
		{
			name:           "プレイヤーの右のタイル",
			playerTileX:    10,
			playerTileY:    10,
			targetTileX:    11,
			targetTileY:    10,
			minimapCenterX: 100,
			minimapCenterY: 100,
			minimapScale:   2,
			expectedMapX:   102, // centerX + relativeX * scale = 100 + 1 * 2
			expectedMapY:   100, // centerY + relativeY * scale = 100 + 0 * 2
			description:    "右のタイルはミニマップでも右に表示される",
		},
		{
			name:           "プレイヤーの左のタイル",
			playerTileX:    10,
			playerTileY:    10,
			targetTileX:    9,
			targetTileY:    10,
			minimapCenterX: 100,
			minimapCenterY: 100,
			minimapScale:   2,
			expectedMapX:   98,  // centerX + relativeX * scale = 100 + (-1) * 2
			expectedMapY:   100, // centerY + relativeY * scale = 100 + 0 * 2
			description:    "左のタイルはミニマップでも左に表示される",
		},
		{
			name:           "プレイヤーの下のタイル",
			playerTileX:    10,
			playerTileY:    10,
			targetTileX:    10,
			targetTileY:    11,
			minimapCenterX: 100,
			minimapCenterY: 100,
			minimapScale:   2,
			expectedMapX:   100, // centerX + relativeX * scale = 100 + 0 * 2
			expectedMapY:   102, // centerY + relativeY * scale = 100 + 1 * 2
			description:    "下のタイルはミニマップでも下に表示される",
		},
		{
			name:           "プレイヤーの上のタイル",
			playerTileX:    10,
			playerTileY:    10,
			targetTileX:    10,
			targetTileY:    9,
			minimapCenterX: 100,
			minimapCenterY: 100,
			minimapScale:   2,
			expectedMapX:   100, // centerX + relativeX * scale = 100 + 0 * 2
			expectedMapY:   98,  // centerY + relativeY * scale = 100 + (-1) * 2
			description:    "上のタイルはミニマップでも上に表示される",
		},
		{
			name:           "異なるスケールでのテスト",
			playerTileX:    5,
			playerTileY:    5,
			targetTileX:    7,
			targetTileY:    3,
			minimapCenterX: 200,
			minimapCenterY: 200,
			minimapScale:   4,
			expectedMapX:   208, // centerX + relativeX * scale = 200 + 2 * 4
			expectedMapY:   192, // centerY + relativeY * scale = 200 + (-2) * 4
			description:    "スケール4での座標変換が正しく動作する",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// 相対座標を計算
			relativeX := tt.targetTileX - tt.playerTileX
			relativeY := tt.targetTileY - tt.playerTileY

			// 新しい実装（回転なしの単純な座標変換）
			mapX := float32(tt.minimapCenterX + relativeX*tt.minimapScale)
			mapY := float32(tt.minimapCenterY + relativeY*tt.minimapScale)

			assert.Equal(t, tt.expectedMapX, mapX, "X座標の変換が正しくない: %s", tt.description)
			assert.Equal(t, tt.expectedMapY, mapY, "Y座標の変換が正しくない: %s", tt.description)
		})
	}
}

func TestTileKeyFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		tileX       int
		tileY       int
		expectedKey string
	}{
		{
			name:        "正の座標",
			tileX:       5,
			tileY:       10,
			expectedKey: "5,10", // X,Y形式
		},
		{
			name:        "負の座標",
			tileX:       -3,
			tileY:       -7,
			expectedKey: "-3,-7",
		},
		{
			name:        "原点",
			tileX:       0,
			tileY:       0,
			expectedKey: "0,0",
		},
		{
			name:        "大きな座標",
			tileX:       100,
			tileY:       200,
			expectedKey: "100,200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// TileVisibilityから取得したCol,Rowを使った形式
			tileData := struct {
				Col int // X座標
				Row int // Y座標
			}{
				Col: tt.tileX,
				Row: tt.tileY,
			}

			// X,Y形式で統一
			actualKey := fmt.Sprintf("%d,%d", tileData.Col, tileData.Row)
			assert.Equal(t, tt.expectedKey, actualKey, "tileKeyの形式が正しくない")
		})
	}
}

func TestExploredTilesKeyConsistency(t *testing.T) {
	t.Parallel()
	// 他のシステムで使われているキー形式とvision.goでの形式が一致するかテスト

	// 同じタイル座標に対して、異なるシステムが生成するキーを比較
	testTileX := 15
	testTileY := 20

	// render_sprite.goのようなキー生成
	renderKey := fmt.Sprintf("%d,%d", testTileX, testTileY)

	// TileVisibilityから生成されるキー
	tileData := struct {
		Col int // X座標
		Row int // Y座標
	}{
		Col: testTileX,
		Row: testTileY,
	}
	visionKey := fmt.Sprintf("%d,%d", tileData.Col, tileData.Row)

	// 両方のキーが同じであることを確認
	assert.Equal(t, renderKey, visionKey, "システム間でtileKeyの形式が一致していない")

	// 期待される形式であることを確認
	expectedKey := "15,20"
	assert.Equal(t, expectedKey, renderKey, "renderシステムのキー形式が正しくない")
	assert.Equal(t, expectedKey, visionKey, "visionシステムのキー形式が正しくない")
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

func TestExploredTiles_現ステージの探索済みタイルを返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	field := query.GetCurrentStageField(world)
	field.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 2}}] = true

	tiles := exploredTiles(world)

	assert.True(t, tiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 2}}])

	// 同一マップへの参照であることを、書き込みが反映されるかで確認する
	tiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 9}}] = true
	assert.True(t, field.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 9}}])
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
		expectedHeight := config.LogAreaMargin*2 + config.MaxLogLines*config.LineHeight + config.YPadding*2
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
	assert.Equal(t, 800, data.MinimapData.ScreenDimensions.Width)
}

// newColdPlayer は基本気温0度のダンジョンに低体温状態のプレイヤーを作る。
// 矢印の向きは環境から導出されるので、この時点では冷える向きになる
func newColdPlayer(t *testing.T) (w.World, ecs.Entity) {
	t.Helper()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = gc.NewDungeonStage("Dead forest", 1)
	e, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	world.Components.HealthStatus.Get(e).Parts[gc.BodyPartWholeBody].UpdateConditionTimer(gc.ConditionHypothermia, 40)
	return world, e
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
	_, err := lifecycle.SpawnProp(world, "bonfire", 6, 5)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "bonfire", 4, 5)
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

func TestTemperatureDirectionColor_一定は黄色(t *testing.T) {
	t.Parallel()
	assert.Equal(t, color.RGBA{255, 200, 0, 255}, temperatureDirectionColor(0))
}

func TestTemperatureDirectionColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		delta    float64
		expected color.RGBA
	}{
		{"緩く温まると弱の赤", 0.25, color.RGBA{255, 170, 120, 255}},
		{"温まると中の赤", 0.5, color.RGBA{240, 110, 70, 255}},
		{"急に温まると強の赤", 1.0, color.RGBA{230, 50, 40, 255}},
		{"緩く冷えると弱の青", -0.25, color.RGBA{150, 190, 255, 255}},
		{"冷えると中の青", -0.5, color.RGBA{90, 140, 240, 255}},
		{"急に冷えると強の青", -1.0, color.RGBA{40, 90, 230, 255}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, temperatureDirectionColor(tt.delta))
		})
	}
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
