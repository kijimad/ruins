package aiinput

import (
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// Planner はエンティティの行動計画を担うインターフェースを表す。
// 各ターンのAPループ内で呼ばれ、次のアクションを返す
type Planner interface {
	Plan(world w.World, entity ecs.Entity) activity.Behavior
}

// maxActivitiesPerTurn は1ターン中に実行可能なアクティビティの上限を表す
const maxActivitiesPerTurn = 10

// runAPLoop はAPが残っている限りPlannerのアクションを連続実行する
func runAPLoop(world w.World, entity ecs.Entity, planner Planner, log *logger.Logger) {
	executed := 0

	for executed < maxActivitiesPerTurn {
		if entity.HasComponent(world.Components.Dead) {
			log.Debug("エンティティが死亡したため処理中断", "entity", entity)
			break
		}

		behavior := planner.Plan(world, entity)
		if behavior == nil {
			break
		}

		actionCost := behavior.Info().ActionPointCost
		tbComp := world.Components.TurnBased.Get(entity)
		if tbComp == nil || tbComp.(*gc.TurnBased).AP.Current < actionCost {
			log.Debug("AP不足", "entity", entity, "activity", behavior.Name(), "cost", actionCost)
			break
		}

		result, err := activity.Execute(behavior, entity, world)
		if err != nil {
			log.Warn("アクション実行失敗", "entity", entity, "activity", behavior.Name(), "error", err.Error())
			break
		}

		log.Debug("アクション実行", "entity", entity, "activity", behavior.Name(), "success", result.Success)
		executed++

		if !result.Success {
			break
		}
	}
}

// gridDistance は2つのGridElement間のチェビシェフ距離を返す
func gridDistance(a, b *gc.GridElement) int {
	return geometry.ChebyshevDistance(int(a.X), int(a.Y), int(b.X), int(b.Y))
}

// eightDirections は隣接8方向の座標差分を定義する
var eightDirections = []consts.Coord[int]{
	{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
	{X: -1, Y: 0}, {X: 1, Y: 0},
	{X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
}

// calculateMoveCandidates はターゲットに向かう移動候補を計算する
func calculateMoveCandidates(delta consts.Coord[int]) []consts.Coord[int] {
	var candidates []consts.Coord[int]
	dx, dy := delta.X, delta.Y

	if dx != 0 && dy != 0 {
		moveX := 1
		if dx < 0 {
			moveX = -1
		}
		moveY := 1
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, consts.Coord[int]{X: moveX, Y: moveY})

		if geometry.Abs(dx) > geometry.Abs(dy) {
			candidates = append(candidates, consts.Coord[int]{X: moveX, Y: 0})
			candidates = append(candidates, consts.Coord[int]{X: 0, Y: moveY})
		} else {
			candidates = append(candidates, consts.Coord[int]{X: 0, Y: moveY})
			candidates = append(candidates, consts.Coord[int]{X: moveX, Y: 0})
		}
	} else if dx != 0 {
		moveX := 1
		if dx < 0 {
			moveX = -1
		}
		candidates = append(candidates, consts.Coord[int]{X: moveX, Y: 0})
		candidates = append(candidates, consts.Coord[int]{X: 0, Y: 1})
		candidates = append(candidates, consts.Coord[int]{X: 0, Y: -1})
	} else if dy != 0 {
		moveY := 1
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, consts.Coord[int]{X: 0, Y: moveY})
		candidates = append(candidates, consts.Coord[int]{X: 1, Y: 0})
		candidates = append(candidates, consts.Coord[int]{X: -1, Y: 0})
	}

	return candidates
}

// tryMoveCandidates は移動候補を順に試行し、最初に移動可能な方向へ移動するアクションを返す
func tryMoveCandidates(world w.World, entity ecs.Entity, from *gc.GridElement, candidates []consts.Coord[int]) (activity.Behavior, bool) {
	fromPos := consts.Coord[int]{X: int(from.X), Y: int(from.Y)}
	for _, c := range candidates {
		dest := consts.Coord[int]{X: fromPos.X + c.X, Y: fromPos.Y + c.Y}
		if activity.CanMoveTo(world, dest, fromPos, entity) {
			return moveAction(dest), true
		}
	}
	return nil, false
}

// moveAction は指定座標への移動アクションを生成する
func moveAction(dest consts.Coord[int]) activity.Behavior {
	return &activity.MoveActivity{
		Destination: gc.GridElement{X: consts.Tile(dest.X), Y: consts.Tile(dest.Y)},
	}
}

// waitAction は待機アクションを生成する
func waitAction(reason string) activity.Behavior {
	return &activity.WaitActivity{Duration: 1, Reason: reason}
}

// shuffledEightDirections は8方向をシャッフルして返す
func shuffledEightDirections(rng *rand.Rand) []consts.Coord[int] {
	shuffled := make([]consts.Coord[int], len(eightDirections))
	copy(shuffled, eightDirections)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

// isAdjacent は2つのタイルが隣接しているかを判定する
func isAdjacent(a, b *gc.GridElement) bool {
	return geometry.IsAdjacent(int(a.X), int(a.Y), int(b.X), int(b.Y))
}
