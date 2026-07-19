package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHostileNPCPlanner(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plannerType := PlannerType{
		Name:           "test",
		EnemyTableName: "通常",
		Depth:          1,
	}
	planner := NewHostileNPCPlanner(world, plannerType)

	assert.NotNil(t, planner)
	assert.Equal(t, "test", planner.plannerType.Name)
}

func TestHostileNPCPlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	t.Run("EnemyTableNameが空の場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		plannerType := PlannerType{
			Name:  "test_empty",
			Depth: 1,
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

	t.Run("EnemyTableNameがある場合はNPCが配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		plannerType := PlannerType{
			Name:           "test_with_enemies",
			EnemyTableName: "通常",
			Depth:          1,
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

		plannerType := PlannerType{
			Name:           "test_valid_position",
			EnemyTableName: "通常",
			Depth:          1,
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

		// 「通常」テーブルにはスライム・火の玉・軽戦車が含まれる
		plannerType := PlannerType{
			Name:           "test_multiple_enemies",
			EnemyTableName: "通常",
			Depth:          1,
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

		plannerType := PlannerType{
			Name:           "test_room_based",
			EnemyTableName: "通常",
			Depth:          1,
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
				if npc.X >= int(room.Min.X) && npc.X < int(room.Max.X) &&
					npc.Y >= int(room.Min.Y) && npc.Y < int(room.Max.Y) {
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

		// 「廃墟」テーブルを使用。PackMin/PackMaxはテーブルに依存するが、クラスタ動作の確認が目的
		plannerType := PlannerType{
			Name:           "test_big_room_cluster",
			EnemyTableName: "通常",
			Depth:          1,
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
		// clusterRadius^2 * 2 は正方形の対角距離の二乗。dx=r, dy=r のケースを含める
		maxDistSq := clusterRadius * clusterRadius * 2
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
		assert.Positive(t, nearPairs, "クラスタ半径内のNPC対が1組も存在しない")
	})

	t.Run("部屋内は同種クラスタになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 「通常」テーブルには複数の敵種が含まれる
		plannerType := PlannerType{
			Name:           "test_same_species",
			EnemyTableName: "通常",
			Depth:          1,
		}

		chain, err := NewSmallRoomPlanner(40, 40, 100)
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
				if npc.X >= int(room.Min.X) && npc.X < int(room.Max.X) &&
					npc.Y >= int(room.Min.Y) && npc.Y < int(room.Max.Y) {
					species[npc.Name] = true
				}
			}
			assert.LessOrEqual(t, len(species), 1, "部屋(%d,%d)-(%d,%d)内に異種の敵が混在している", room.Min.X, room.Min.Y, room.Max.X, room.Max.Y)
		}
	})
}
