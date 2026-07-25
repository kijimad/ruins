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
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAllyMember はFactionAllyとGridElementのみを持つ簡易隊員エンティティを作る
func newAllyMember(t *testing.T, world w.World, x, y int) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}})
	world.Components.FactionAlly.Add(e, &gc.FactionAlly{})
	return e
}

// markExploredNeighbors はgridの隣接8マスを探索済みにする
func markExploredNeighbors(world w.World, grid *gc.GridElement) {
	field := query.GetCurrentStageField(world)
	for _, d := range eightDirections {
		dest := grid.Add(d)
		field.ExploredTiles[gc.GridElement{Coord: dest}] = true
	}
}

func TestGatherSquadContext(t *testing.T) {
	t.Parallel()

	t.Run("隊員のコンテキストを正しく収集する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx, ok := sp.gatherSquadContext(world, member)
		require.True(t, ok)
		require.NotNil(t, ctx)

		assert.Equal(t, world.Components.GridElement.Get(member), ctx.Grid)
		assert.Equal(t, world.Components.SquadAI.Get(member), ctx.Squad)
		assert.Equal(t, leader, ctx.LeaderEntity)
		assert.Equal(t, world.Components.GridElement.Get(leader), ctx.LeaderGrid)
	})

	t.Run("SquadAIがなければfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := world.ECS.NewEntity()
		world.Components.GridElement.Add(member, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		sp := newSquadPlanner(newTestRNG())
		ctx, ok := sp.gatherSquadContext(world, member)
		assert.False(t, ok)
		assert.Nil(t, ctx)
	})

	t.Run("プレイヤーがいなければfalseを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Playerタグなしのダミーリーダーを使い、実在のプレイヤーを不在にする
		fakeLeader := world.ECS.NewEntity()
		world.Components.GridElement.Add(fakeLeader, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		member, err := lifecycle.SpawnSquadMember(world, fakeLeader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		ctx, ok := sp.gatherSquadContext(world, member)
		assert.False(t, ok)
		assert.Nil(t, ctx)
	})
}

func TestSquadPlanner_Plan(t *testing.T) {
	t.Parallel()

	t.Run("SquadAIがなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)

		member := world.ECS.NewEntity()
		world.Components.GridElement.Add(member, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 12, Y: 10}})

		sp := newSquadPlanner(newTestRNG())
		assert.Nil(t, sp.Plan(world, member))
	})

	t.Run("コンテキストが揃っていれば行動を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		leader, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "Ash")
		require.NoError(t, err)
		member, err := lifecycle.SpawnSquadMember(world, leader, "隊員A", testAbilities(), "player")
		require.NoError(t, err)

		sp := newSquadPlanner(newTestRNG())
		b := sp.Plan(world, member)
		require.NotNil(t, b, "デフォルトポリシーでは常に何らかの行動を返す")

		switch b.(type) {
		case *activity.WaitActivity, *activity.MoveActivity:
		default:
			require.Failf(t, "想定外の行動種別", "%T", b)
		}
	})
}

func TestSquadPlanner_ShouldRetreatLowHP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		hp   *gc.HP
		want bool
	}{
		{"HPコンポーネントがなければ後退しない", nil, false},
		{"Maxが0なら後退しない", &gc.HP{Current: 0, Max: 0}, false},
		{"HP25%なら後退する", &gc.HP{Current: 25, Max: 100}, true},
		{"HP26%なら後退しない", &gc.HP{Current: 26, Max: 100}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)

			entity := world.ECS.NewEntity()
			if tt.hp != nil {
				world.Components.HP.Add(entity, tt.hp)
			}

			sp := newSquadPlanner(newTestRNG())
			assert.Equal(t, tt.want, sp.shouldRetreatLowHP(world, entity))
		})
	}
}

func TestIsOutsideExploredArea(t *testing.T) {
	t.Parallel()

	t.Run("未探索タイルはtrue", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}

		sp := newSquadPlanner(newTestRNG())
		assert.True(t, sp.isOutsideExploredArea(world, grid))
	})

	t.Run("探索済みタイルはfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}

		field := query.GetCurrentStageField(world)
		field.ExploredTiles[*grid] = true

		sp := newSquadPlanner(newTestRNG())
		assert.False(t, sp.isOutsideExploredArea(world, grid))
	})
}

func TestPlanRetreatAction(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	member := newAllyMember(t, world, 10, 10)
	sp := newSquadPlanner(newTestRNG())
	ctx := &squadContext{
		Grid:       world.Components.GridElement.Get(member),
		LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 15, Y: 10}},
	}

	b, ok := sp.planRetreatAction(world, member, ctx)
	require.True(t, ok)
	move, ok := b.(*activity.MoveActivity)
	require.True(t, ok, "型が *activity.MoveActivity であるべき")
	newGrid := &gc.GridElement{Coord: move.Destination.Coord}
	assert.Less(t, gridDistance(newGrid, ctx.LeaderGrid), gridDistance(ctx.Grid, ctx.LeaderGrid), "リーダーに近づく")
}

func TestPlanReturnToExploredArea(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	member := newAllyMember(t, world, 10, 10)
	sp := newSquadPlanner(newTestRNG())
	ctx := &squadContext{
		Grid:       world.Components.GridElement.Get(member),
		LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 15, Y: 10}},
	}

	b, ok := sp.planReturnToExploredArea(world, member, ctx)
	require.True(t, ok)
	move, ok := b.(*activity.MoveActivity)
	require.True(t, ok, "型が *activity.MoveActivity であるべき")
	newGrid := &gc.GridElement{Coord: move.Destination.Coord}
	assert.Less(t, gridDistance(newGrid, ctx.LeaderGrid), gridDistance(ctx.Grid, ctx.LeaderGrid), "リーダーに近づく")
}

func TestPlanCombatAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		combat gc.CombatPolicy
	}{
		{"CombatAttackで視界内に敵がいなければ何もしない", gc.CombatAttack},
		{"CombatEvadeで視界内に敵がいなければ何もしない", gc.CombatEvade},
		{"未対応ポリシーは何もしない", gc.CombatIgnore},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)

			member := newAllyMember(t, world, 10, 10)
			sp := newSquadPlanner(newTestRNG())
			ctx := &squadContext{
				Grid:  world.Components.GridElement.Get(member),
				Squad: &gc.SquadAI{CombatCurrent: tt.combat, ViewDistance: 5},
			}

			_, ok := sp.planCombatAction(world, member, ctx)
			assert.False(t, ok)
		})
	}
}

func TestPlanAttackAction(t *testing.T) {
	t.Parallel()

	t.Run("隣接する敵を攻撃する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		enemy := setupTestAI(t, world, 11, 10, &gc.SoloAI{})

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:  world.Components.GridElement.Get(member),
			Squad: &gc.SquadAI{ViewDistance: 5},
		}

		b, ok := sp.planAttackAction(world, member, ctx)
		require.True(t, ok)
		attack, ok := b.(*activity.AttackActivity)
		require.True(t, ok, "型が *activity.AttackActivity であるべき")
		assert.Equal(t, enemy, attack.Target)
	})

	t.Run("視界内の離れた敵に接近する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		setupTestAI(t, world, 13, 10, &gc.SoloAI{})

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:  world.Components.GridElement.Get(member),
			Squad: &gc.SquadAI{ViewDistance: 5},
		}

		b, ok := sp.planAttackAction(world, member, ctx)
		require.True(t, ok)
		_, ok = b.(*activity.MoveActivity)
		assert.True(t, ok, "型が *activity.MoveActivity であるべき")
	})

	t.Run("視界内に敵がいなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:  world.Components.GridElement.Get(member),
			Squad: &gc.SquadAI{ViewDistance: 5},
		}

		_, ok := sp.planAttackAction(world, member, ctx)
		assert.False(t, ok)
	})
}

func TestPlanEvadeAction(t *testing.T) {
	t.Parallel()

	t.Run("敵から離れる方向へ移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		enemy := setupTestAI(t, world, 11, 10, &gc.SoloAI{})

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:  world.Components.GridElement.Get(member),
			Squad: &gc.SquadAI{ViewDistance: 5},
		}

		before := gridDistance(ctx.Grid, world.Components.GridElement.Get(enemy))

		b, ok := sp.planEvadeAction(world, member, ctx)
		require.True(t, ok)
		move, ok := b.(*activity.MoveActivity)
		require.True(t, ok, "型が *activity.MoveActivity であるべき")

		newGrid := &gc.GridElement{Coord: move.Destination.Coord}
		after := gridDistance(newGrid, world.Components.GridElement.Get(enemy))
		assert.Greater(t, after, before, "敵から遠ざかる")
	})

	t.Run("視界内に敵がいなければ何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:  world.Components.GridElement.Get(member),
			Squad: &gc.SquadAI{ViewDistance: 5},
		}

		_, ok := sp.planEvadeAction(world, member, ctx)
		assert.False(t, ok)
	})
}

func TestPlanPositionAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		movement   gc.SquadMovement
		wantReason string
	}{
		{"護衛は近距離なら待機する", gc.SquadEscort, "隊員護衛位置"},
		{"前衛は近距離かつ未探索なら移動失敗で待機する", gc.SquadVanguard, "隊員前衛移動失敗"},
		{"巡回は未探索なら移動失敗で待機する", gc.SquadPatrol, "隊員巡回移動失敗"},
		{"待機ポリシーは常に待機する", gc.SquadStationary, "隊員待機"},
		{"退避は護衛と同じ挙動をする", gc.SquadRetreat, "隊員護衛位置"},
		{"未対応ポリシーはデフォルト待機する", gc.SquadMovement(""), "隊員デフォルト待機"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)

			member := newAllyMember(t, world, 10, 10)
			sp := newSquadPlanner(newTestRNG())
			ctx := &squadContext{
				Grid:       world.Components.GridElement.Get(member),
				Squad:      &gc.SquadAI{Movement: tt.movement},
				LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}},
			}

			b := sp.planPositionAction(world, member, ctx)
			wait, ok := b.(*activity.WaitActivity)
			require.True(t, ok, "型が *activity.WaitActivity であるべき")
			assert.Equal(t, tt.wantReason, wait.Reason)
		})
	}
}

func TestPlanEscortAction(t *testing.T) {
	t.Parallel()

	t.Run("護衛距離内なら待機する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:       world.Components.GridElement.Get(member),
			LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}},
		}

		b := sp.planEscortAction(world, member, ctx)
		wait, ok := b.(*activity.WaitActivity)
		require.True(t, ok, "型が *activity.WaitActivity であるべき")
		assert.Equal(t, "隊員護衛位置", wait.Reason)
	})

	t.Run("護衛距離を超えたらリーダーに近づく", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:       world.Components.GridElement.Get(member),
			LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 15, Y: 10}},
		}

		b := sp.planEscortAction(world, member, ctx)
		move, ok := b.(*activity.MoveActivity)
		require.True(t, ok, "型が *activity.MoveActivity であるべき")
		newGrid := &gc.GridElement{Coord: move.Destination.Coord}
		assert.Less(t, gridDistance(newGrid, ctx.LeaderGrid), gridDistance(ctx.Grid, ctx.LeaderGrid))
	})
}

func TestPlanVanguardAction(t *testing.T) {
	t.Parallel()

	t.Run("前衛距離を超えたらリーダーに近づく", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:       world.Components.GridElement.Get(member),
			LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 15, Y: 10}},
		}

		b := sp.planVanguardAction(world, member, ctx)
		move, ok := b.(*activity.MoveActivity)
		require.True(t, ok, "型が *activity.MoveActivity であるべき")
		newGrid := &gc.GridElement{Coord: move.Destination.Coord}
		assert.Less(t, gridDistance(newGrid, ctx.LeaderGrid), gridDistance(ctx.Grid, ctx.LeaderGrid))
	})

	t.Run("前衛距離内かつ未探索エリアなら移動失敗で待機する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:       world.Components.GridElement.Get(member),
			LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}},
		}

		b := sp.planVanguardAction(world, member, ctx)
		wait, ok := b.(*activity.WaitActivity)
		require.True(t, ok, "型が *activity.WaitActivity であるべき")
		assert.Equal(t, "隊員前衛移動失敗", wait.Reason)
	})

	t.Run("前衛距離内かつ探索済みなら周辺を移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 25, 25)
		grid := world.Components.GridElement.Get(member)
		markExploredNeighbors(world, grid)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{
			Grid:       grid,
			LeaderGrid: &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 26, Y: 25}},
		}

		b := sp.planVanguardAction(world, member, ctx)
		move, ok := b.(*activity.MoveActivity)
		require.True(t, ok, "型が *activity.MoveActivity であるべき")

		field := query.GetCurrentStageField(world)
		assert.True(t, field.ExploredTiles[move.Destination], "探索済みの隣接マスへ移動する")
	})
}

func TestPlanSquadPatrolAction(t *testing.T) {
	t.Parallel()

	t.Run("未探索エリアなら移動失敗で待機する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 10, 10)
		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{Grid: world.Components.GridElement.Get(member)}

		b := sp.planSquadPatrolAction(world, member, ctx)
		wait, ok := b.(*activity.WaitActivity)
		require.True(t, ok, "型が *activity.WaitActivity であるべき")
		assert.Equal(t, "隊員巡回移動失敗", wait.Reason)
	})

	t.Run("探索済みエリアなら周辺を移動する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		member := newAllyMember(t, world, 25, 25)
		grid := world.Components.GridElement.Get(member)
		markExploredNeighbors(world, grid)

		sp := newSquadPlanner(newTestRNG())
		ctx := &squadContext{Grid: grid}

		b := sp.planSquadPatrolAction(world, member, ctx)
		move, ok := b.(*activity.MoveActivity)
		require.True(t, ok, "型が *activity.MoveActivity であるべき")

		field := query.GetCurrentStageField(world)
		assert.True(t, field.ExploredTiles[move.Destination], "探索済みの隣接マスへ移動する")
	})
}

func TestTryMoveToward(t *testing.T) {
	t.Parallel()

	t.Run("目標に近づく次の一歩を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		member := newAllyMember(t, world, 10, 10)

		sp := newSquadPlanner(newTestRNG())
		from := world.Components.GridElement.Get(member)
		target := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 15, Y: 10}}

		b, ok := sp.tryMoveToward(world, member, from, target)
		require.True(t, ok)
		move, ok := b.(*activity.MoveActivity)
		require.True(t, ok, "型が *activity.MoveActivity であるべき")
		newGrid := &gc.GridElement{Coord: move.Destination.Coord}
		assert.Less(t, gridDistance(newGrid, target), gridDistance(from, target))
	})

	t.Run("移動元と目標が同じ座標なら移動しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		member := newAllyMember(t, world, 10, 10)

		sp := newSquadPlanner(newTestRNG())
		from := world.Components.GridElement.Get(member)
		target := &gc.GridElement{Coord: from.Coord}

		_, ok := sp.tryMoveToward(world, member, from, target)
		assert.False(t, ok)
	})
}

func TestTryMoveAway(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	member := newAllyMember(t, world, 10, 10)

	sp := newSquadPlanner(newTestRNG())
	from := world.Components.GridElement.Get(member)
	threat := &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 11, Y: 10}}

	b, ok := sp.tryMoveAway(world, member, from, threat)
	require.True(t, ok)
	move, ok := b.(*activity.MoveActivity)
	require.True(t, ok, "型が *activity.MoveActivity であるべき")
	newGrid := &gc.GridElement{Coord: move.Destination.Coord}
	assert.Greater(t, gridDistance(newGrid, threat), gridDistance(from, threat))
}
