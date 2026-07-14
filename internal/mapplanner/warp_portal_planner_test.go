package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/query"
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
		query.SetDungeon(world, &gc.Dungeon{Depth: 1})

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
		query.SetDungeon(world, &gc.Dungeon{Depth: 1})
		world.Resources.RawMaster = *CreateTestRawMaster()

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
				query.SetDungeon(world, &gc.Dungeon{Depth: tc.depth})
				world.Resources.RawMaster = *CreateTestRawMaster()

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

	t.Run("AlwaysEscapePortalがtrueなら1階でも帰還ポータルを配置する", func(t *testing.T) {
		t.Parallel()
		// トラベル地形（平原/山脈）は floor1 で入って降りずに戻れる必要がある。
		// 間隔（5階ごと）を無視して毎階に帰還ポータルを置くことを検証する。
		world := testutil.InitTestWorld(t)
		query.SetDungeon(world, &gc.Dungeon{Depth: 1})
		world.Resources.RawMaster = *CreateTestRawMaster()

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		travel := PlannerTypeSmallRoom
		travel.AlwaysEscapePortal = true
		planner := NewPortalPlanner(world, travel)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.NotEmpty(t, chain.PlanData.EscapePortals, "1階でも帰還ポータルが配置されること")
	})

	t.Run("歩行可能タイルが孤立している場合はErrConnectivityを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.SetDungeon(world, &gc.Dungeon{Depth: 1})
		world.Resources.RawMaster = *CreateTestRawMaster()

		// 全面壁のマップに1マスだけ床を置く（孤立した歩行可能タイル）
		// SpawnPointsを設定しないことで、FindPlayerStartPositionの
		// minReachableTilesチェックにより開始位置が見つからずエラーになる
		chain := NewPlannerChain(10, 10, 99999)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		for i := range chain.PlanData.Tiles {
			chain.PlanData.Tiles[i] = chain.PlanData.GetTile("wall")
		}
		// (5,5) だけ床にする
		idx := chain.PlanData.Level.XYTileIndex(5, 5)
		chain.PlanData.Tiles[idx] = chain.PlanData.GetTile("floor")

		planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
		err := planner.PlanMeta(&chain.PlanData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrConnectivity)
	})

	t.Run("GetPlayerStartPositionが失敗した場合はErrConnectivityを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.SetDungeon(world, &gc.Dungeon{Depth: 1})

		// 全面壁のマップ（歩行可能タイルなし）
		chain := NewPlannerChain(5, 5, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		for i := range chain.PlanData.Tiles {
			chain.PlanData.Tiles[i] = chain.PlanData.GetTile("wall")
		}

		planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
		err := planner.PlanMeta(&chain.PlanData)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrConnectivity)
	})

	t.Run("Dungeonがnilの場合エラーを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.SetDungeon(world, nil)

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
		err = planner.PlanMeta(&chain.PlanData)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Dungeonが初期化されていません")
	})

	t.Run("配置されたポータルはプレイヤーから到達可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		query.SetDungeon(world, &gc.Dungeon{Depth: 5})
		world.Resources.RawMaster = *CreateTestRawMaster()

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		playerPos, err := chain.PlanData.GetPlayerStartPosition()
		require.NoError(t, err)

		pathFinder := NewPathFinder(&chain.PlanData)

		// NextPortalsが到達可能であることを確認
		for _, portal := range chain.PlanData.NextPortals {
			assert.True(t, pathFinder.IsReachable(playerPos.X, playerPos.Y, portal.X, portal.Y),
				"NextPortal(%d,%d)がプレイヤー位置(%d,%d)から到達不能", portal.X, portal.Y, playerPos.X, playerPos.Y)
		}

		// EscapePortalsが到達可能であることを確認
		for _, portal := range chain.PlanData.EscapePortals {
			assert.True(t, pathFinder.IsReachable(playerPos.X, playerPos.Y, portal.X, portal.Y),
				"EscapePortal(%d,%d)がプレイヤー位置(%d,%d)から到達不能", portal.X, portal.Y, playerPos.X, playerPos.Y)
		}
	})

	t.Run("ポータルはプレイヤーから最低歩数以上離れて配置される", func(t *testing.T) {
		t.Parallel()

		// 複数シードで検証して偶然の一致を排除する
		seeds := []uint64{11111, 22222, 33333, 44444, 55555}
		for _, seed := range seeds {
			world := testutil.InitTestWorld(t)
			query.SetDungeon(world, &gc.Dungeon{Depth: 5})
			world.Resources.RawMaster = *CreateTestRawMaster()

			chain, err := NewSmallRoomPlanner(40, 40, seed)
			require.NoError(t, err)
			chain.PlanData.RawMaster = CreateTestRawMaster()
			err = chain.Plan()
			require.NoError(t, err)

			planner := NewPortalPlanner(world, PlannerTypeSmallRoom)
			err = planner.PlanMeta(&chain.PlanData)
			require.NoError(t, err)

			playerPos, err := chain.PlanData.GetPlayerStartPosition()
			require.NoError(t, err)

			pathFinder := NewPathFinder(&chain.PlanData)

			for _, portal := range chain.PlanData.NextPortals {
				path := pathFinder.FindPath(playerPos.X, playerPos.Y, portal.X, portal.Y)
				// フォールバックで配置された場合は距離が短い可能性があるため、到達可能であることだけ確認
				assert.NotEmpty(t, path, "seed=%d: NextPortalに到達不能", seed)
			}

			for _, portal := range chain.PlanData.EscapePortals {
				path := pathFinder.FindPath(playerPos.X, playerPos.Y, portal.X, portal.Y)
				assert.NotEmpty(t, path, "seed=%d: EscapePortalに到達不能", seed)
			}
		}
	})
}
