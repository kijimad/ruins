package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// territorialRadius はTerritorial移動パターンでスポーン地点から離れられる最大距離を定義する
const territorialRadius = 5

// roamingPlanner は敵・中立NPC用の行動計画を実装する。
// AIStateの状態遷移とMovementPolicyによる移動を統合して行動を決定する
type roamingPlanner struct {
	visionSystem VisionSystem
	logger       *logger.Logger
}

func newRoamingPlanner() *roamingPlanner {
	return &roamingPlanner{
		visionSystem: NewVisionSystem(),
		logger:       logger.New(logger.CategoryTurn),
	}
}

// Plan は状態遷移の評価とアクション決定を一体的に行う。
// APループ内で繰り返し呼ばれ、状態遷移は同一ターン内でべき等
func (rp *roamingPlanner) Plan(world w.World, entity ecs.Entity) (activity.Behavior, activity.ActionParams) {
	aiComp := world.Components.AI.Get(entity)
	if aiComp == nil {
		rp.logger.Warn("AIコンポーネントなし", "entity", entity)
		return nil, activity.ActionParams{}
	}
	ai := aiComp.(*gc.AI)
	grid := world.Components.GridElement.Get(entity).(*gc.GridElement)

	playerEntity := findPlayer(world)
	if playerEntity == nil {
		return nil, activity.ActionParams{}
	}
	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil, activity.ActionParams{}
	}

	canSeePlayer := rp.visionSystem.CanSeeTarget(world, entity, *playerEntity, ai)
	turnNumber := query.GetTurnState(world).TurnNumber

	rp.updateState(ai, canSeePlayer, turnNumber)

	switch ai.SubState {
	case gc.AIStateChasing:
		return rp.planChaseAction(world, entity, *playerEntity, grid)
	case gc.AIStateFleeing:
		return rp.planFleeAction(world, entity, *playerEntity, grid)
	case gc.AIStateDriving:
		return rp.planDrivingAction(world, entity, ai, grid)
	case gc.AIStateWaiting:
		return waitAction(entity, "AI待機")
	default:
		return waitAction(entity, "AIデフォルト待機")
	}
}

// ========== 状態遷移ロジック ==========

// updateState はAIの状態を更新する。Plan()内で毎回呼ばれるが同一ターン内ではべき等
func (rp *roamingPlanner) updateState(ai *gc.AI, canSeePlayer bool, currentTurn int) {
	elapsedTurns := currentTurn - ai.StartSubStateTurn

	switch ai.SubState {
	case gc.AIStateWaiting:
		rp.updateFromWaiting(ai, canSeePlayer, elapsedTurns, currentTurn)
	case gc.AIStateDriving:
		rp.updateFromDriving(ai, canSeePlayer, elapsedTurns, currentTurn)
	case gc.AIStateChasing:
		rp.updateFromChasing(ai, canSeePlayer, elapsedTurns, currentTurn)
	case gc.AIStateFleeing:
		rp.updateFromFleeing(ai, canSeePlayer, elapsedTurns, currentTurn)
	default:
		rp.initializeToWaiting(ai, currentTurn)
	}
}

// shouldChase はAIの方針に基づいて追跡すべきかを判定する
func shouldChase(ai *gc.AI) bool {
	return ai.CombatCurrent == gc.CombatAttack
}

// shouldFlee はAIの方針に基づいて逃亡すべきかを判定する
func shouldFlee(ai *gc.AI) bool {
	return ai.CombatCurrent == gc.CombatEvade
}

func (rp *roamingPlanner) updateFromWaiting(ai *gc.AI, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if canSeePlayer {
		if shouldFlee(ai) {
			rp.transitionToFleeing(ai, currentTurn)
		} else if shouldChase(ai) {
			rp.transitionToChasing(ai, currentTurn)
		}
	} else if elapsedTurns >= ai.DurationSubStateTurns {
		rp.transitionToDriving(ai, currentTurn)
	}
}

func (rp *roamingPlanner) updateFromDriving(ai *gc.AI, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if canSeePlayer {
		if shouldFlee(ai) {
			rp.transitionToFleeing(ai, currentTurn)
		} else if shouldChase(ai) {
			rp.transitionToChasing(ai, currentTurn)
		}
	} else if elapsedTurns >= ai.DurationSubStateTurns {
		rp.transitionToWaiting(ai, currentTurn)
	}
}

func (rp *roamingPlanner) updateFromChasing(ai *gc.AI, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if !canSeePlayer {
		if elapsedTurns >= 3 {
			rp.transitionToDriving(ai, currentTurn)
		}
	} else if elapsedTurns >= ai.DurationSubStateTurns {
		rp.transitionToWaiting(ai, currentTurn)
	}
}

func (rp *roamingPlanner) updateFromFleeing(ai *gc.AI, canSeePlayer bool, elapsedTurns, currentTurn int) {
	if !canSeePlayer && elapsedTurns >= ai.DurationSubStateTurns {
		ai.ResetCombat()
		rp.transitionToDriving(ai, currentTurn)
	} else if canSeePlayer {
		ai.StartSubStateTurn = currentTurn
	}
}

func (rp *roamingPlanner) transitionToWaiting(ai *gc.AI, currentTurn int) {
	ai.SubState = gc.AIStateWaiting
	ai.StartSubStateTurn = currentTurn
	ai.DurationSubStateTurns = 2 + rand.IntN(4)
}

func (rp *roamingPlanner) transitionToDriving(ai *gc.AI, currentTurn int) {
	ai.SubState = gc.AIStateDriving
	ai.StartSubStateTurn = currentTurn
	ai.DurationSubStateTurns = 3 + rand.IntN(7)
}

func (rp *roamingPlanner) transitionToChasing(ai *gc.AI, currentTurn int) {
	ai.SubState = gc.AIStateChasing
	ai.StartSubStateTurn = currentTurn
	ai.DurationSubStateTurns = 10 + rand.IntN(5)
}

func (rp *roamingPlanner) transitionToFleeing(ai *gc.AI, currentTurn int) {
	ai.SubState = gc.AIStateFleeing
	ai.StartSubStateTurn = currentTurn
	ai.DurationSubStateTurns = 5 + rand.IntN(5)
}

func (rp *roamingPlanner) initializeToWaiting(ai *gc.AI, currentTurn int) {
	ai.SubState = gc.AIStateWaiting
	ai.StartSubStateTurn = currentTurn
	ai.DurationSubStateTurns = 2 + rand.IntN(3)
}

// ========== アクション計画ロジック ==========

// planChaseAction はプレイヤー追跡アクションを計画する
func (rp *roamingPlanner) planChaseAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)

	if isAdjacent(aiGrid, playerGrid) {
		return &activity.AttackActivity{}, activity.ActionParams{
			Actor:  aiEntity,
			Target: &playerEntity,
		}
	}

	dx := int(playerGrid.X) - int(aiGrid.X)
	dy := int(playerGrid.Y) - int(aiGrid.Y)

	candidates := calculateMoveCandidates(dx, dy)
	if b, p, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b, p
	}

	return waitAction(aiEntity, "AI追跡失敗")
}

// planFleeAction はプレイヤーから逃亡するアクションを計画する
func (rp *roamingPlanner) planFleeAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	playerGrid := world.Components.GridElement.Get(playerEntity).(*gc.GridElement)

	dx := int(aiGrid.X) - int(playerGrid.X)
	dy := int(aiGrid.Y) - int(playerGrid.Y)

	candidates := calculateMoveCandidates(dx, dy)
	if b, p, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b, p
	}

	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

// planRandomMoveAction はランダム移動アクションを計画する
func (rp *roamingPlanner) planRandomMoveAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	if rand.Float64() < 0.3 {
		return waitAction(aiEntity, "AIランダム待機")
	}

	fromX, fromY := int(aiGrid.X), int(aiGrid.Y)
	for _, d := range shuffledEightDirections() {
		destX := fromX + d.X
		destY := fromY + d.Y
		if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
			return moveAction(aiEntity, consts.Coord[int]{X: destX, Y: destY})
		}
	}

	return waitAction(aiEntity, "AIランダム移動失敗")
}

// planDrivingAction はMovementPolicyに基づく移動アクションを計画する
func (rp *roamingPlanner) planDrivingAction(world w.World, aiEntity ecs.Entity, ai *gc.AI, grid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	switch ai.Movement {
	case gc.MovementStationary:
		return waitAction(aiEntity, "AI固定待機")
	case gc.MovementWander:
		return rp.planWanderAction(world, aiEntity, grid)
	case gc.MovementWallHug:
		return rp.planWallHugAction(world, aiEntity, grid)
	case gc.MovementSwarm:
		return rp.planSwarmAction(world, aiEntity, grid)
	case gc.MovementPatrol:
		return rp.planPatrolAction(world, aiEntity, ai, grid)
	case gc.MovementTerritorial:
		return rp.planTerritorialAction(world, aiEntity, ai, grid)
	default:
		return rp.planRandomMoveAction(world, aiEntity, grid)
	}
}

// planWanderAction は低頻度でランダム移動するアクションを計画する
func (rp *roamingPlanner) planWanderAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	if rand.Float64() < 0.8 {
		return waitAction(aiEntity, "AI徘徊待機")
	}
	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

// planWallHugAction は壁に隣接するタイルを優先して移動するアクションを計画する
func (rp *roamingPlanner) planWallHugAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	if rand.Float64() < 0.3 {
		return waitAction(aiEntity, "AI壁沿い待機")
	}

	si := query.GetSpatialIndex(world)

	type scoredDir struct {
		consts.Coord[int]
		score int
	}
	var candidates []scoredDir

	fromX, fromY := int(aiGrid.X), int(aiGrid.Y)
	for _, d := range eightDirections {
		destX := fromX + d.X
		destY := fromY + d.Y

		if !activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
			continue
		}

		wallCount := 0
		for _, adj := range []consts.Coord[int]{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}} {
			if si.IsBlockPass(destX+adj.X, destY+adj.Y) {
				wallCount++
			}
		}
		candidates = append(candidates, scoredDir{consts.Coord[int]{X: d.X, Y: d.Y}, wallCount})
	}

	if len(candidates) == 0 {
		return waitAction(aiEntity, "AI壁沿い移動失敗")
	}

	best := candidates[0].score
	for _, c := range candidates[1:] {
		if c.score > best {
			best = c.score
		}
	}
	var tied []scoredDir
	for _, c := range candidates {
		if c.score == best {
			tied = append(tied, c)
		}
	}
	chosen := tied[rand.IntN(len(tied))]

	return moveAction(aiEntity, consts.Coord[int]{X: fromX + chosen.X, Y: fromY + chosen.Y})
}

// planSwarmAction は最寄りのAIエンティティに接近するアクションを計画する
func (rp *roamingPlanner) planSwarmAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	var nearestGrid *gc.GridElement
	nearestDist := -1

	world.Manager.Join(
		world.Components.AI,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity == aiEntity {
			return
		}
		if entity.HasComponent(world.Components.Dead) {
			return
		}

		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		dist := geometry.Abs(int(grid.X)-int(aiGrid.X)) + geometry.Abs(int(grid.Y)-int(aiGrid.Y))
		if nearestDist < 0 || dist < nearestDist {
			nearestDist = dist
			nearestGrid = grid
		}
	}))

	if nearestGrid == nil || nearestDist <= 1 {
		return rp.planRandomMoveAction(world, aiEntity, aiGrid)
	}

	dx := int(nearestGrid.X) - int(aiGrid.X)
	dy := int(nearestGrid.Y) - int(aiGrid.Y)

	candidates := calculateMoveCandidates(dx, dy)
	if b, p, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b, p
	}

	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

// planPatrolAction は一方向に直進し、進めなくなったら反転する巡回アクションを計画する
func (rp *roamingPlanner) planPatrolAction(world w.World, aiEntity ecs.Entity, ai *gc.AI, grid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	fromX, fromY := int(grid.X), int(grid.Y)

	destX := fromX + ai.PatrolDirX
	destY := fromY + ai.PatrolDirY
	if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
		return moveAction(aiEntity, consts.Coord[int]{X: destX, Y: destY})
	}

	ai.PatrolDirX = -ai.PatrolDirX
	ai.PatrolDirY = -ai.PatrolDirY

	destX = fromX + ai.PatrolDirX
	destY = fromY + ai.PatrolDirY
	if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
		return moveAction(aiEntity, consts.Coord[int]{X: destX, Y: destY})
	}

	return waitAction(aiEntity, "AI巡回移動失敗")
}

// planTerritorialAction はスポーン地点から一定範囲内でランダム移動するアクションを計画する
func (rp *roamingPlanner) planTerritorialAction(world w.World, aiEntity ecs.Entity, ai *gc.AI, grid *gc.GridElement) (activity.Behavior, activity.ActionParams) {
	fromX, fromY := int(grid.X), int(grid.Y)

	for _, d := range shuffledEightDirections() {
		destX := fromX + d.X
		destY := fromY + d.Y

		dx := geometry.Abs(destX - ai.SpawnX)
		dy := geometry.Abs(destY - ai.SpawnY)
		if dx > territorialRadius || dy > territorialRadius {
			continue
		}

		if activity.CanMoveTo(world, destX, destY, fromX, fromY, aiEntity) {
			return moveAction(aiEntity, consts.Coord[int]{X: destX, Y: destY})
		}
	}

	return waitAction(aiEntity, "AI縄張り移動失敗")
}
