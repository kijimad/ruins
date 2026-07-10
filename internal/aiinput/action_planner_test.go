package aiinput

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/mlange-42/ark/ecs"
)

// testRNG はテスト用の固定seed乱数生成器
var testRNG = rand.New(rand.NewPCG(0, 0))

// setupTestAI はテスト用の敵AIエンティティを作成する
func setupTestAI(t *testing.T, world w.World, x, y int, solo *gc.SoloAI) ecs.Entity {
	t.Helper()
	entity := world.World.NewEntity()
	world.Components.Name.Add(entity, &gc.Name{Name: "テストAI"})
	world.Components.GridElement.Add(entity, &gc.GridElement{X: consts.Tile(x), Y: consts.Tile(y)})
	world.Components.SoloAI.Add(entity, solo)
	world.Components.FactionEnemy.Add(entity, &gc.FactionEnemyData{})
	world.Components.TurnBased.Add(entity, &gc.TurnBased{
		AP:    gc.IntPool{Current: 200, Max: 200},
		Speed: 100,
	})
	return entity
}

func TestPlanAction_WaitingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateWaiting
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)

	// Waiting状態では待機を返す（視界外のプレイヤーでは遷移しない）
	// Plan()経由でテスト。状態遷移も含む
	behavior := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanAction_ChasingState_Adjacent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateChasing
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 6, 5, solo)

	rp := newSoloPlanner(testRNG)

	behavior := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorAttack, behavior.Name())
	attack := behavior.(*activity.AttackActivity)
	assert.NotZero(t, attack.Target)
}

func TestPlanAction_ChasingState_NotAdjacent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateChasing
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 10, 10, solo)

	rp := newSoloPlanner(testRNG)

	behavior := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	move := behavior.(*activity.MoveActivity)
	assert.NotZero(t, move.Destination)
}

func TestPlanAction_FleeingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateFleeing
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 10, 10, solo)

	rp := newSoloPlanner(testRNG)

	behavior := rp.Plan(world, entity)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"逃亡時は移動か待機を返すべき: got %s", name)
}

func TestPlanAction_DrivingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)

	behavior := rp.Plan(world, entity)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"Driving状態は移動か待機を返すべき: got %s", name)
}

func TestPlanAction_UnknownState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateSubState("UNKNOWN")
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)

	behavior := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanDrivingAction_Stationary(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloStationary,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planDrivingAction(world, entity, solo, grid)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanDrivingAction_Wander(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloWander,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planDrivingAction(world, entity, solo, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_WallHug(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloWallHug,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planDrivingAction(world, entity, solo, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Swarm(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloSwarm,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planDrivingAction(world, entity, solo, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Territorial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloTerritorial,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	solo.OriginX = 20
	solo.OriginY = 20
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planDrivingAction(world, entity, solo, grid)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
}

func TestPlanDrivingAction_Random(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planDrivingAction(world, entity, solo, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Patrol(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloPatrol,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	solo.OriginX = 20
	solo.OriginY = 20
	solo.PatrolDirX = 1
	solo.PatrolDirY = 0
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planDrivingAction(world, entity, solo, grid)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	move := behavior.(*activity.MoveActivity)
	assert.Equal(t, consts.Tile(21), move.Destination.X)
	assert.Equal(t, consts.Tile(20), move.Destination.Y)
}

func TestPlanPatrolAction_ReverseOnBlock(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	wall := world.World.NewEntity()
	world.Components.GridElement.Add(wall, &gc.GridElement{X: 21, Y: 20})
	world.Components.BlockPass.Add(wall, &gc.BlockPass{})

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloPatrol,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	solo.OriginX = 20
	solo.OriginY = 20
	solo.PatrolDirX = 1
	solo.PatrolDirY = 0
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planPatrolAction(world, entity, solo, grid)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	move := behavior.(*activity.MoveActivity)
	assert.Equal(t, consts.Tile(19), move.Destination.X)
	assert.Equal(t, -1, solo.PatrolDirX)
}

func TestPlanPatrolAction_BothBlocked(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	for _, x := range []int{19, 21} {
		wall := world.World.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{X: consts.Tile(x), Y: 20})
		world.Components.BlockPass.Add(wall, &gc.BlockPass{})
	}

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloPatrol,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	solo.OriginX = 20
	solo.OriginY = 20
	solo.PatrolDirX = 1
	solo.PatrolDirY = 0
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planPatrolAction(world, entity, solo, grid)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanTerritorialAction_StaysInRange(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloTerritorial,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	solo.OriginX = 20
	solo.OriginY = 20
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)

	for i := range 100 {
		grid := world.Components.GridElement.Get(entity)

		behavior := rp.planTerritorialAction(world, entity, solo, grid)
		if behavior.Name() == gc.BehaviorMove {
			move := behavior.(*activity.MoveActivity)
			grid.X = move.Destination.X
			grid.Y = move.Destination.Y
		}

		dx := int(grid.X) - solo.OriginX
		dy := int(grid.Y) - solo.OriginY
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

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloTerritorial,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	solo.OriginX = 20
	solo.OriginY = 20
	entity := setupTestAI(t, world, 25, 25, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	for i := range 50 {
		behavior := rp.planTerritorialAction(world, entity, solo, grid)
		if behavior.Name() == gc.BehaviorMove {
			move := behavior.(*activity.MoveActivity)
			dx := int(move.Destination.X) - solo.OriginX
			dy := int(move.Destination.Y) - solo.OriginY
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

	rp := newSoloPlanner(testRNG)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloWander,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)
	grid := world.Components.GridElement.Get(entity)

	gotMove := false
	gotWait := false
	for range 50 {
		behavior := rp.planWanderAction(world, entity, grid)
		switch behavior.Name() { //nolint:exhaustive // テストはMove/Waitの発生のみ検証するため全caseは扱わない
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
		wall := world.World.NewEntity()
		world.Components.GridElement.Add(wall, &gc.GridElement{X: consts.Tile(x), Y: 19})
		world.Components.BlockPass.Add(wall, &gc.BlockPass{})
	}

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloWallHug,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	moved := false
	for range 50 {
		behavior := rp.planWallHugAction(world, entity, grid)
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

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloSwarm,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	behavior := rp.planSwarmAction(world, entity, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"仲間がいない場合は移動か待機を返すべき: got %s", name)
}

func TestPlanSwarmAction_WithAlly(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloSwarm,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	allyAI := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloSwarm,
		ViewDistance:  5,
	}
	ally := world.World.NewEntity()
	world.Components.GridElement.Add(ally, &gc.GridElement{X: 25, Y: 25})
	world.Components.SoloAI.Add(ally, allyAI)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	moved := false
	for range 50 {
		behavior := rp.planSwarmAction(world, entity, grid)
		if behavior.Name() == gc.BehaviorMove {
			move := behavior.(*activity.MoveActivity)
			if move.Destination.X > grid.X || move.Destination.Y > grid.Y {
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
		candidates := calculateMoveCandidates(consts.Coord[int]{X: 3, Y: 2})
		require.NotEmpty(t, candidates)
		assert.Equal(t, 1, candidates[0].X)
		assert.Equal(t, 1, candidates[0].Y)
	})

	t.Run("水平方向のみ", func(t *testing.T) {
		t.Parallel()
		candidates := calculateMoveCandidates(consts.Coord[int]{X: -5, Y: 0})
		require.NotEmpty(t, candidates)
		assert.Equal(t, -1, candidates[0].X)
		assert.Equal(t, 0, candidates[0].Y)
	})

	t.Run("垂直方向のみ", func(t *testing.T) {
		t.Parallel()
		candidates := calculateMoveCandidates(consts.Coord[int]{X: 0, Y: 4})
		require.NotEmpty(t, candidates)
		assert.Equal(t, 0, candidates[0].X)
		assert.Equal(t, 1, candidates[0].Y)
	})

	t.Run("差分なし", func(t *testing.T) {
		t.Parallel()
		candidates := calculateMoveCandidates(consts.Coord[int]{X: 0, Y: 0})
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

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 20, 20, solo)

	rp := newSoloPlanner(testRNG)
	grid := world.Components.GridElement.Get(entity)

	gotMove := false
	gotWait := false
	for range 50 {
		behavior := rp.planRandomMoveAction(world, entity, grid)
		switch behavior.Name() { //nolint:exhaustive // テストはMove/Waitの発生のみ検証するため全caseは扱わない
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

func TestFindNearestHostile_プレイヤーのみ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	entity := setupTestAI(t, world, 6, 5, solo)

	rp := newSoloPlanner(testRNG)
	target := rp.findNearestHostile(world, entity)
	require.NotNil(t, target, "プレイヤーが見つかるべき")
	assert.True(t, world.Components.Player.Has(target))
}

func TestFindNearestHostile_隊員が最寄り(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, 20, 20, "Ash")
	require.NoError(t, err)

	abilities := gc.Abilities{
		Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
		Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
	}
	member, err := lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "player")
	require.NoError(t, err)
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(6)
	memberGrid.Y = consts.Tile(5)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	entity := setupTestAI(t, world, 5, 5, solo)

	rp := newSoloPlanner(testRNG)
	target := rp.findNearestHostile(world, entity)
	require.NotNil(t, target, "隊員が見つかるべき")
	assert.True(t, world.Components.SquadMember.Has(target), "最寄りの隊員が選ばれるべき")
}

func TestFindNearestHostile_敵対対象がいない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	entity := setupTestAI(t, world, 5, 5, solo)

	rp := newSoloPlanner(testRNG)
	target := rp.findNearestHostile(world, entity)
	assert.Nil(t, target)
}

func TestPlanAction_ChasingState_隊員に隣接で攻撃(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, 20, 20, "Ash")
	require.NoError(t, err)

	abilities := gc.Abilities{
		Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
		Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
	}
	member, err := lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "player")
	require.NoError(t, err)
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(6)
	memberGrid.Y = consts.Tile(5)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateChasing
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 5, 5, solo)

	rp := newSoloPlanner(testRNG)
	behavior := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorAttack, behavior.Name(), "隣接する隊員を攻撃すべき")
	attack := behavior.(*activity.AttackActivity)
	assert.NotZero(t, attack.Target)
}

func TestPlanAction_ChasingState_隊員に接近(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, 30, 30, "Ash")
	require.NoError(t, err)

	abilities := gc.Abilities{
		Vitality: gc.Ability{Base: 10}, Strength: gc.Ability{Base: 8},
		Sensation: gc.Ability{Base: 7}, Dexterity: gc.Ability{Base: 6},
		Agility: gc.Ability{Base: 9}, Defense: gc.Ability{Base: 5},
	}
	member, err := lifecycle.SpawnSquadMember(world, player, "隊員", abilities, "player")
	require.NoError(t, err)
	memberGrid := world.Components.GridElement.Get(member)
	memberGrid.X = consts.Tile(8)
	memberGrid.Y = consts.Tile(5)

	solo := &gc.SoloAI{
		CombatDefault: gc.CombatAttack,
		CombatCurrent: gc.CombatAttack,
		Movement:      gc.SoloRandom,
		ViewDistance:  5,
	}
	solo.SubState = gc.AIStateChasing
	solo.StartSubStateTurn = 1
	solo.DurationSubStateTurns = 100
	entity := setupTestAI(t, world, 5, 5, solo)

	rp := newSoloPlanner(testRNG)
	behavior := rp.Plan(world, entity)
	assert.Equal(t, gc.BehaviorMove, behavior.Name(), "離れた隊員に向かって移動すべき")
	move := behavior.(*activity.MoveActivity)
	assert.Greater(t, int(move.Destination.X), 5, "隊員方向に移動すべき")
}
