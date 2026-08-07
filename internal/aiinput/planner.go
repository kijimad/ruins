package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/mlange-42/ark/ecs"
)

// Planner はエンティティの行動計画を担うインターフェースを表す。
// 各ターンのAPループ内で呼ばれ、次のアクションを返す
type Planner interface {
	Plan(world w.World, entity ecs.Entity) *gc.Activity
}

// maxActivitiesPerTurn は1ターン中に実行可能なアクティビティの上限を表す
const maxActivitiesPerTurn = 10

// runAPLoop はAPが残っている限りPlannerのアクションを連続実行する
func runAPLoop(world w.World, entity ecs.Entity, planner Planner, log *logger.Logger) {
	executed := 0

	for executed < maxActivitiesPerTurn {
		if world.Components.Dead.Has(entity) {
			log.Debug("entity is dead, aborting", "entity", entity)
			break
		}

		comp := planner.Plan(world, entity)
		if comp == nil {
			break
		}

		b, err := activity.GetBehavior(comp.BehaviorName)
		if err != nil {
			log.Warn("failed to get behavior", "entity", entity, "activity", comp.BehaviorName, "error", err.Error())
			break
		}
		actionCost := b.Info().ActionPointCost
		tbComp := world.Components.TurnBased.Get(entity)
		if tbComp == nil || tbComp.AP.Current < actionCost {
			log.Debug("insufficient AP", "entity", entity, "activity", comp.BehaviorName, "cost", actionCost)
			break
		}

		result, err := activity.Execute(comp, entity, world)
		if err != nil {
			log.Warn("failed to execute action", "entity", entity, "activity", comp.BehaviorName, "error", err.Error())
			break
		}

		log.Debug("action executed", "entity", entity, "activity", comp.BehaviorName, "success", result.Success)
		executed++

		if !result.Success {
			break
		}
	}
}

// gridDistance は2つのGridElement間のチェビシェフ距離を返す
func gridDistance(a, b *gc.GridElement) int {
	return geometry.ChebyshevDistance(a.Coord, b.Coord)
}

// eightDirections は隣接8方向の座標差分を定義する
var eightDirections = []consts.Coord[consts.Tile]{
	{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
	{X: -1, Y: 0}, {X: 1, Y: 0},
	{X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
}

// calculateMoveCandidates はターゲットに向かう移動候補を計算する
func calculateMoveCandidates(delta consts.Coord[consts.Tile]) []consts.Coord[consts.Tile] {
	var candidates []consts.Coord[consts.Tile]
	dx, dy := delta.X, delta.Y

	switch {
	case dx != 0 && dy != 0:
		moveX := consts.Tile(1)
		if dx < 0 {
			moveX = -1
		}
		moveY := consts.Tile(1)
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, consts.Coord[consts.Tile]{X: moveX, Y: moveY})

		if geometry.Abs(dx) > geometry.Abs(dy) {
			candidates = append(candidates, consts.Coord[consts.Tile]{X: moveX, Y: 0})
			candidates = append(candidates, consts.Coord[consts.Tile]{X: 0, Y: moveY})
		} else {
			candidates = append(candidates, consts.Coord[consts.Tile]{X: 0, Y: moveY})
			candidates = append(candidates, consts.Coord[consts.Tile]{X: moveX, Y: 0})
		}
	case dx != 0:
		moveX := consts.Tile(1)
		if dx < 0 {
			moveX = -1
		}
		candidates = append(candidates, consts.Coord[consts.Tile]{X: moveX, Y: 0})
		candidates = append(candidates, consts.Coord[consts.Tile]{X: 0, Y: 1})
		candidates = append(candidates, consts.Coord[consts.Tile]{X: 0, Y: -1})
	case dy != 0:
		moveY := consts.Tile(1)
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, consts.Coord[consts.Tile]{X: 0, Y: moveY})
		candidates = append(candidates, consts.Coord[consts.Tile]{X: 1, Y: 0})
		candidates = append(candidates, consts.Coord[consts.Tile]{X: -1, Y: 0})
	}

	return candidates
}

// tryMoveCandidates は移動候補を順に試行し、最初に移動可能な方向へ移動するアクションを返す
func tryMoveCandidates(world w.World, entity ecs.Entity, from *gc.GridElement, candidates []consts.Coord[consts.Tile]) (*gc.Activity, bool) {
	fromPos := from.Coord
	for _, c := range candidates {
		dest := fromPos.Add(c)
		if activity.CanMoveTo(world, dest, fromPos, entity) {
			return moveAction(dest), true
		}
	}
	return nil, false
}

// moveAction は指定座標への移動アクションを生成する
func moveAction(dest consts.Coord[consts.Tile]) *gc.Activity {
	return activity.NewMoveActivity(gc.GridElement{Coord: dest})
}

// waitAction は待機アクションを生成する
func waitAction() *gc.Activity {
	return activity.NewWaitActivity(1)
}

// shuffledEightDirections は8方向をシャッフルして返す
func shuffledEightDirections(rng *rand.Rand) []consts.Coord[consts.Tile] {
	shuffled := make([]consts.Coord[consts.Tile], len(eightDirections))
	copy(shuffled, eightDirections)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

// isAdjacent は2つのタイルが隣接しているかを判定する
func isAdjacent(a, b *gc.GridElement) bool {
	return geometry.IsAdjacent(a.Coord, b.Coord)
}
