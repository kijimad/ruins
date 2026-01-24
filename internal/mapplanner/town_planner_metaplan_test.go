package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewTownPlannerMetaPlan(t *testing.T) {
	t.Parallel()

	const doorPropKey = "door"

	// テスト用のワールドを作成
	world := testutil.InitTestWorld(t)
	world.Resources.RawMaster = createTownTestRawMaster()

	// TownPlannerで街マップを生成
	width, height := 50, 50
	chain, err := NewTownPlanner(gc.Tile(width), gc.Tile(height), 123)
	require.NoError(t, err)
	chain.PlanData.RawMaster = createTownTestRawMaster()

	// マップを構築
	err = chain.Plan()
	require.NoError(t, err)

	// MetaPlanが正しく生成されているかテスト
	metaPlan := &chain.PlanData

	// サイズ確認
	assert.Equal(t, gc.Tile(50), metaPlan.Level.TileWidth, "TileWidth should be 50")
	assert.Equal(t, gc.Tile(50), metaPlan.Level.TileHeight, "TileHeight should be 50")

	// タイル数確認
	expectedTiles := 50 * 50
	assert.Equal(t, expectedTiles, len(metaPlan.Tiles), "Should have correct number of tiles")

	// NPCが配置されている
	assert.Equal(t, 4, len(metaPlan.NPCs), "Town should have 4 NPCs")

	// ワープポータルが配置されているか確認
	assert.Equal(t, 1, len(metaPlan.WarpPortals), "Should have exactly one warp portal")

	// Propsが配置されているか確認
	assert.Greater(t, len(metaPlan.Props), 0, "Should have Props")

	// Doorsが配置されているか確認
	doorCount := 0
	for _, prop := range metaPlan.Props {
		if prop.Name == doorPropKey {
			doorCount++
		}
	}
	assert.Greater(t, doorCount, 0, "Should have Doors")

	// 壁と床タイルが適切に設定されているか確認
	wallCount := 0
	floorCount := 0
	for _, tile := range metaPlan.Tiles {
		if !tile.Walkable {
			wallCount++
		} else {
			floorCount++
		}
	}

	assert.Greater(t, wallCount, 0, "Should have wall tiles")
	assert.Greater(t, floorCount, 0, "Should have floor tiles")

	t.Logf("街マップ生成成功: 壁=%d, 床=%d, NPC=%d, Props=%d, ドア=%d, ワープ=%d",
		wallCount, floorCount, len(metaPlan.NPCs), len(metaPlan.Props), doorCount, len(metaPlan.WarpPortals))

	// ドアが床の位置に配置されているか確認
	doorIndex := 0
	for _, prop := range metaPlan.Props {
		if prop.Name == doorPropKey {
			doorIndex++
			idx := metaPlan.Level.XYTileIndex(gc.Tile(prop.X), gc.Tile(prop.Y))
			tile := metaPlan.Tiles[idx]
			assert.True(t, tile.Walkable, "ドア%d (%d, %d) は歩行可能タイル上にあるべき", doorIndex, prop.X, prop.Y)
			t.Logf("ドア%d: (%d, %d) - タイル: %s", doorIndex, prop.X, prop.Y, tile.Name)
		}
	}
}

func TestTownLayoutValidation(t *testing.T) {
	t.Parallel()

	// 正常なレイアウトのテスト
	tileMap, entityMap := getTownLayout()
	err := validateTownLayout(tileMap, entityMap)
	assert.NoError(t, err, "Valid town layout should not have errors")

	// ワープホール数の確認
	warpCount := 0
	for _, row := range entityMap {
		for _, char := range row {
			if char == 'w' {
				warpCount++
			}
		}
	}
	assert.Equal(t, 1, warpCount, "Should have exactly one warp portal")

	// プレイヤー開始位置の確認
	playerCount := 0
	for _, row := range entityMap {
		for _, char := range row {
			if char == '@' {
				playerCount++
			}
		}
	}
	assert.Equal(t, 1, playerCount, "Should have exactly one player start position")
}

// createTownTestRawMaster はテスト用のRawMasterを作成する
func createTownTestRawMaster() *raw.Master {
	rawMaster, _ := raw.Load(`
[[Tiles]]
Name = "Floor"
Description = "床タイル"
Walkable = true

[[Tiles]]
Name = "Wall"
Description = "壁タイル"
Walkable = false

[[Tiles]]
Name = "Dirt"
Description = "土タイル"
Walkable = true

[[ItemTables]]
Name = "通常"

[[ItemTables.Entries]]
ItemName = "TestItem"
Weight = 1.0
MinDepth = 1
MaxDepth = 50

[[EnemyTables]]
Name = "通常"

[[EnemyTables.Entries]]
EnemyName = "スライム"
Weight = 1.2
MinDepth = 1
MaxDepth = 50

[[EnemyTables.Entries]]
EnemyName = "火の玉"
Weight = 1.0
MinDepth = 1
MaxDepth = 50

[[EnemyTables]]
Name = "洞窟"

[[EnemyTables.Entries]]
EnemyName = "スライム"
Weight = 1.0
MinDepth = 1
MaxDepth = 50

[[EnemyTables.Entries]]
EnemyName = "火の玉"
Weight = 1.0
MinDepth = 1
MaxDepth = 50

[[EnemyTables]]
Name = "森"

[[EnemyTables.Entries]]
EnemyName = "スライム"
Weight = 1.2
MinDepth = 1
MaxDepth = 50

[[EnemyTables.Entries]]
EnemyName = "火の玉"
Weight = 1.0
MinDepth = 1
MaxDepth = 50

[[EnemyTables]]
Name = "廃墟"

[[EnemyTables.Entries]]
EnemyName = "スライム"
Weight = 0.9
MinDepth = 1
MaxDepth = 50

[[EnemyTables.Entries]]
EnemyName = "火の玉"
Weight = 1.0
MinDepth = 1
MaxDepth = 50
`)
	return &rawMaster
}
