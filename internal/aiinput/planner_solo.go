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
	"github.com/mlange-42/ark/ecs"
)

// territorialRadius はTerritorial移動パターンでスポーン地点から離れられる最大距離を定義する
const territorialRadius = 5

// soloPlanner は敵・中立NPC用の行動計画を実装する。
// AIStateの状態遷移とSoloMovementによる移動を統合して行動を決定する
type soloPlanner struct {
	visionSystem VisionSystem
	logger       *logger.Logger
	rng          *rand.Rand
}

func newSoloPlanner(rng *rand.Rand) *soloPlanner {
	return &soloPlanner{
		visionSystem: NewVisionSystem(),
		logger:       logger.New(logger.CategoryTurn),
		rng:          rng,
	}
}

// Plan は状態遷移の評価とアクション決定を一体的に行う。
// APループ内で繰り返し呼ばれ、状態遷移は同一ターン内でべき等
func (rp *soloPlanner) Plan(world w.World, entity ecs.Entity) activity.Behavior {
	solo := world.Components.SoloAI.Get(entity)
	if solo == nil {
		rp.logger.Warn("SoloAIコンポーネントなし", "entity", entity)
		return nil
	}
	grid := world.Components.GridElement.Get(entity)

	target := rp.findNearestHostile(world, entity)
	if target == nil {
		return rp.planDrivingAction(world, entity, solo, grid)
	}

	turnNumber := query.GetTurnState(world).TurnNumber
	canSee := rp.visionSystem.CanSeeTarget(world, entity, *target, solo.ViewDistance)
	rp.updateState(solo, canSee, turnNumber)

	switch solo.SubState {
	case gc.AIStateChasing:
		return rp.planChaseAction(world, entity, *target, grid)
	case gc.AIStateFleeing:
		return rp.planFleeAction(world, entity, *target, grid)
	case gc.AIStateDriving:
		return rp.planDrivingAction(world, entity, solo, grid)
	case gc.AIStateWaiting:
		return waitAction("AI待機")
	default:
		return waitAction("AIデフォルト待機")
	}
}

// findNearestHostile は最寄りの敵対エンティティを探す。
// 視界判定は含まない。Chasing状態で視界外の対象を追い続けるため
func (rp *soloPlanner) findNearestHostile(world w.World, entity ecs.Entity) *ecs.Entity {
	grid := world.Components.GridElement.Get(entity)
	nearest, _, _ := query.FindNearestCharacter(world, entity, grid, func(target ecs.Entity) bool {
		return query.FactionRelation(world, entity, target) == query.RelationHostile
	})
	return nearest
}

// ========== 状態遷移ロジック ==========

func (rp *soloPlanner) updateState(solo *gc.SoloAI, canSeePlayer bool, currentTurn consts.Turn) {
	elapsedTurns := currentTurn - solo.StartSubStateTurn

	switch solo.SubState {
	case gc.AIStateWaiting:
		rp.updateFromWaiting(solo, canSeePlayer, elapsedTurns, currentTurn)
	case gc.AIStateDriving:
		rp.updateFromDriving(solo, canSeePlayer, elapsedTurns, currentTurn)
	case gc.AIStateChasing:
		rp.updateFromChasing(solo, canSeePlayer, elapsedTurns, currentTurn)
	case gc.AIStateFleeing:
		rp.updateFromFleeing(solo, canSeePlayer, elapsedTurns, currentTurn)
	default:
		rp.initializeToWaiting(solo, currentTurn)
	}
}

func (rp *soloPlanner) updateFromWaiting(solo *gc.SoloAI, canSeePlayer bool, elapsedTurns, currentTurn consts.Turn) {
	if canSeePlayer {
		switch solo.CombatCurrent {
		case gc.CombatEvade:
			rp.transitionToFleeing(solo, currentTurn)
		case gc.CombatAttack:
			rp.transitionToChasing(solo, currentTurn)
		case gc.CombatIgnore:
		}
	} else if elapsedTurns >= solo.DurationSubStateTurns {
		rp.transitionToDriving(solo, currentTurn)
	}
}

func (rp *soloPlanner) updateFromDriving(solo *gc.SoloAI, canSeePlayer bool, elapsedTurns, currentTurn consts.Turn) {
	if canSeePlayer {
		switch solo.CombatCurrent {
		case gc.CombatEvade:
			rp.transitionToFleeing(solo, currentTurn)
		case gc.CombatAttack:
			rp.transitionToChasing(solo, currentTurn)
		case gc.CombatIgnore:
		}
	} else if elapsedTurns >= solo.DurationSubStateTurns {
		rp.transitionToWaiting(solo, currentTurn)
	}
}

func (rp *soloPlanner) updateFromChasing(solo *gc.SoloAI, canSeePlayer bool, elapsedTurns, currentTurn consts.Turn) {
	if !canSeePlayer {
		if elapsedTurns >= 3 {
			rp.transitionToDriving(solo, currentTurn)
		}
	} else if elapsedTurns >= solo.DurationSubStateTurns {
		rp.transitionToWaiting(solo, currentTurn)
	}
}

func (rp *soloPlanner) updateFromFleeing(solo *gc.SoloAI, canSeePlayer bool, elapsedTurns, currentTurn consts.Turn) {
	if !canSeePlayer && elapsedTurns >= solo.DurationSubStateTurns {
		solo.ResetCombat()
		rp.transitionToDriving(solo, currentTurn)
	} else if canSeePlayer {
		solo.StartSubStateTurn = currentTurn
	}
}

func (rp *soloPlanner) transitionToWaiting(solo *gc.SoloAI, currentTurn consts.Turn) {
	solo.SubState = gc.AIStateWaiting
	solo.StartSubStateTurn = currentTurn
	solo.DurationSubStateTurns = consts.Turn(2 + rp.rng.IntN(4))
}

func (rp *soloPlanner) transitionToDriving(solo *gc.SoloAI, currentTurn consts.Turn) {
	solo.SubState = gc.AIStateDriving
	solo.StartSubStateTurn = currentTurn
	solo.DurationSubStateTurns = consts.Turn(3 + rp.rng.IntN(7))
}

func (rp *soloPlanner) transitionToChasing(solo *gc.SoloAI, currentTurn consts.Turn) {
	solo.SubState = gc.AIStateChasing
	solo.StartSubStateTurn = currentTurn
	solo.DurationSubStateTurns = consts.Turn(10 + rp.rng.IntN(5))
}

func (rp *soloPlanner) transitionToFleeing(solo *gc.SoloAI, currentTurn consts.Turn) {
	solo.SubState = gc.AIStateFleeing
	solo.StartSubStateTurn = currentTurn
	solo.DurationSubStateTurns = consts.Turn(5 + rp.rng.IntN(5))
}

func (rp *soloPlanner) initializeToWaiting(solo *gc.SoloAI, currentTurn consts.Turn) {
	solo.SubState = gc.AIStateWaiting
	solo.StartSubStateTurn = currentTurn
	solo.DurationSubStateTurns = consts.Turn(2 + rp.rng.IntN(3))
}

// ========== アクション計画ロジック ==========

func (rp *soloPlanner) planChaseAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) activity.Behavior {
	playerGrid := world.Components.GridElement.Get(playerEntity)

	if isAdjacent(aiGrid, playerGrid) {
		return &activity.AttackActivity{Target: playerEntity}
	}

	dx := int(playerGrid.X) - int(aiGrid.X)
	dy := int(playerGrid.Y) - int(aiGrid.Y)

	candidates := calculateMoveCandidates(consts.Coord[int]{X: dx, Y: dy})
	if b, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b
	}

	return waitAction("AI追跡失敗")
}

func (rp *soloPlanner) planFleeAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) activity.Behavior {
	playerGrid := world.Components.GridElement.Get(playerEntity)

	dx := int(aiGrid.X) - int(playerGrid.X)
	dy := int(aiGrid.Y) - int(playerGrid.Y)

	candidates := calculateMoveCandidates(consts.Coord[int]{X: dx, Y: dy})
	if b, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b
	}

	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

func (rp *soloPlanner) planRandomMoveAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) activity.Behavior {
	if rp.rng.Float64() < 0.3 {
		return waitAction("AIランダム待機")
	}

	from := consts.Coord[int]{X: int(aiGrid.X), Y: int(aiGrid.Y)}
	for _, d := range shuffledEightDirections(rp.rng) {
		dest := consts.Coord[int]{X: from.X + d.X, Y: from.Y + d.Y}
		if activity.CanMoveTo(world, dest, from, aiEntity) {
			return moveAction(dest)
		}
	}

	return waitAction("AIランダム移動失敗")
}

func (rp *soloPlanner) planDrivingAction(world w.World, aiEntity ecs.Entity, solo *gc.SoloAI, grid *gc.GridElement) activity.Behavior {
	switch solo.Movement {
	case gc.SoloStationary:
		return waitAction("AI固定待機")
	case gc.SoloWander:
		return rp.planWanderAction(world, aiEntity, grid)
	case gc.SoloWallHug:
		return rp.planWallHugAction(world, aiEntity, grid)
	case gc.SoloSwarm:
		return rp.planSwarmAction(world, aiEntity, grid)
	case gc.SoloPatrol:
		return rp.planPatrolAction(world, aiEntity, solo, grid)
	case gc.SoloTerritorial:
		return rp.planTerritorialAction(world, aiEntity, solo, grid)
	default:
		return rp.planRandomMoveAction(world, aiEntity, grid)
	}
}

func (rp *soloPlanner) planWanderAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) activity.Behavior {
	if rp.rng.Float64() < 0.8 {
		return waitAction("AI徘徊待機")
	}
	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

func (rp *soloPlanner) planWallHugAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) activity.Behavior {
	if rp.rng.Float64() < 0.3 {
		return waitAction("AI壁沿い待機")
	}

	si := query.GetSpatialIndex(world)

	type scoredDir struct {
		consts.Coord[int]
		score int
	}
	var candidates []scoredDir

	from := consts.Coord[int]{X: int(aiGrid.X), Y: int(aiGrid.Y)}
	for _, d := range eightDirections {
		dest := consts.Coord[int]{X: from.X + d.X, Y: from.Y + d.Y}

		if !activity.CanMoveTo(world, dest, from, aiEntity) {
			continue
		}

		wallCount := 0
		for _, adj := range []consts.Coord[int]{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}} {
			if si.IsBlockPass(dest.X+adj.X, dest.Y+adj.Y) {
				wallCount++
			}
		}
		candidates = append(candidates, scoredDir{consts.Coord[int]{X: d.X, Y: d.Y}, wallCount})
	}

	if len(candidates) == 0 {
		return waitAction("AI壁沿い移動失敗")
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
	chosen := tied[rp.rng.IntN(len(tied))]

	return moveAction(consts.Coord[int]{X: from.X + chosen.X, Y: from.Y + chosen.Y})
}

func (rp *soloPlanner) planSwarmAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) activity.Behavior {
	_, nearestGrid, nearestDist := query.FindNearestCharacter(world, aiEntity, aiGrid, func(entity ecs.Entity) bool {
		return world.Components.SoloAI.Has(entity) || world.Components.SquadAI.Has(entity)
	})

	if nearestGrid == nil || nearestDist <= 1 {
		return rp.planRandomMoveAction(world, aiEntity, aiGrid)
	}

	dx := int(nearestGrid.X) - int(aiGrid.X)
	dy := int(nearestGrid.Y) - int(aiGrid.Y)

	candidates := calculateMoveCandidates(consts.Coord[int]{X: dx, Y: dy})
	if b, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b
	}

	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

func (rp *soloPlanner) planPatrolAction(world w.World, aiEntity ecs.Entity, solo *gc.SoloAI, grid *gc.GridElement) activity.Behavior {
	from := consts.Coord[int]{X: int(grid.X), Y: int(grid.Y)}

	dest := consts.Coord[int]{X: from.X + solo.PatrolDir.X, Y: from.Y + solo.PatrolDir.Y}
	if activity.CanMoveTo(world, dest, from, aiEntity) {
		return moveAction(dest)
	}

	solo.PatrolDir.X = -solo.PatrolDir.X
	solo.PatrolDir.Y = -solo.PatrolDir.Y

	dest = consts.Coord[int]{X: from.X + solo.PatrolDir.X, Y: from.Y + solo.PatrolDir.Y}
	if activity.CanMoveTo(world, dest, from, aiEntity) {
		return moveAction(dest)
	}

	return waitAction("AI巡回移動失敗")
}

func (rp *soloPlanner) planTerritorialAction(world w.World, aiEntity ecs.Entity, solo *gc.SoloAI, grid *gc.GridElement) activity.Behavior {
	from := consts.Coord[int]{X: int(grid.X), Y: int(grid.Y)}

	for _, d := range shuffledEightDirections(rp.rng) {
		dest := consts.Coord[int]{X: from.X + d.X, Y: from.Y + d.Y}

		dx := geometry.Abs(dest.X - solo.Origin.X)
		dy := geometry.Abs(dest.Y - solo.Origin.Y)
		if dx > territorialRadius || dy > territorialRadius {
			continue
		}

		if activity.CanMoveTo(world, dest, from, aiEntity) {
			return moveAction(dest)
		}
	}

	return waitAction("AI縄張り移動失敗")
}
