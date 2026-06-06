package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
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
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

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
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

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
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

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
			tileIdx := chain.PlanData.Level.XYTileIndex(consts.Tile(npc.X), consts.Tile(npc.Y))
			tile := chain.PlanData.Tiles[tileIdx]
			assert.False(t, tile.BlockPass, "NPC(%d,%d)が壁タイルに配置されている", npc.X, npc.Y)
		}
	})

	t.Run("複数の敵タイプが重みに応じて選択される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

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

	t.Run("部屋がある場合はNPCが部屋内に配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_room_based",
			EnemyEntries: []SpawnEntry{
				{Name: "スライム", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)
		require.NotEmpty(t, chain.PlanData.Rooms, "テストにはRoomsが必要")

		planner := NewHostileNPCPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		// 各NPCがいずれかの部屋内にいることを確認
		for _, npc := range chain.PlanData.NPCs {
			inRoom := false
			for _, room := range chain.PlanData.Rooms {
				if npc.X >= int(room.X1) && npc.X < int(room.X2) &&
					npc.Y >= int(room.Y1) && npc.Y < int(room.Y2) {
					inRoom = true
					break
				}
			}
			assert.True(t, inRoom, "NPC(%d,%d)がどの部屋にも属していない", npc.X, npc.Y)
		}
	})

	t.Run("大部屋でもクラスタメンバーが密集して配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_big_room_cluster",
			EnemyEntries: []SpawnEntry{
				{Name: "スライム", Weight: 1.0, PackMin: 1, PackMax: 3},
			},
		}

		chain, err := NewBigRoomPlanner(40, 40, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)
		require.NotEmpty(t, chain.PlanData.Rooms)

		planner := NewHostileNPCPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)
		require.Greater(t, len(chain.PlanData.NPCs), 1, "クラスタ検証には2体以上のNPCが必要")

		// ホットスポット配置により、少なくとも一部のNPC対がclusterRadius以内に密集していることを確認する
		npcs := chain.PlanData.NPCs
		nearPairs := 0
		maxDistSq := clusterRadius * clusterRadius * 2 // 対角距離を考慮
		for i, a := range npcs {
			for j := i + 1; j < len(npcs); j++ {
				b := npcs[j]
				dx := a.X - b.X
				dy := a.Y - b.Y
				if dx*dx+dy*dy <= maxDistSq {
					nearPairs++
				}
			}
		}
		assert.Greater(t, nearPairs, 0, "クラスタ半径内のNPC対が1組も存在しない")
	})

	t.Run("部屋内は同種クラスタになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_same_species",
			EnemyEntries: []SpawnEntry{
				{Name: "スライム", Weight: 1.0},
				{Name: "ゴブリン", Weight: 1.0},
				{Name: "コボルト", Weight: 1.0},
			},
		}

		chain, err := NewSmallRoomPlanner(40, 40, 99999)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)
		require.NotEmpty(t, chain.PlanData.Rooms)

		planner := NewHostileNPCPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		for _, room := range chain.PlanData.Rooms {
			species := map[string]bool{}
			for _, npc := range chain.PlanData.NPCs {
				if npc.X >= int(room.X1) && npc.X < int(room.X2) &&
					npc.Y >= int(room.Y1) && npc.Y < int(room.Y2) {
					species[npc.Name] = true
				}
			}
			assert.LessOrEqual(t, len(species), 1, "部屋(%d,%d)-(%d,%d)内に異種の敵が混在している", room.X1, room.Y1, room.X2, room.Y2)
		}
	})
}
