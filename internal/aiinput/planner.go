package aiinput

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
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

// runAPLoop はAPが残っている限りPlannerのアクションを連続実行する。
// 全AIエンティティ共通のAPループ
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

// EntityContext はAIエンティティの必要な情報をまとめる
type EntityContext struct {
	GridElement *gc.GridElement
	Vision      *gc.AIVision
	State       *gc.AIState
	Policy      *gc.AIPolicy
}

// gatherEntityContext はエンティティから必要なコンポーネントを収集する
func gatherEntityContext(world w.World, entity ecs.Entity) (*EntityContext, error) {
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

	return &EntityContext{
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

// calculateMoveCandidates はターゲットに向かう移動候補を計算する
func calculateMoveCandidates(dx, dy int) []MoveCandidate {
	ap := &DefaultActionPlanner{}
	return ap.calculateMoveCandidates(dx, dy)
}

// tryMoveCandidates は移動候補を順に試行する
func tryMoveCandidates(world w.World, entity ecs.Entity, from *gc.GridElement, candidates []MoveCandidate) (activity.Behavior, activity.ActionParams, bool) {
	ap := &DefaultActionPlanner{}
	return ap.tryMoveCandidates(world, entity, from, candidates)
}
