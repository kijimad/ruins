package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// setupTestAI はテスト用のAIエンティティを作成する
func setupTestAI(t *testing.T, world w.World, x, y int, mp gc.MovementPattern, roaming *gc.AIRoaming) ecs.Entity {
	t.Helper()
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.Name, &gc.Name{Name: "テストAI"})
	entity.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(x), Y: consts.Tile(y)})
	entity.AddComponent(world.Components.AIMoveFSM, &gc.AIMoveFSM{})
	entity.AddComponent(world.Components.AIRoaming, roaming)
	entity.AddComponent(world.Components.AIVision, &gc.AIVision{ViewDistance: 5})
	entity.AddComponent(world.Components.TurnBased, &gc.TurnBased{
		AP:    gc.IntPool{Current: 200, Max: 200},
		Speed: 100,
	})
	entity.AddComponent(world.Components.Disposition, &gc.Disposition{
		Default: gc.DispositionHostile,
		Current: gc.DispositionHostile,
	})
	entity.AddComponent(world.Components.MovementPattern, &mp)
	return entity
}

func TestPlanAction_WaitingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := worldhelper.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Vision:          world.Components.AIVision.Get(entity).(*gc.AIVision),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, params := ap.PlanAction(world, entity, player, context)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
	assert.Equal(t, entity, params.Actor)
}

func TestPlanAction_ChasingState_Adjacent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 6, 5, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Vision:          world.Components.AIVision.Get(entity).(*gc.AIVision),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, params := ap.PlanAction(world, entity, player, context)
	assert.Equal(t, gc.BehaviorAttack, behavior.Name())
	assert.NotNil(t, params.Target)
}

func TestPlanAction_ChasingState_NotAdjacent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingChasing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 10, 10, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Vision:          world.Components.AIVision.Get(entity).(*gc.AIVision),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, params := ap.PlanAction(world, entity, player, context)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	assert.NotNil(t, params.Destination)
}

func TestPlanAction_FleeingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingFleeing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 10, 10, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Vision:          world.Components.AIVision.Get(entity).(*gc.AIVision),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.PlanAction(world, entity, player, context)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"逃亡時は移動か待機を返すべき: got %s", name)
}

func TestPlanAction_DrivingState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := worldhelper.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Vision:          world.Components.AIVision.Get(entity).(*gc.AIVision),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.PlanAction(world, entity, player, context)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"Driving状態は移動か待機を返すべき: got %s", name)
}

func TestPlanAction_UnknownState(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := worldhelper.SpawnPlayer(world, 1, 1, "Ash")
	require.NoError(t, err)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingSubState("UNKNOWN"),
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Vision:          world.Components.AIVision.Get(entity).(*gc.AIVision),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.PlanAction(world, entity, player, context)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanDrivingAction_Stationary(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementStationary
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.planDrivingAction(world, entity, context)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanDrivingAction_Wander(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementWander
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.planDrivingAction(world, entity, context)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_WallHug(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementWallHug
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.planDrivingAction(world, entity, context)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Swarm(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementSwarm
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.planDrivingAction(world, entity, context)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Territorial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementTerritorial
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		SpawnX:                20,
		SpawnY:                20,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.planDrivingAction(world, entity, context)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
}

func TestPlanDrivingAction_Random(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.planDrivingAction(world, entity, context)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait)
}

func TestPlanDrivingAction_Patrol(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementPatrol
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		SpawnX:                20,
		SpawnY:                20,
		PatrolDirX:            1,
		PatrolDirY:            0,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, params := ap.planDrivingAction(world, entity, context)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	assert.Equal(t, consts.Tile(21), params.Destination.X)
	assert.Equal(t, consts.Tile(20), params.Destination.Y)
}

func TestPlanPatrolAction_ReverseOnBlock(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 壁を右隣に配置する
	wall := world.Manager.NewEntity()
	wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: 21, Y: 20})
	wall.AddComponent(world.Components.BlockPass, &gc.BlockPass{})

	mp := gc.MovementPatrol
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		SpawnX:                20,
		SpawnY:                20,
		PatrolDirX:            1,
		PatrolDirY:            0,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, params := ap.planPatrolAction(world, entity, context)
	assert.Equal(t, gc.BehaviorMove, behavior.Name())
	assert.Equal(t, consts.Tile(19), params.Destination.X)
	assert.Equal(t, -1, roaming.PatrolDirX)
}

func TestPlanPatrolAction_BothBlocked(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 両方向に壁を配置する
	for _, x := range []int{19, 21} {
		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(x), Y: 20})
		wall.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
	}

	mp := gc.MovementPatrol
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		SpawnX:                20,
		SpawnY:                20,
		PatrolDirX:            1,
		PatrolDirY:            0,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	context := &EntityContext{
		GridElement:     world.Components.GridElement.Get(entity).(*gc.GridElement),
		Roaming:         roaming,
		MovementPattern: mp,
	}

	behavior, _ := ap.planPatrolAction(world, entity, context)
	assert.Equal(t, gc.BehaviorWait, behavior.Name())
}

func TestPlanTerritorialAction_StaysInRange(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementTerritorial
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		SpawnX:                20,
		SpawnY:                20,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}

	for i := 0; i < 100; i++ {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		context := &EntityContext{
			GridElement:     grid,
			Roaming:         roaming,
			MovementPattern: mp,
		}

		behavior, params := ap.planTerritorialAction(world, entity, context)
		if behavior.Name() == gc.BehaviorMove && params.Destination != nil {
			grid.X = params.Destination.X
			grid.Y = params.Destination.Y
		}

		dx := int(grid.X) - roaming.SpawnX
		dy := int(grid.Y) - roaming.SpawnY
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

	mp := gc.MovementTerritorial
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
		SpawnX:                20,
		SpawnY:                20,
	}
	// 範囲境界にいるエンティティ
	entity := setupTestAI(t, world, 25, 25, mp, roaming)

	ap := &DefaultActionPlanner{}
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
	context := &EntityContext{
		GridElement:     grid,
		Roaming:         roaming,
		MovementPattern: mp,
	}

	for i := 0; i < 50; i++ {
		behavior, params := ap.planTerritorialAction(world, entity, context)
		if behavior.Name() == gc.BehaviorMove && params.Destination != nil {
			dx := int(params.Destination.X) - roaming.SpawnX
			dy := int(params.Destination.Y) - roaming.SpawnY
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

	mp := gc.MovementWander
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	gotMove := false
	gotWait := false
	for i := 0; i < 50; i++ {
		behavior, _ := ap.planWanderAction(world, entity, grid)
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

	// 壁を配置する
	for x := 19; x <= 21; x++ {
		wall := world.Manager.NewEntity()
		wall.AddComponent(world.Components.GridElement, &gc.GridElement{X: consts.Tile(x), Y: 19})
		wall.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
	}

	mp := gc.MovementWallHug
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	moved := false
	for i := 0; i < 50; i++ {
		behavior, _ := ap.planWallHugAction(world, entity, grid)
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

	mp := gc.MovementSwarm
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	behavior, _ := ap.planSwarmAction(world, entity, grid)
	name := behavior.Name()
	assert.True(t, name == gc.BehaviorMove || name == gc.BehaviorWait,
		"仲間がいない場合は移動か待機を返すべき: got %s", name)
}

func TestPlanSwarmAction_WithAlly(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementSwarm
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	// 離れた位置に仲間を配置する
	ally := world.Manager.NewEntity()
	ally.AddComponent(world.Components.GridElement, &gc.GridElement{X: 25, Y: 25})
	ally.AddComponent(world.Components.AIMoveFSM, &gc.AIMoveFSM{})

	ap := &DefaultActionPlanner{}
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	moved := false
	for i := 0; i < 50; i++ {
		behavior, params := ap.planSwarmAction(world, entity, grid)
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
	ap := &DefaultActionPlanner{}

	t.Run("斜め方向", func(t *testing.T) {
		t.Parallel()
		candidates := ap.calculateMoveCandidates(3, 2)
		require.NotEmpty(t, candidates)
		assert.Equal(t, 1, candidates[0].x)
		assert.Equal(t, 1, candidates[0].y)
	})

	t.Run("水平方向のみ", func(t *testing.T) {
		t.Parallel()
		candidates := ap.calculateMoveCandidates(-5, 0)
		require.NotEmpty(t, candidates)
		assert.Equal(t, -1, candidates[0].x)
		assert.Equal(t, 0, candidates[0].y)
	})

	t.Run("垂直方向のみ", func(t *testing.T) {
		t.Parallel()
		candidates := ap.calculateMoveCandidates(0, 4)
		require.NotEmpty(t, candidates)
		assert.Equal(t, 0, candidates[0].x)
		assert.Equal(t, 1, candidates[0].y)
	})

	t.Run("差分なし", func(t *testing.T) {
		t.Parallel()
		candidates := ap.calculateMoveCandidates(0, 0)
		assert.Empty(t, candidates)
	})
}

func TestIsAdjacent(t *testing.T) {
	t.Parallel()
	ap := &DefaultActionPlanner{}

	assert.True(t, ap.isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 6, Y: 5},
	))
	assert.True(t, ap.isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 6, Y: 6},
	))
	assert.False(t, ap.isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 5, Y: 5},
	))
	assert.False(t, ap.isAdjacent(
		&gc.GridElement{X: 5, Y: 5},
		&gc.GridElement{X: 7, Y: 5},
	))
}

func TestPlanRandomMoveAction(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	mp := gc.MovementRandom
	roaming := &gc.AIRoaming{
		SubState:              gc.AIRoamingDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 100,
	}
	entity := setupTestAI(t, world, 20, 20, mp, roaming)

	ap := &DefaultActionPlanner{}
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	gotMove := false
	gotWait := false
	for i := 0; i < 50; i++ {
		behavior, _ := ap.planRandomMoveAction(world, entity, grid)
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
