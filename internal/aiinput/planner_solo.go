package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// territorialRadius はTerritorial移動パターンでスポーン地点から離れられる最大距離を定義する
const territorialRadius consts.Tile = 5

// followStartDistance は休んでいる随伴者が追いかけ始めるプレイヤーとのチェビシェフ距離
const followStartDistance = 4

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
func (rp *soloPlanner) Plan(world w.World, entity ecs.Entity) *gc.Activity {
	solo := world.Components.SoloAI.Get(entity)
	if solo == nil {
		rp.logger.Warn("no SoloAI component", "entity", entity)
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
		return waitAction()
	default:
		return waitAction()
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
		case oapi.Evade:
			rp.transitionToFleeing(solo, currentTurn)
		case oapi.Attack:
			rp.transitionToChasing(solo, currentTurn)
		case oapi.Ignore:
		}
	} else if elapsedTurns >= solo.DurationSubStateTurns {
		rp.transitionToDriving(solo, currentTurn)
	}
}

func (rp *soloPlanner) updateFromDriving(solo *gc.SoloAI, canSeePlayer bool, elapsedTurns, currentTurn consts.Turn) {
	if canSeePlayer {
		switch solo.CombatCurrent {
		case oapi.Evade:
			rp.transitionToFleeing(solo, currentTurn)
		case oapi.Attack:
			rp.transitionToChasing(solo, currentTurn)
		case oapi.Ignore:
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

func (rp *soloPlanner) planChaseAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) *gc.Activity {
	playerGrid := world.Components.GridElement.Get(playerEntity)

	if isAdjacent(aiGrid, playerGrid) {
		return activity.NewMeleeActivity(playerEntity)
	}

	dx := playerGrid.X - aiGrid.X
	dy := playerGrid.Y - aiGrid.Y

	candidates := calculateMoveCandidates(consts.Coord[consts.Tile]{X: dx, Y: dy})
	if b, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b
	}

	return waitAction()
}

func (rp *soloPlanner) planFleeAction(world w.World, aiEntity, playerEntity ecs.Entity, aiGrid *gc.GridElement) *gc.Activity {
	playerGrid := world.Components.GridElement.Get(playerEntity)

	dx := aiGrid.X - playerGrid.X
	dy := aiGrid.Y - playerGrid.Y

	candidates := calculateMoveCandidates(consts.Coord[consts.Tile]{X: dx, Y: dy})
	if b, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b
	}

	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

func (rp *soloPlanner) planRandomMoveAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) *gc.Activity {
	if rp.rng.Float64() < 0.3 {
		return waitAction()
	}

	from := aiGrid.Coord
	for _, d := range shuffledEightDirections(rp.rng) {
		dest := from.Add(d)
		if activity.CanMoveTo(world, dest, from, aiEntity) {
			return moveAction(dest)
		}
	}

	return waitAction()
}

func (rp *soloPlanner) planDrivingAction(world w.World, aiEntity ecs.Entity, solo *gc.SoloAI, grid *gc.GridElement) *gc.Activity {
	switch solo.Movement {
	case oapi.Stationary:
		return waitAction()
	case oapi.Wander:
		return rp.planWanderAction(world, aiEntity, grid)
	case oapi.WallHug:
		return rp.planWallHugAction(world, aiEntity, grid)
	case oapi.Swarm:
		return rp.planSwarmAction(world, aiEntity, grid)
	case oapi.Patrol:
		return rp.planPatrolAction(world, aiEntity, solo, grid)
	case oapi.Territorial:
		return rp.planTerritorialAction(world, aiEntity, solo, grid)
	case oapi.Follow:
		return rp.planFollowAction(world, aiEntity, solo, grid)
	default:
		return rp.planRandomMoveAction(world, aiEntity, grid)
	}
}

func (rp *soloPlanner) planWanderAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) *gc.Activity {
	if rp.rng.Float64() < 0.8 {
		return waitAction()
	}
	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

func (rp *soloPlanner) planWallHugAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) *gc.Activity {
	if rp.rng.Float64() < 0.3 {
		return waitAction()
	}

	si := query.GetSpatialIndex(world)

	type scoredDir struct {
		consts.Coord[consts.Tile]
		score int
	}
	var candidates []scoredDir

	from := aiGrid.Coord
	for _, d := range eightDirections {
		dest := from.Add(d)

		if !activity.CanMoveTo(world, dest, from, aiEntity) {
			continue
		}

		wallCount := 0
		for _, adj := range []consts.Coord[consts.Tile]{{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0}} {
			if si.IsBlockPass(dest.Add(adj)) {
				wallCount++
			}
		}
		candidates = append(candidates, scoredDir{d, wallCount})
	}

	if len(candidates) == 0 {
		return waitAction()
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

	return moveAction(consts.Coord[consts.Tile]{X: from.X + chosen.X, Y: from.Y + chosen.Y})
}

func (rp *soloPlanner) planSwarmAction(world w.World, aiEntity ecs.Entity, aiGrid *gc.GridElement) *gc.Activity {
	_, nearestGrid, nearestDist := query.FindNearestCharacter(world, aiEntity, aiGrid, func(entity ecs.Entity) bool {
		return world.Components.SoloAI.Has(entity)
	})

	if nearestGrid == nil || nearestDist <= 1 {
		return rp.planRandomMoveAction(world, aiEntity, aiGrid)
	}

	dx := nearestGrid.X - aiGrid.X
	dy := nearestGrid.Y - aiGrid.Y

	candidates := calculateMoveCandidates(consts.Coord[consts.Tile]{X: dx, Y: dy})
	if b, ok := tryMoveCandidates(world, aiEntity, aiGrid, candidates); ok {
		return b
	}

	return rp.planRandomMoveAction(world, aiEntity, aiGrid)
}

func (rp *soloPlanner) planPatrolAction(world w.World, aiEntity ecs.Entity, solo *gc.SoloAI, grid *gc.GridElement) *gc.Activity {
	from := grid.Coord

	dest := from.Add(solo.PatrolDir)
	if activity.CanMoveTo(world, dest, from, aiEntity) {
		return moveAction(dest)
	}

	solo.PatrolDir.X = -solo.PatrolDir.X
	solo.PatrolDir.Y = -solo.PatrolDir.Y

	dest = from.Add(solo.PatrolDir)
	if activity.CanMoveTo(world, dest, from, aiEntity) {
		return moveAction(dest)
	}

	return waitAction()
}

func (rp *soloPlanner) planTerritorialAction(world w.World, aiEntity ecs.Entity, solo *gc.SoloAI, grid *gc.GridElement) *gc.Activity {
	from := grid.Coord

	for _, d := range shuffledEightDirections(rp.rng) {
		dest := from.Add(d)

		dx := geometry.Abs(dest.X - solo.Origin.X)
		dy := geometry.Abs(dest.Y - solo.Origin.Y)
		if dx > territorialRadius || dy > territorialRadius {
			continue
		}

		if activity.CanMoveTo(world, dest, from, aiEntity) {
			return moveAction(dest)
		}
	}

	return waitAction()
}

// planFollowAction は休む・追いつくの2状態で随伴し、状態を SubState の待機と移動で持つ。
// 燃料ポイントが足りなければ待機の方針へ切り替える。展開中は方針を保ったまま休む。
func (rp *soloPlanner) planFollowAction(world w.World, aiEntity ecs.Entity, solo *gc.SoloAI, grid *gc.GridElement) *gc.Activity {
	if world.Components.Deployed.Has(aiEntity) {
		solo.SubState = gc.AIStateWaiting
		return waitAction()
	}
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return waitAction()
	}
	playerGrid := world.Components.GridElement.Get(player)
	if isAdjacent(grid, playerGrid) {
		solo.SubState = gc.AIStateWaiting
		return waitAction()
	}
	if solo.SubState != gc.AIStateDriving {
		if geometry.ChebyshevDistance(grid.Coord, playerGrid.Coord) < followStartDistance {
			return waitAction()
		}
		solo.SubState = gc.AIStateDriving
	}

	next, ok := activity.FindNextStep(world, aiEntity, grid.Coord, playerGrid.Coord)
	if !ok {
		return waitAction()
	}

	// AP 不足で runAPLoop が計画を捨てると燃料だけ減るので、払う前に1歩分の AP を確かめる
	if tb := world.Components.TurnBased.Get(aiEntity); tb == nil || tb.AP.Current < consts.StandardActionCost {
		return nil
	}
	if !lifecycle.ConsumeCubeFuel(world, aiEntity, query.FollowFuelCost(query.CubeWeight(world, aiEntity))) {
		solo.Movement = oapi.Stationary
		solo.SubState = gc.AIStateWaiting
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "The cube is out of fuel and stops.")).
			Log()
		return waitAction()
	}
	return moveAction(next)
}
