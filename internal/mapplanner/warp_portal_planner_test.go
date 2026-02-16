package mapplanner

import (
	"testing"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPortalPlanner(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	planner := NewPortalPlanner(world, PlannerTypeSmallRoom)

	assert.NotNil(t, planner)
	assert.Equal(t, PlannerTypeSmallRoom.Name, planner.plannerType.Name)
}

func TestPortalPlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	t.Run("UseFixedPortalPosがtrueの場合はポータルを配置しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}

		// UseFixedPortalPos が true のプランナータイプ
		plannerType := PlannerType{
			Name:              "test_fixed",
			UseFixedPortalPos: true,
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewPortalPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		// ポータルが配置されていないことを確認
		assert.Empty(t, chain.PlanData.NextPortals)
		assert.Empty(t, chain.PlanData.EscapePortals)
	})

	t.Run("プロシージャルマップではNextPortalsが配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 1}
		world.Resources.RawMaster = CreateTestRawMaster()

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		// NextPortalsが配置されていることを確認
		assert.NotEmpty(t, chain.PlanData.NextPortals)
	})

	t.Run("5階層ごとにEscapePortalsが配置される", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			depth        int
			expectEscape bool
			description  string
		}{
			{1, false, "1階層目は帰還ポータルなし"},
			{5, true, "5階層目は帰還ポータルあり"},
			{6, false, "6階層目は帰還ポータルなし"},
			{10, true, "10階層目は帰還ポータルあり"},
			{15, true, "15階層目は帰還ポータルあり"},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				t.Parallel()
				world := testutil.InitTestWorld(t)
				world.Resources.Dungeon = &resources.Dungeon{Depth: tc.depth}
				world.Resources.RawMaster = CreateTestRawMaster()

				chain, err := NewSmallRoomPlanner(30, 30, 12345)
				require.NoError(t, err)
				chain.PlanData.RawMaster = CreateTestRawMaster()
				err = chain.Plan()
				require.NoError(t, err)

				planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
				err = planner.PlanMeta(&chain.PlanData)
				require.NoError(t, err)

				if tc.expectEscape {
					assert.NotEmpty(t, chain.PlanData.EscapePortals, tc.description)
				} else {
					assert.Empty(t, chain.PlanData.EscapePortals, tc.description)
				}
			})
		}
	})

	t.Run("Dungeonがnilの場合エラーを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = nil

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
		err = planner.PlanMeta(&chain.PlanData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Dungeonが初期化されてない")
	})

	t.Run("配置されたポータルはプレイヤーから到達可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon = &resources.Dungeon{Depth: 5}
		world.Resources.RawMaster = CreateTestRawMaster()

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		playerX, playerY, err := chain.PlanData.GetPlayerStartPosition()
		require.NoError(t, err)

		pathFinder := NewPathFinder(&chain.PlanData)

		// NextPortalsが到達可能であることを確認
		for _, portal := range chain.PlanData.NextPortals {
			assert.True(t, pathFinder.IsReachable(playerX, playerY, portal.X, portal.Y),
				"NextPortal(%d,%d)がプレイヤー位置(%d,%d)から到達不能", portal.X, portal.Y, playerX, playerY)
		}

		// EscapePortalsが到達可能であることを確認
		for _, portal := range chain.PlanData.EscapePortals {
			assert.True(t, pathFinder.IsReachable(playerX, playerY, portal.X, portal.Y),
				"EscapePortal(%d,%d)がプレイヤー位置(%d,%d)から到達不能", portal.X, portal.Y, playerX, playerY)
		}
	})
}
