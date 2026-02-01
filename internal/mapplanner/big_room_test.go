package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
)

func TestBigRoomPlanner(t *testing.T) {
	t.Parallel()

	width, height := gc.Tile(20), gc.Tile(20)
	seed := uint64(12345)

	chain, err := NewBigRoomPlanner(width, height, seed)
	require.NoError(t, err)
	chain.PlanData.RawMaster = CreateTestRawMaster()
	err = chain.Plan()
	require.NoError(t, err)

	// 部屋が1つだけ生成されることを確認
	if len(chain.PlanData.Rooms) != 1 {
		t.Errorf("期待される部屋数: 1, 実際: %d", len(chain.PlanData.Rooms))
	}

	// 生成された部屋のサイズを確認
	room := chain.PlanData.Rooms[0]
	expectedMinWidth := int(width) - 6 // margin * 2 + 境界
	expectedMinHeight := int(height) - 6

	actualWidth := int(room.X2 - room.X1 + 1)
	actualHeight := int(room.Y2 - room.Y1 + 1)

	if actualWidth < expectedMinWidth {
		t.Errorf("部屋の幅が小さすぎます。期待最小値: %d, 実際: %d", expectedMinWidth, actualWidth)
	}

	if actualHeight < expectedMinHeight {
		t.Errorf("部屋の高さが小さすぎます。期待最小値: %d, 実際: %d", expectedMinHeight, actualHeight)
	}

	// 床と壁の両方が存在することを確認
	floorCount := 0
	wallCount := 0
	for _, tile := range chain.PlanData.Tiles {
		if !tile.BlockPass {
			floorCount++
		} else {
			wallCount++
		}
	}

	if floorCount == 0 {
		t.Error("床タイルが存在しません")
	}
	if wallCount == 0 {
		t.Error("壁タイルが存在しません")
	}

	t.Logf("床タイル: %d, 壁タイル: %d", floorCount, wallCount)
}

func TestBigRoomVariations(t *testing.T) {
	t.Parallel()

	// 異なるシードで複数回テストして、各バリエーションが出ることを確認
	seeds := []uint64{1, 42, 123, 456, 789, 1024, 2048, 3333, 5000, 9999}

	variantCounts := make(map[string]int)

	for _, seed := range seeds {
		chain, err := NewBigRoomPlanner(20, 20, seed)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		// 部屋が1つ生成されることを確認
		if len(chain.PlanData.Rooms) != 1 {
			t.Errorf("Seed %d: Expected 1 room, got %d", seed, len(chain.PlanData.Rooms))
		}

		// タイル構成を分析してバリエーションを推測
		wallCount := 0
		floorCount := 0

		for _, tile := range chain.PlanData.Tiles {
			if !tile.BlockPass {
				floorCount++
			} else {
				wallCount++
			}
		}

		// 壁と床の比率から大まかなバリエーションを判定
		ratio := float64(wallCount) / float64(wallCount+floorCount)
		variantType := ""

		if ratio <= 0.25 {
			variantType = "basic"
		} else if ratio <= 0.35 {
			variantType = "pillars_obstacles_platform"
		} else {
			variantType = "maze"
		}

		variantCounts[variantType]++

		t.Logf("Seed %d: walls=%d, floors=%d, ratio=%.3f, type=%s",
			seed, wallCount, floorCount, ratio, variantType)
	}

	// 複数のバリエーションが生成されていることを確認
	if len(variantCounts) < 2 {
		t.Errorf("Expected multiple variants to be generated, got: %v", variantCounts)
	}

	t.Logf("Variant distribution: %v", variantCounts)
}

func TestBigRoomPlannerBoundaries(t *testing.T) {
	t.Parallel()

	// 部屋の内部が床、部屋の四周が壁になっているか確認
	width, height := gc.Tile(20), gc.Tile(20)
	seed := uint64(11111)

	chain, err := NewBigRoomPlanner(width, height, seed)
	require.NoError(t, err)
	chain.PlanData.RawMaster = CreateTestRawMaster()
	err = chain.Plan()
	require.NoError(t, err)
	require.Len(t, chain.PlanData.Rooms, 1)

	room := chain.PlanData.Rooms[0]

	// 部屋の内部が床になっているか
	for x := room.X1; x <= room.X2; x++ {
		for y := room.Y1; y <= room.Y2; y++ {
			idx := chain.PlanData.Level.XYTileIndex(x, y)
			require.Equal(t, "floor", chain.PlanData.Tiles[idx].Name,
				"部屋内部[%d,%d]が床でない", x, y)
		}
	}

	// 部屋の四周（コーナー含む）が壁になっているか
	for y := room.Y1 - 1; y <= room.Y2+1; y++ {
		if y < 0 || int(y) >= int(height) {
			continue
		}
		// 左辺
		if x := room.X1 - 1; x >= 0 {
			idx := chain.PlanData.Level.XYTileIndex(x, y)
			require.Equal(t, "wall", chain.PlanData.Tiles[idx].Name,
				"部屋左辺壁[%d,%d]が壁でない", x, y)
		}
		// 右辺
		if x := room.X2 + 1; int(x) < int(width) {
			idx := chain.PlanData.Level.XYTileIndex(x, y)
			require.Equal(t, "wall", chain.PlanData.Tiles[idx].Name,
				"部屋右辺壁[%d,%d]が壁でない", x, y)
		}
	}
	for x := room.X1; x <= room.X2; x++ {
		// 上辺
		if y := room.Y1 - 1; y >= 0 {
			idx := chain.PlanData.Level.XYTileIndex(x, y)
			require.Equal(t, "wall", chain.PlanData.Tiles[idx].Name,
				"部屋上辺壁[%d,%d]が壁でない", x, y)
		}
		// 下辺
		if y := room.Y2 + 1; int(y) < int(height) {
			idx := chain.PlanData.Level.XYTileIndex(x, y)
			require.Equal(t, "wall", chain.PlanData.Tiles[idx].Name,
				"部屋下辺壁[%d,%d]が壁でない", x, y)
		}
	}
}
