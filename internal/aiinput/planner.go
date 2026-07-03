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

// Planner はエンティティの行動計画を担うインターフェースを表す。
// 各ターンのAPループ内で呼ばれ、次のアクションを返す
type Planner interface {
	Plan(world w.World, entity ecs.Entity) (activity.Behavior, activity.ActionParams)
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

		behavior, params := planner.Plan(world, entity)
		if behavior == nil {
			break
		}

		actionCost := behavior.Info().ActionPointCost
		tbComp := world.Components.TurnBased.Get(entity)
		if tbComp == nil || tbComp.(*gc.TurnBased).AP.Current < actionCost {
			log.Debug("AP不足", "entity", entity, "activity", behavior.Name(), "cost", actionCost)
			break
		}

		result, err := activity.Execute(behavior, params, world)
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

// entityContext はAIエンティティの必要な情報をまとめて保持する
type entityContext struct {
	GridElement *gc.GridElement
	Vision      *gc.AIVision
	State       *gc.AIState
	Policy      *gc.AIPolicy
}

// gatherEntityContext はエンティティから必要なコンポーネントを収集する
func gatherEntityContext(world w.World, entity ecs.Entity) (*entityContext, error) {
	gridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

	aiVision := world.Components.AIVision.Get(entity)
	if aiVision == nil {
		return nil, &AIError{Type: "component_missing", Message: "AIVisionコンポーネントなし", Entity: &entity}
	}

	aiState := world.Components.AIState.Get(entity)
	if aiState == nil {
		return nil, &AIError{Type: "component_missing", Message: "AIStateコンポーネントなし", Entity: &entity}
	}

	var policy *gc.AIPolicy
	if p := world.Components.AIPolicy.Get(entity); p != nil {
		policy = p.(*gc.AIPolicy)
	}

	return &entityContext{
		GridElement: gridElement,
		Vision:      aiVision.(*gc.AIVision),
		State:       aiState.(*gc.AIState),
		Policy:      policy,
	}, nil
}

// findPlayer はプレイヤーエンティティを探す
func findPlayer(world w.World) *ecs.Entity {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return nil
	}
	return si.PlayerEntity
}

// gridDistance は2つのGridElement間のチェビシェフ距離を返す
func gridDistance(a, b *gc.GridElement) int {
	return geometry.ChebyshevDistance(int(a.X), int(a.Y), int(b.X), int(b.Y))
}

// eightDirections は隣接8方向の座標差分を定義する
var eightDirections = []struct{ x, y int }{
	{-1, -1}, {0, -1}, {1, -1},
	{-1, 0}, {1, 0},
	{-1, 1}, {0, 1}, {1, 1},
}

// moveCandidate は移動先の座標候補を保持する
type moveCandidate struct {
	x, y int
}

// calculateMoveCandidates はターゲットに向かう移動候補を計算する
func calculateMoveCandidates(dx, dy int) []moveCandidate {
	var candidates []moveCandidate

	if dx != 0 && dy != 0 {
		moveX := 1
		if dx < 0 {
			moveX = -1
		}
		moveY := 1
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, moveCandidate{moveX, moveY})

		if geometry.Abs(dx) > geometry.Abs(dy) {
			candidates = append(candidates, moveCandidate{moveX, 0})
			candidates = append(candidates, moveCandidate{0, moveY})
		} else {
			candidates = append(candidates, moveCandidate{0, moveY})
			candidates = append(candidates, moveCandidate{moveX, 0})
		}
	} else if dx != 0 {
		moveX := 1
		if dx < 0 {
			moveX = -1
		}
		candidates = append(candidates, moveCandidate{moveX, 0})
		candidates = append(candidates, moveCandidate{0, 1})
		candidates = append(candidates, moveCandidate{0, -1})
	} else if dy != 0 {
		moveY := 1
		if dy < 0 {
			moveY = -1
		}
		candidates = append(candidates, moveCandidate{0, moveY})
		candidates = append(candidates, moveCandidate{1, 0})
		candidates = append(candidates, moveCandidate{-1, 0})
	}

	return candidates
}

// tryMoveCandidates は移動候補を順に試行し、最初に移動可能な方向へ移動するアクションを返す
func tryMoveCandidates(world w.World, entity ecs.Entity, from *gc.GridElement, candidates []moveCandidate) (activity.Behavior, activity.ActionParams, bool) {
	fromX, fromY := int(from.X), int(from.Y)
	for _, candidate := range candidates {
		destX := fromX + candidate.x
		destY := fromY + candidate.y
		if activity.CanMoveTo(world, destX, destY, fromX, fromY, entity) {
			b, p := moveAction(entity, destX, destY)
			return b, p, true
		}
	}
	return nil, activity.ActionParams{}, false
}

// moveAction は指定座標への移動アクションを生成する
func moveAction(aiEntity ecs.Entity, destX, destY int) (activity.Behavior, activity.ActionParams) {
	dest := gc.GridElement{X: consts.Tile(destX), Y: consts.Tile(destY)}
	return &activity.MoveActivity{}, activity.ActionParams{
		Actor:       aiEntity,
		Destination: &dest,
	}
}

// waitAction は待機アクションを生成する
func waitAction(aiEntity ecs.Entity, reason string) (activity.Behavior, activity.ActionParams) {
	return &activity.WaitActivity{}, activity.ActionParams{Actor: aiEntity, Duration: 1, Reason: reason}
}

// shuffledEightDirections は8方向をシャッフルして返す
func shuffledEightDirections() []struct{ x, y int } {
	shuffled := make([]struct{ x, y int }, len(eightDirections))
	copy(shuffled, eightDirections)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}

// isAdjacent は2つのタイルが隣接しているかを判定する
func isAdjacent(a, b *gc.GridElement) bool {
	return geometry.IsAdjacent(int(a.X), int(a.Y), int(b.X), int(b.Y))
}
