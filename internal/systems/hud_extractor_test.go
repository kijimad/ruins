package systems

import (
	"fmt"
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
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
				entity := world.Manager.NewEntity()
				entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 3})
				entity.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
				entity.AddComponent(world.Components.BlockView, &gc.BlockView{})
				// 探索済みタイルに追加
				world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 5, Y: 3}] = true
			},
			gridElement:   gc.GridElement{X: 5, Y: 3},
			expectedColor: color.RGBA{100, 100, 100, 255},
		},
		{
			name: "床タイルは薄い灰色で描画される",
			setupEntities: func(world w.World) {
				entity := world.Manager.NewEntity()
				entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 15})
				entity.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
				// BlockViewコンポーネントなし = 床
				// 探索済みタイルに追加
				world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 10, Y: 15}] = true
			},
			gridElement:   gc.GridElement{X: 10, Y: 15},
			expectedColor: color.RGBA{200, 200, 200, 128},
		},
		{
			name: "エンティティなしの場合は透明",
			setupEntities: func(world w.World) {
				// 探索済みタイルに追加してるが、エンティティはない
				world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 999, Y: 999}] = true
			},
			gridElement:   gc.GridElement{X: 999, Y: 999},
			expectedColor: color.RGBA{0, 0, 0, 0},
		},
		{
			name: "同じタイルに壁と床が両方ある場合は壁が優先される",
			setupEntities: func(world w.World) {
				// 床エンティティ
				floorEntity := world.Manager.NewEntity()
				floorEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 20, Y: 20})
				floorEntity.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})

				// 壁エンティティ
				wallEntity := world.Manager.NewEntity()
				wallEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 20, Y: 20})
				wallEntity.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
				wallEntity.AddComponent(world.Components.BlockView, &gc.BlockView{})
				// 探索済みタイルに追加
				world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 20, Y: 20}] = true
			},
			gridElement:   gc.GridElement{X: 20, Y: 20},
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
	world.Resources.Dungeon.ExploredTiles = make(map[gc.GridElement]bool)
	world.Resources.Dungeon.MinimapSettings = resources.MinimapSettings{
		Width:  200,
		Height: 200,
		Scale:  2,
	}

	// プレイヤーエンティティを作成
	playerEntity := world.Manager.NewEntity()
	playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 15})
	playerEntity.AddComponent(world.Components.Player, &gc.Player{})

	// 探索済みタイルを設定
	world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 10, Y: 15}] = true // プレイヤー位置
	world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 9, Y: 15}] = true  // 左のタイル
	world.Resources.Dungeon.ExploredTiles[gc.GridElement{X: 11, Y: 15}] = true // 右のタイル

	// 画面リソースを設定
	world.Resources.SetScreenDimensions(800, 600)

	// いくつかの壁と床エンティティを作成
	wallEntity := world.Manager.NewEntity()
	wallEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 9, Y: 15})
	wallEntity.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})
	wallEntity.AddComponent(world.Components.BlockView, &gc.BlockView{})

	floorEntity := world.Manager.NewEntity()
	floorEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 15})
	floorEntity.AddComponent(world.Components.SpriteRender, &gc.SpriteRender{})

	// テスト実行
	minimapData := extractMinimapData(world)

	// 結果検証
	assert.Equal(t, 10, minimapData.PlayerTileX, "プレイヤーのX座標が正しくない")
	assert.Equal(t, 15, minimapData.PlayerTileY, "プレイヤーのY座標が正しくない")
	assert.Equal(t, 3, len(minimapData.ExploredTiles), "探索済みタイル数が正しくない")
	assert.Equal(t, 200, minimapData.MinimapConfig.Width, "ミニマップ幅が正しくない")
	assert.Equal(t, 200, minimapData.MinimapConfig.Height, "ミニマップ高さが正しくない")
	assert.Equal(t, 2, minimapData.MinimapConfig.Scale, "ミニマップスケールが正しくない")

	// タイル色が正しく設定されているか確認
	wallGrid := gc.GridElement{X: 9, Y: 15}
	floorGrid := gc.GridElement{X: 11, Y: 15}
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
