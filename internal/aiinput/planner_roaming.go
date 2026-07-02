package aiinput

import (
	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// roamingPlanner は敵・中立NPC用の行動計画を実装する。
// AIStateの状態遷移サイクルに基づいて行動を決定する
type roamingPlanner struct {
	actionPlanner ActionPlanner
	logger        *logger.Logger
}

func newRoamingPlanner() *roamingPlanner {
	return &roamingPlanner{
		actionPlanner: NewActionPlanner(),
		logger:        logger.New(logger.CategoryTurn),
	}
}

// Plan はEntityContextを収集し、ActionPlannerに行動決定を委譲する
func (rp *roamingPlanner) Plan(world w.World, entity ecs.Entity) (activity.Behavior, activity.ActionParams) {
	context, err := gatherEntityContext(world, entity)
	if err != nil {
		rp.logger.Warn("コンテキスト取得失敗", "entity", entity, "error", err.Error())
		return nil, activity.ActionParams{}
	}

	playerEntity := findPlayer(world)
	if playerEntity == nil {
		return nil, activity.ActionParams{}
	}

	if !playerEntity.HasComponent(world.Components.GridElement) {
		return nil, activity.ActionParams{}
	}

	return rp.actionPlanner.PlanAction(world, entity, *playerEntity, context)
}
