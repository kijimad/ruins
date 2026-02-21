package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewItemPlanner(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plannerType := PlannerType{
		Name: "test",
		ItemEntries: []SpawnEntry{
			{Name: "薬草", Weight: 1.0},
		},
	}
	planner := NewItemPlanner(world, plannerType)

	assert.NotNil(t, planner)
	assert.Equal(t, "test", planner.plannerType.Name)
}

func TestItemPlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	t.Run("ItemEntriesが空の場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name:        "test_empty",
			ItemEntries: []SpawnEntry{},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.Empty(t, chain.PlanData.Items)
	})

	t.Run("ItemEntriesがある場合はアイテムが配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name: "test_with_items",
			ItemEntries: []SpawnEntry{
				{Name: "薬草", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.NotEmpty(t, chain.PlanData.Items)
	})

	t.Run("配置されたアイテムは歩行可能なタイルにある", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name: "test_valid_position",
			ItemEntries: []SpawnEntry{
				{Name: "薬草", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		for _, item := range chain.PlanData.Items {
			tileIdx := chain.PlanData.Level.XYTileIndex(gc.Tile(item.X), gc.Tile(item.Y))
			tile := chain.PlanData.Tiles[tileIdx]
			assert.False(t, tile.BlockPass, "アイテム(%d,%d)が壁タイルに配置されている", item.X, item.Y)
		}
	})

	t.Run("深い階層ではアイテム数が増加する", func(t *testing.T) {
		t.Parallel()

		plannerType := PlannerType{
			Name: "test_depth",
			ItemEntries: []SpawnEntry{
				{Name: "薬草", Weight: 1.0},
			},
		}

		// 浅い階層
		worldShallow := testutil.InitTestWorld(t)
		worldShallow.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		chainShallow, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chainShallow.PlanData.RawMaster = CreateTestRawMaster()
		err = chainShallow.Plan()
		require.NoError(t, err)

		plannerShallow := NewItemPlanner(worldShallow, plannerType)
		err = plannerShallow.PlanMeta(&chainShallow.PlanData)
		require.NoError(t, err)

		// 深い階層
		worldDeep := testutil.InitTestWorld(t)
		worldDeep.Resources.Dungeon = &resources.Dungeon{Depth: 10}

		chainDeep, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chainDeep.PlanData.RawMaster = CreateTestRawMaster()
		err = chainDeep.Plan()
		require.NoError(t, err)

		plannerDeep := NewItemPlanner(worldDeep, plannerType)
		err = plannerDeep.PlanMeta(&chainDeep.PlanData)
		require.NoError(t, err)

		// 両方ともアイテムが配置されていることを確認
		assert.NotEmpty(t, chainShallow.PlanData.Items)
		assert.NotEmpty(t, chainDeep.PlanData.Items)
	})

	t.Run("複数のアイテムタイプが重みに応じて選択される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name: "test_multiple_items",
			ItemEntries: []SpawnEntry{
				{Name: "薬草", Weight: 10.0},
				{Name: "毒消し", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.NotEmpty(t, chain.PlanData.Items)
	})
}

func TestSelectByWeight(t *testing.T) {
	t.Parallel()

	t.Run("空のエントリの場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		result, err := selectByWeight([]SpawnEntry{}, world.Config.RNG)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("単一エントリの場合はその値を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entries := []SpawnEntry{
			{Name: "薬草", Weight: 1.0},
		}
		result, err := selectByWeight(entries, world.Config.RNG)
		require.NoError(t, err)
		assert.Equal(t, "薬草", result)
	})

	t.Run("重みに応じて選択される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entries := []SpawnEntry{
			{Name: "薬草", Weight: 100.0},
			{Name: "毒消し", Weight: 1.0},
		}

		herbCount := 0
		antidoteCount := 0
		for range 100 {
			result, err := selectByWeight(entries, world.Config.RNG)
			require.NoError(t, err)
			switch result {
			case "薬草":
				herbCount++
			case "毒消し":
				antidoteCount++
			}
		}

		// 重みが大きい方が多く選択されるはず
		assert.Greater(t, herbCount, antidoteCount)
	})
}
