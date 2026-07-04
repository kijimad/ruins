package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// setupTestAI はテスト用のAIエンティティを作成する
func setupTestAI(t *testing.T, world w.World, x, y int, ai *gc.AI) ecs.Entity {
	t.Helper()
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.Name, &gc.Name{Name: "テストAI"})
	entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(x), Y: consts.Tile(y)})
	entity.AddComponent(world.Components.AI, ai)
	entity.AddComponent(world.Components.TurnBased, &gc.TurnBased{
		AP:    gc.IntPool{Current: 200, Max: 200},
		Speed: 100,
	})
	return entity
}

// hostileAI はテスト用の敵対AIを生成するヘルパー
func hostileAI(movement gc.MovementPolicy) *gc.AI {
	return &gc.AI{
		Planner:       gc.PlannerRoaming,
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      movement,
		ViewDistance:  5,
	}
}

func TestPlanAction_WaitingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateWaiting
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()

	// Waiting状態では待機を返す（視界外のプレイヤーでは遷移しない）
	// Plan()経由でテスト。状態遷移も含む
	behavior, params := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
	assert.Equal(t, entity, params.Actor)
}

func TestPlanAction_ChasingState_Adjacent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateChasing
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 6, 5, ai)

	rp := newRoamingPlanner()

	behavior, params := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorAttack, behavior.Name())
	assert.NotNil(t, params.Target)
}

func TestPlanAction_ChasingState_NotAdjacent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateChasing
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 10, 10, ai)

	rp := newRoamingPlanner()

	behavior, params := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	assert.NotNil(t, params.Destination)
}

func TestPlanAction_FleeingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateFleeing
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 10, 10, ai)

	rp := newRoamingPlanner()

	behavior, _ := rp.Plan(world, entity)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"逃亡時は移動か待機を返すべき: got %s", name)
}

func TestPlanAction_DrivingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()

	behavior, _ := rp.Plan(world, entity)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"Driving状態は移動か待機を返すべき: got %s", name)
}

func TestPlanAction_UnknownState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateSubState("UNKNOWN")
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()

	behavior, _ := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanDrivingAction_Stationary(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementStationary)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planDrivingAction(world, entity, ai, grid)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanDrivingAction_Wander(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementWander)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planDrivingAction(world, entity, ai, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_WallHug(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementWallHug)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planDrivingAction(world, entity, ai, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Swarm(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementSwarm)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planDrivingAction(world, entity, ai, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Territorial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementTerritorial)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	ai.SpawnX = 20
	ai.SpawnY = 20
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planDrivingAction(world, entity, ai, grid)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
}

func TestPlanDrivingAction_Random(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planDrivingAction(world, entity, ai, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Patrol(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementPatrol)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	ai.SpawnX = 20
	ai.SpawnY = 20
	ai.PatrolDirX = 1
	ai.PatrolDirY = 0
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, params := rp.planDrivingAction(world, entity, ai, grid)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	assert.Equal(t, consts.Tile(21), params.Destination.X)
	assert.Equal(t, consts.Tile(20), params.Destination.Y)
}

func TestPlanPatrolAction_ReverseOnBlock(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	wall := world.Manager.NewEntity()
	wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: 21, Y: 20})
	wall.AddComponent(world.Components.BlockPass, &gc.BlockPass{})

	ai := hostileAI(gc.MovementPatrol)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	ai.SpawnX = 20
	ai.SpawnY = 20
	ai.PatrolDirX = 1
	ai.PatrolDirY = 0
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, params := rp.planPatrolAction(world, entity, ai, grid)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	assert.Equal(t, consts.Tile(19), params.Destination.X)
	assert.Equal(t, -1, ai.PatrolDirX)
}

func TestPlanPatrolAction_BothBlocked(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	for _, x := range []int{19, 21} {
		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(x), Y: 20})
		wall.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
	}

	ai := hostileAI(gc.MovementPatrol)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	ai.SpawnX = 20
	ai.SpawnY = 20
	ai.PatrolDirX = 1
	ai.PatrolDirY = 0
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planPatrolAction(world, entity, ai, grid)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanTerritorialAction_StaysInRange(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementTerritorial)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	ai.SpawnX = 20
	ai.SpawnY = 20
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()

	for i := 0; i < 100; i++ {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

		behavior, params := rp.planTerritorialAction(world, entity, ai, grid)
		if behavior.Name() == gc.BehaviorMove && params.Destination != nil {
			grid.X = params.Destination.X
			grid.Y = params.Destination.Y
		}

		dx := int(grid.X) - ai.SpawnX
		dy := int(grid.Y) - ai.SpawnY
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		assert.LessOrEqual(t, dx, territorialRadius, "iteration %d: X within range", i)
		assert.LessOrEqual(t, dy, territorialRadius, "iteration %d: Y within range", i)
	}
}

func TestPlanTerritorialAction_AtBoundary(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementTerritorial)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	ai.SpawnX = 20
	ai.SpawnY = 20
	entity := setupTestAI(t, world, 25, 25, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	for i := 0; i < 50; i++ {
		behavior, params := rp.planTerritorialAction(world, entity, ai, grid)
		if behavior.Name() == gc.BehaviorMove && params.Destination != nil {
			dx := int(params.Destination.X) - ai.SpawnX
			dy := int(params.Destination.Y) - ai.SpawnY
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			assert.LessOrEqual(t, dx, territorialRadius, "iter %d: destination X within range", i)
			assert.LessOrEqual(t, dy, territorialRadius, "iter %d: destination Y within range", i)
		}
	}
}

func TestPlanWanderAction(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementWander)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := &gc.GridElement{X: 20, Y: 20}

	gotMove := false
	gotWait := false
	for i := 0; i < 50; i++ {
		entity := setupTestAI(t, world, 20, 20, ai)
		behavior, _ := rp.planWanderAction(world, entity, grid)
		switch behavior.Name() { //nolint:exhaustive
		case gc.BehaviorMove:
			gotMove = true
		case gc.BehaviorWait:
			gotWait = true
		}
		if gotMove && gotWait {
			break
		}
	}
	assert.True(t, gotMove, "Wanderは移動を返すことがあるべき")
	assert.True(t, gotWait, "Wanderは待機を返すことがあるべき")
}

func TestPlanWallHugAction(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	for x := 19; x <= 21; x++ {
		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(x), Y: 19})
		wall.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
	}

	ai := hostileAI(gc.MovementWallHug)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	moved := false
	for i := 0; i < 50; i++ {
		behavior, _ := rp.planWallHugAction(world, entity, grid)
		if behavior.Name() == gc.BehaviorMove {
			moved = true
			break
		}
	}
	assert.True(t, moved, "WallHugは移動を返すことがあるべき")
}

func TestPlanSwarmAction_NoAllies(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementSwarm)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := rp.planSwarmAction(world, entity, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"仲間がいない場合は移動か待機を返すべき: got %s", name)
}

func TestPlanSwarmAction_WithAlly(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementSwarm)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	allyAI := hostileAI(gc.MovementSwarm)
	ally := world.Manager.NewEntity()
	ally.AddComponent(world.Components.GridElement, &gc.GridElement{X: 25, Y: 25})
	ally.AddComponent(world.Components.AI, allyAI)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	moved := false
	for i := 0; i < 50; i++ {
		behavior, params := rp.planSwarmAction(world, entity, grid)
		if behavior.Name() == gc.BehaviorMove && params.Destination != nil {
			if params.Destination.X > grid.X || params.Destination.Y > grid.Y {
				moved = true
				break
			}
		}
	}
	assert.True(t, moved, "仲間がいる場合は接近方向に移動するべき")
}

func TestCalculateMoveCandidates(t *testing.T) {
	t.Parallel()

	t.Run("斜め方向", func(t *testing.T) {
		t.Parallel()
		candidates := calculateMoveCandidates(3, 2)
		require.NotEmpty(t, candidates)
		assert.Equal(t, 1, candidates[0].X)
		assert.Equal(t, 1, candidates[0].Y)
	})

	t.Run("水平方向のみ", func(t *testing.T) {
		t.Parallel()
		candidates := calculateMoveCandidates(-5, 0)
		require.NotEmpty(t, candidates)
		assert.Equal(t, -1, candidates[0].X)
		assert.Equal(t, 0, candidates[0].Y)
	})

	t.Run("垂直方向のみ", func(t *testing.T) {
		t.Parallel()
		candidates := calculateMoveCandidates(0, 4)
		require.NotEmpty(t, candidates)
		assert.Equal(t, 0, candidates[0].X)
		assert.Equal(t, 1, candidates[0].Y)
	})

	t.Run("差分なし", func(t *testing.T) {
		t.Parallel()
		candidates := calculateMoveCandidates(0, 0)
		assert.Empty(t, candidates)
	})
}

func TestIsAdjacent(t *testing.T) {
	t.Parallel()

	assert.True(t, isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 6, Y: 5},
	))
	assert.True(t, isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 6, Y: 6},
	))
	assert.False(t, isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 5, Y: 5},
	))
	assert.False(t, isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 7, Y: 5},
	))
}

func TestPlanRandomMoveAction(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	ai := hostileAI(gc.MovementRandom)
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = 1
	ai.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, ai)

	rp := newRoamingPlanner()
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	gotMove := false
	gotWait := false
	for i := 0; i < 50; i++ {
		behavior, _ := rp.planRandomMoveAction(world, entity, grid)
		switch behavior.Name() { //nolint:exhaustive
		case gc.BehaviorMove:
			gotMove = true
		case gc.BehaviorWait:
			gotWait = true
		}
		if gotMove && gotWait {
			break
		}
	}
	assert.True(t, gotMove, "ランダム移動は移動を返すことがあるべき")
	assert.True(t, gotWait, "ランダム移動は待機を返すことがあるべき")
}
