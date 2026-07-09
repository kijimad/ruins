package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRandomPlanner(t *testing.T) {
	t.Parallel()

	width, height := consts.Tile(20), consts.Tile(20)

	// 同じシードで複数回実行して同じビルダータイプが選択されることを確認
	seed := uint64(12345)

	chain1, err := NewRandomPlanner(width, height, seed)
	require.NoError(t, err)
	chain1.PlanData.RawMaster = CreateTestRawMaster()
	err = chain1.Plan()
	require.NoError(t, err)

	chain2, err := NewRandomPlanner(width, height, seed)
	require.NoError(t, err)
	chain2.PlanData.RawMaster = CreateTestRawMaster()
	err = chain2.Plan()
	require.NoError(t, err)

	// 同じシードなので同じビルダータイプが選ばれ、同じ結果になるはず
	assert.Len(t, chain2.PlanData.Rooms, len(chain1.PlanData.Rooms), "同じシードなのに部屋数が異なります")

	// タイル配置が同じことを確認
	require.Len(t, chain2.PlanData.Tiles, len(chain1.PlanData.Tiles), "同じシードなのにタイル数が異なります")

	for i, tile1 := range chain1.PlanData.Tiles {
		if chain2.PlanData.Tiles[i].Name != tile1.Name {
			assert.Equal(t, tile1.Name, chain2.PlanData.Tiles[i].Name, "タイル[%d]が異なります", i)
			break // 最初の違いだけ報告
		}
	}
}

func TestRandomPlannerTypes(t *testing.T) {
	t.Parallel()

	// 特定のシードで特定のビルダータイプが選ばれることを確認
	// これによりランダム性が正しく機能していることを検証

	width, height := consts.Tile(20), consts.Tile(20)

	// 複数のシードでテストして、異なるタイプのビルダーが選ばれることを確認
	seedResults := make(map[uint64]int) // seed -> 部屋数

	testSeeds := []uint64{1, 2, 3, 4, 5, 10, 20, 30, 100, 200}

	for _, seed := range testSeeds {
		chain, err := NewRandomPlanner(width, height, seed)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		roomCount := len(chain.PlanData.Rooms)
		seedResults[seed] = roomCount

		// タイル総数の確認
		expectedTileCount := int(width) * int(height)
		require.Len(t, chain.PlanData.Tiles, expectedTileCount,
			"シード%dでタイル数が不正", seed)

		// 部屋が生成されていることを確認
		require.Positive(t, roomCount,
			"シード%dで部屋が生成されませんでした", seed)

		// 床タイルと壁タイルの両方が存在することを確認
		floorCount := 0
		wallCount := 0
		for _, tile := range chain.PlanData.Tiles {
			if !tile.BlockPass {
				floorCount++
			} else {
				wallCount++
			}
		}
		require.Positive(t, floorCount,
			"シード%dで床タイルが生成されませんでした", seed)
		require.Positive(t, wallCount,
			"シード%dで壁タイルが生成されませんでした", seed)

		// 床と壁でタイル総数と一致することを確認
		require.Equal(t, expectedTileCount, floorCount+wallCount,
			"シード%dで床+壁がタイル総数と一致しません", seed)
	}

	// 異なるシードで異なる部屋数が生成されることを確認（ランダム性の検証）
	uniqueRoomCounts := make(map[int]bool)
	for _, count := range seedResults {
		uniqueRoomCounts[count] = true
	}
	require.GreaterOrEqual(t, len(uniqueRoomCounts), 2,
		"異なるシードで同じ部屋数しか生成されていません: %v", seedResults)

	t.Logf("各シードでの部屋数: %v", seedResults)
}
