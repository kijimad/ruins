package aiinput

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exploreAllTiles はテスト用マップ全域を探索済みにする
func exploreAllTiles(world w.World) {
	field := query.GetCurrentStageField(world)
	for x := range 50 {
		for y := range 50 {
			field.ExploredTiles[gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}] = true
		}
	}
}

// surroundWithWalls は指定座標の隣接8マスすべてにBlockPassの障害物を置き、移動不可にする
func surroundWithWalls(world w.World, center consts.Coord[consts.Tile]) {
	for _, d := range eightDirections {
		wall := world.ECS.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{Coord: center.Add(d)})
		world.Components.BlockPass.Add(wall, &gc.BlockPass{})
	}
	query.InvalidateSpatialIndex(world)
}

func TestSquadPlanner_GatherSquadContext(t *testing.T) {
	t.Parallel()

	t.Run("SquadAIがなければfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		entity := world.ECS.NewEntity()
		world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}})

		sp := newSquadPlanner(newTestRNG())
		ctx, ok := sp.gatherSquadSnapshot(world, entity)
		assert.False(t, ok)
		assert.Nil(t, ctx)
	})

	t.Run("プレイヤーがいなければfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		entity := world.ECS.NewEntity()
		world.Components.GridElement.Add(entity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}})
		world.Components.SquadAI.Add(entity, &gc.SquadAI{})

		sp := newSquadPlanner(newTestRNG())
		ctx, ok := sp.gatherSquadSnapshot(world, entity)
		assert.False(t, ok)
		assert.Nil(t, ctx)
	})

	t.Run("正常系ではコンテキストを構築する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx, ok := sp.gatherSquadSnapshot(world, member)
		require.True(t, ok)
		require.NotNil(t, ctx)
		assert.Equal(t, leader, ctx.LeaderEntity)
		assert.Equal(t, world.Components.GridElement.Get(member), ctx.Grid)
		assert.Equal(t, world.Components.GridElement.Get(leader), ctx.LeaderGrid)
	})
}

func TestSquadPlanner_PlanRetreatAction(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = 15
	memberGrid.Y = 10
	query.InvalidateSpatialIndex(world)

	sp := newSquadPlanner(newTestRNG())
	ctx := &squadSnapshot{
		Grid:         memberGrid,
		Squad:        &gc.SquadAI{},
		LeaderEntity: leader,
		LeaderGrid:   world.Components.GridElement.Get(leader),
	}

	b, ok := sp.planRetreatAction(world, member, ctx)
	require.True(t, ok, "リーダーに向かって後退できるべき")
	require.NotNil(t, b)
	assert.Equal(t, gc.BehaviorMove, b.Name())
}

func TestSquadPlanner_IsOutsideExploredArea(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	grid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
	sp := newSquadPlanner(newTestRNG())

	assert.True(t, sp.isOutsideExploredArea(world, grid), "未探索なら圏外")

	query.GetCurrentStageField(world).ExploredTiles[*grid] = true
	assert.False(t, sp.isOutsideExploredArea(world, grid), "探索済みなら圏内")
}

func TestSquadPlanner_PlanReturnToExploredArea(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = 20
	memberGrid.Y = 20
	query.InvalidateSpatialIndex(world)

	sp := newSquadPlanner(newTestRNG())
	ctx := &squadSnapshot{
		Grid:         memberGrid,
		Squad:        &gc.SquadAI{},
		LeaderEntity: leader,
		LeaderGrid:   world.Components.GridElement.Get(leader),
	}

	b, ok := sp.planReturnToExploredArea(world, member, ctx)
	require.True(t, ok, "リーダーに向かって移動できるべき")
	assert.Equal(t, gc.BehaviorMove, b.Name())
}

func TestSquadPlanner_PlanAction(t *testing.T) {
	t.Parallel()

	t.Run("HP低下時は隣接する敵がいても後退を優先する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 15
		memberGrid.Y = 10
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 16, Y: 10}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		hp := world.Components.HP.Get(member)
		hp.Current = hp.Max * 10 / 100

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{CombatCurrent: gc.CombatAttack, ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b := sp.planAction(world, member, ctx)
		require.NotNil(t, b)
		_, isAttack := b.(*activity.AttackActivity)
		assert.False(t, isAttack, "HP低下時は攻撃せず後退するべき")
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("HP低下でも後退できなければ次の優先度に進む", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 15
		memberGrid.Y = 10
		surroundWithWalls(world, memberGrid.Coord)

		hp := world.Components.HP.Get(member)
		hp.Current = hp.Max * 10 / 100

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{CombatCurrent: gc.CombatIgnore, Movement: gc.SquadStationary, ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b := sp.planAction(world, member, ctx)
		require.NotNil(t, b)
		assert.Equal(t, gc.BehaviorWait, b.Name(), "後退できなければ位置ポリシーの待機に落ちる")
	})

	t.Run("エリア外なら戦闘より復帰を優先する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		// 未探索の位置に配置し、隣に敵も置く
		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 15
		memberGrid.Y = 10
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 16, Y: 10}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{CombatCurrent: gc.CombatAttack, ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b := sp.planAction(world, member, ctx)
		require.NotNil(t, b)
		_, isAttack := b.(*activity.AttackActivity)
		assert.False(t, isAttack, "未探索エリアでは攻撃せずリーダーへ復帰するべき")
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("何も優先条件がなければ位置ポリシーに委ねる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		exploreAllTiles(world)

		memberGrid := world.Components.GridElement.Get(member)
		leaderGrid := world.Components.GridElement.Get(leader)
		memberGrid.X = leaderGrid.X
		memberGrid.Y = leaderGrid.Y

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{CombatCurrent: gc.CombatIgnore, Movement: gc.SquadEscort, ItemPickup: gc.PolicyIgnore, ItemHandling: gc.PolicyKeep, ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   leaderGrid,
		}

		b := sp.planAction(world, member, ctx)
		require.NotNil(t, b)
		assert.Equal(t, gc.BehaviorWait, b.Name(), "護衛距離内では待機するべき")
	})
}

func TestSquadPlanner_PlanCombatAction(t *testing.T) {
	t.Parallel()

	t.Run("CombatAttackなら攻撃計画に委譲する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 11
		memberGrid.Y = 10
		enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 12, Y: 10}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{CombatCurrent: gc.CombatAttack, ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b, ok := sp.planCombatAction(world, member, ctx)
		require.True(t, ok)
		attack, ok := b.(*activity.AttackActivity)
		require.True(t, ok, "型が *activity.AttackActivity であるべき")
		assert.Equal(t, enemy, attack.Target)
	})

	t.Run("CombatEvadeなら回避計画に委譲する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 12
		memberGrid.Y = 10
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{CombatCurrent: gc.CombatEvade, ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b, ok := sp.planCombatAction(world, member, ctx)
		require.True(t, ok)
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("CombatIgnoreでは何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         world.Components.GridElement.Get(member),
			Squad:        &gc.SquadAI{CombatCurrent: gc.CombatIgnore, ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		_, ok := sp.planCombatAction(world, member, ctx)
		assert.False(t, ok, "CombatIgnoreは戦闘計画を持たない")
	})
}

func TestSquadPlanner_PlanAttackAction(t *testing.T) {
	t.Parallel()

	t.Run("視界内に敵がいなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         world.Components.GridElement.Get(member),
			Squad:        &gc.SquadAI{ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		_, ok := sp.planAttackAction(world, member, ctx)
		assert.False(t, ok)
	})

	t.Run("視界内で離れた敵には接近する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 10
		memberGrid.Y = 15
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 18}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b, ok := sp.planAttackAction(world, member, ctx)
		require.True(t, ok)
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})
}

func TestSquadPlanner_PlanEvadeAction(t *testing.T) {
	t.Parallel()

	t.Run("視界内に敵がいなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         world.Components.GridElement.Get(member),
			Squad:        &gc.SquadAI{ViewDistance: 5},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		_, ok := sp.planEvadeAction(world, member, ctx)
		assert.False(t, ok)
	})

	t.Run("視界内の敵から離れる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 12
		memberGrid.Y = 10
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 13, Y: 10}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		initialX := int(memberGrid.X)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{
			Grid:         memberGrid,
			Squad:        &gc.SquadAI{ViewDistance: 10},
			LeaderEntity: leader,
			LeaderGrid:   world.Components.GridElement.Get(leader),
		}

		b, ok := sp.planEvadeAction(world, member, ctx)
		require.True(t, ok)
		move, ok := b.(*activity.MoveActivity)
		require.True(t, ok)
		assert.Less(t, int(move.Destination.X), initialX, "敵から離れる方向に移動するべき")
	})
}

func TestSquadPlanner_PlanPositionAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		movement gc.SquadMovement
	}{
		{"護衛", gc.SquadEscort},
		{"前衛", gc.SquadVanguard},
		{"巡回", gc.SquadPatrol},
		{"待機", gc.SquadStationary},
		{"後退は護衛と同じ扱い", gc.SquadRetreat},
		{"未知のポリシーはデフォルト待機", gc.SquadMovement("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
			require.NoError(t, err)
			member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
			require.NoError(t, err)

			memberGrid := world.Components.GridElement.Get(member)
			leaderGrid := world.Components.GridElement.Get(leader)
			memberGrid.X = leaderGrid.X
			memberGrid.Y = leaderGrid.Y

			sp := newSquadPlanner(newTestRNG())
			ctx := &squadSnapshot{
				Grid:         memberGrid,
				Squad:        &gc.SquadAI{Movement: tt.movement},
				LeaderEntity: leader,
				LeaderGrid:   leaderGrid,
			}

			b := sp.planPositionAction(world, member, ctx)
			require.NotNil(t, b, "常に何らかの行動を返す")
		})
	}
}

func TestSquadPlanner_PlanEscortAction(t *testing.T) {
	t.Parallel()

	t.Run("護衛距離内なら待機する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		leaderGrid := world.Components.GridElement.Get(leader)
		memberGrid.X = leaderGrid.X
		memberGrid.Y = leaderGrid.Y

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, LeaderGrid: leaderGrid}

		b := sp.planEscortAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorWait, b.Name())
	})

	t.Run("離れていれば追従移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 20
		memberGrid.Y = 20
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, LeaderGrid: world.Components.GridElement.Get(leader)}

		b := sp.planEscortAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("離れていて移動もできなければ待機にフォールバックする", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 20
		memberGrid.Y = 20
		surroundWithWalls(world, memberGrid.Coord)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, LeaderGrid: world.Components.GridElement.Get(leader)}

		b := sp.planEscortAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorWait, b.Name(), "壁に囲まれて移動できないときは待機する")
	})
}

func TestSquadPlanner_PlanVanguardAction(t *testing.T) {
	t.Parallel()

	t.Run("離れすぎていればリーダーに接近する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 20
		memberGrid.Y = 20
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, LeaderGrid: world.Components.GridElement.Get(leader)}

		b := sp.planVanguardAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("距離内でランダム移動できれば移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		exploreAllTiles(world)

		memberGrid := world.Components.GridElement.Get(member)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, Squad: &gc.SquadAI{}, LeaderGrid: world.Components.GridElement.Get(leader)}

		b := sp.planVanguardAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("距離内で移動先がなければ待機する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		// 探索済みタイルを一切設定しないので tryRandomMove は必ず失敗する
		memberGrid := world.Components.GridElement.Get(member)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, Squad: &gc.SquadAI{}, LeaderGrid: world.Components.GridElement.Get(leader)}

		b := sp.planVanguardAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorWait, b.Name())
	})
}

func TestSquadPlanner_PlanSquadPatrolAction(t *testing.T) {
	t.Parallel()

	t.Run("探索済みエリアがあれば巡回移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		exploreAllTiles(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: world.Components.GridElement.Get(member), Squad: &gc.SquadAI{}}

		b := sp.planSquadPatrolAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("移動先がなければ待機する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: world.Components.GridElement.Get(member), Squad: &gc.SquadAI{}}

		b := sp.planSquadPatrolAction(world, member, ctx)
		assert.Equal(t, gc.BehaviorWait, b.Name())
	})
}

func TestSquadPlanner_FindNearestEnemy(t *testing.T) {
	t.Parallel()

	t.Run("複数の敵から最も近いものを選ぶ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 10
		memberGrid.Y = 15

		far, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 25}, "火の玉")
		require.NoError(t, err)
		near, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 17}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, Squad: &gc.SquadAI{ViewDistance: 20}}

		gotEntity, gotGrid, dist := sp.findNearestEnemy(world, member, ctx)
		require.NotNil(t, gotEntity)
		assert.Equal(t, near, *gotEntity)
		assert.NotEqual(t, far, *gotEntity)
		assert.Equal(t, world.Components.GridElement.Get(near), gotGrid)
		assert.Equal(t, 2, dist)
	})

	t.Run("視界外の敵は無視する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 10
		memberGrid.Y = 15
		_, err = lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 10, Y: 40}, "火の玉")
		require.NoError(t, err)
		query.InvalidateSpatialIndex(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: memberGrid, Squad: &gc.SquadAI{ViewDistance: 5}}

		gotEntity, _, _ := sp.findNearestEnemy(world, member, ctx)
		assert.Nil(t, gotEntity)
	})
}

func TestSquadPlanner_TryMoveToward(t *testing.T) {
	t.Parallel()

	t.Run("到達可能なら移動アクションを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 15
		memberGrid.Y = 15
		query.InvalidateSpatialIndex(world)

		target := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}

		sp := newSquadPlanner(newTestRNG())
		b, ok := sp.tryMoveToward(world, member, memberGrid, target)
		require.True(t, ok)
		assert.Equal(t, gc.BehaviorMove, b.Name())
	})

	t.Run("既に目的地にいれば移動しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)

		sp := newSquadPlanner(newTestRNG())
		_, ok := sp.tryMoveToward(world, member, memberGrid, memberGrid)
		assert.False(t, ok, "移動元と目的地が同じなら移動しない")
	})

	t.Run("壁に囲まれていれば移動しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		memberGrid := world.Components.GridElement.Get(member)
		memberGrid.X = 20
		memberGrid.Y = 20
		surroundWithWalls(world, memberGrid.Coord)

		target := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}

		sp := newSquadPlanner(newTestRNG())
		_, ok := sp.tryMoveToward(world, member, memberGrid, target)
		assert.False(t, ok)
	})
}

func TestSquadPlanner_TryMoveAway(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
	require.NoError(t, err)

	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = 12
	memberGrid.Y = 10
	query.InvalidateSpatialIndex(world)

	threat := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 13, Y: 10}}

	sp := newSquadPlanner(newTestRNG())
	b, ok := sp.tryMoveAway(world, member, memberGrid, threat)
	require.True(t, ok)
	move, ok := b.(*activity.MoveActivity)
	require.True(t, ok)
	assert.Less(t, int(move.Destination.X), int(memberGrid.X), "脅威から離れる方向に移動する")
}

func TestSquadPlanner_TryRandomMove(t *testing.T) {
	t.Parallel()

	t.Run("探索済みエリアがなければ移動できない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: world.Components.GridElement.Get(member)}
		_, ok := sp.tryRandomMove(world, member, ctx)
		assert.False(t, ok)
	})

	t.Run("探索済みエリア内なら移動できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		exploreAllTiles(world)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadSnapshot{Grid: world.Components.GridElement.Get(member)}
		_, ok := sp.tryRandomMove(world, member, ctx)
		assert.True(t, ok)
	})
}
