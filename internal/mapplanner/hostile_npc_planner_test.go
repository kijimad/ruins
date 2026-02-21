package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHostileNPCPlanner(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plannerType := PlannerType{
		Name: "test",
		EnemyEntries: []SpawnEntry{
			{Name: "スライム", Weight: 1.0},
		},
	}
	planner := NewHostileNPCPlanner(world, plannerType)

	assert.NotNil(t, planner)
	assert.Equal(t, "test", planner.plannerType.Name)
}

func TestHostileNPCPlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	t.Run("EnemyEntriesが空の場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name:         "test_empty",
			EnemyEntries: []SpawnEntry{},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewHostileNPCPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.Empty(t, chain.PlanData.NPCs)
	})

	t.Run("EnemyEntriesがある場合はNPCが配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name: "test_with_enemies",
			EnemyEntries: []SpawnEntry{
				{Name: "スライム", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewHostileNPCPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.NotEmpty(t, chain.PlanData.NPCs)
	})

	t.Run("配置されたNPCは歩行可能なタイルにある", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name: "test_valid_position",
			EnemyEntries: []SpawnEntry{
				{Name: "スライム", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewHostileNPCPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		for _, npc := range chain.PlanData.NPCs {
			tileIdx := chain.PlanData.Level.XYTileIndex(gc.Tile(npc.X), gc.Tile(npc.Y))
			tile := chain.PlanData.Tiles[tileIdx]
			assert.False(t, tile.BlockPass, "NPC(%d,%d)が壁タイルに配置されている", npc.X, npc.Y)
		}
	})

	t.Run("複数の敵タイプが重みに応じて選択される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		plannerType := PlannerType{
			Name: "test_multiple_enemies",
			EnemyEntries: []SpawnEntry{
				{Name: "スライム", Weight: 10.0},
				{Name: "ゴブリン", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewHostileNPCPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.NotEmpty(t, chain.PlanData.NPCs)
	})
}
