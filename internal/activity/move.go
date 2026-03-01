package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// MoveActivity はBehaviorの実装
type MoveActivity struct{}

// Info はBehaviorの実装
func (ma *MoveActivity) Info() Info {
	return Info{
		Name:            "移動",
		Description:     "隣接するタイルに移動する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (ma *MoveActivity) Name() gc.BehaviorName {
	return gc.BehaviorMove
}

// Validate はBehaviorの実装
func (ma *MoveActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Destination == nil {
		return ErrMoveTargetNotSet
	}

	destX, destY := int(comp.Destination.X), int(comp.Destination.Y)
	if destX < 0 || destY < 0 {
		return ErrMoveTargetCoordInvalid
	}

	gridElement := world.Components.GridElement.Get(actor)
	if gridElement == nil {
		return ErrMoveNoGridElement
	}

	if !CanMoveTo(world, destX, destY, actor) {
		return ErrMoveTargetInvalid
	}

	// 所持重量が最大の1.5倍を超えていたら動けない
	if actor.HasComponent(world.Components.Pools) {
		pools := world.Components.Pools.Get(actor).(*gc.Pools)
		overweightLimit := pools.Weight.Max * 1.5
		if pools.Weight.Current > overweightLimit {
			if actor.HasComponent(world.Components.Player) {
				gamelog.New(gamelog.FieldLog).
					Warning("重すぎて動けない").
					Log()
			}
			// エラーではない
			return nil
		}
	}

	return nil
}

// Start はBehaviorの実装
func (ma *MoveActivity) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("移動開始", "actor", actor, "destination", *comp.Destination)
	return nil
}

// DoTurn はBehaviorの実装
func (ma *MoveActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Destination == nil {
		Cancel(comp, "移動先が設定されていません")
		return ErrMoveTargetNotSet
	}

	// GridElementの存在確認
	gridElement := world.Components.GridElement.Get(actor)
	if gridElement == nil {
		Cancel(comp, "移動できません（位置情報なし）")
		return ErrMoveTargetInvalid
	}

	// 移動可能かチェック
	if !CanMoveTo(world, int(comp.Destination.X), int(comp.Destination.Y), actor) {
		Cancel(comp, "移動できません")
		return ErrMoveTargetInvalid
	}

	if err := ma.performMove(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("移動エラー: %s", err.Error()))
		return err
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (ma *MoveActivity) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("移動アクティビティ完了", "actor", actor)

	// プレイヤーの場合のみ移動先のタイルイベントをチェック
	if comp.Destination != nil && actor.HasComponent(world.Components.Player) {
		showTileInteractionMessage(world, comp.Destination)
	}

	return nil
}

// Canceled はBehaviorの実装
func (ma *MoveActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("移動キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

func (ma *MoveActivity) performMove(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	gridElement := world.Components.GridElement.Get(actor)
	if gridElement == nil {
		return ErrGridElementNotFound
	}

	grid := gridElement.(*gc.GridElement)
	oldX, oldY := int(grid.X), int(grid.Y)

	grid.X = comp.Destination.X
	grid.Y = comp.Destination.Y

	// TODO: 移動だけでなく、ターンを消費するすべての操作で空腹度を上げる必要がある
	if actor.HasComponent(world.Components.Player) {
		if hungerComponent := world.Components.Hunger.Get(actor); hungerComponent != nil {
			hunger := hungerComponent.(*gc.Hunger)
			hunger.Decrease(1)
		}
	}

	log.Debug("移動完了",
		"actor", actor,
		"from", fmt.Sprintf("(%d,%d)", oldX, oldY),
		"to", fmt.Sprintf("(%d,%d)", comp.Destination.X, comp.Destination.Y))

	return nil
}
