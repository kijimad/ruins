package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/movement"
	w "github.com/kijimaD/ruins/internal/world"
)

// MoveActivity はActivityInterfaceの実装
type MoveActivity struct{}

// Info はActivityInterfaceの実装
func (ma *MoveActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "移動",
		Description:     "隣接するタイルに移動する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 100,
		TotalRequiredAP: 100,
	}
}

// String はActivityInterfaceの実装
func (ma *MoveActivity) String() string {
	return "Move"
}

// Validate はActivityInterfaceの実装
func (ma *MoveActivity) Validate(act *Activity, world w.World) error {
	if act.Position == nil {
		return ErrMoveTargetNotSet
	}

	destX, destY := int(act.Position.X), int(act.Position.Y)
	if destX < 0 || destY < 0 {
		return ErrMoveTargetCoordInvalid
	}

	gridElement := world.Components.GridElement.Get(act.Actor)
	if gridElement == nil {
		return ErrMoveNoGridElement
	}

	if !movement.CanMoveTo(world, int(act.Position.X), int(act.Position.Y), act.Actor) {
		return ErrMoveTargetInvalid
	}

	// APが足りるかチェック
	// TODO(kijima): 装備時に俊敏性が下がりAPが足りなくなって動けなくなるが、所持重量は変わっていないし最大重量を超えるわけでもないのに動けなくなるのはおかしい
	if act.Actor.HasComponent(world.Components.TurnBased) {
		turnBased := world.Components.TurnBased.Get(act.Actor).(*gc.TurnBased)
		moveCost := ma.Info().ActionPointCost

		if turnBased.AP.Max < moveCost {
			if act.Actor.HasComponent(world.Components.Player) {
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

// Start はActivityInterfaceの実装
func (ma *MoveActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("移動開始", "actor", act.Actor, "destination", *act.Position)
	return nil
}

// DoTurn はActivityInterfaceの実装
func (ma *MoveActivity) DoTurn(act *Activity, world w.World) error {
	if act.Position == nil {
		act.Cancel("移動先が設定されていません")
		return ErrMoveTargetNotSet
	}

	// GridElementの存在確認
	gridElement := world.Components.GridElement.Get(act.Actor)
	if gridElement == nil {
		act.Cancel("移動できません（位置情報なし）")
		return ErrMoveTargetInvalid
	}

	// 移動可能かチェック
	if !movement.CanMoveTo(world, int(act.Position.X), int(act.Position.Y), act.Actor) {
		act.Cancel("移動できません")
		return ErrMoveTargetInvalid
	}

	if err := ma.performMove(act, world); err != nil {
		act.Cancel(fmt.Sprintf("移動エラー: %s", err.Error()))
		return err
	}

	act.Complete()
	return nil
}

// Finish はActivityInterfaceの実装
func (ma *MoveActivity) Finish(act *Activity, _ w.World) error {
	act.Logger.Debug("移動アクティビティ完了", "actor", act.Actor)
	return nil
}

// Canceled はActivityInterfaceの実装
func (ma *MoveActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("移動キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}

func (ma *MoveActivity) performMove(act *Activity, world w.World) error {
	gridElement := world.Components.GridElement.Get(act.Actor)
	if gridElement == nil {
		return ErrGridElementNotFound
	}

	grid := gridElement.(*gc.GridElement)
	oldX, oldY := int(grid.X), int(grid.Y)

	grid.X = gc.Tile(act.Position.X)
	grid.Y = gc.Tile(act.Position.Y)

	// TODO: 移動だけでなく、ターンを消費するすべての操作で空腹度を上げる必要がある
	if act.Actor.HasComponent(world.Components.Player) {
		if hungerComponent := world.Components.Hunger.Get(act.Actor); hungerComponent != nil {
			hunger := hungerComponent.(*gc.Hunger)
			hunger.Decrease(1)
		}
	}

	act.Logger.Debug("移動完了",
		"actor", act.Actor,
		"from", fmt.Sprintf("(%d,%d)", oldX, oldY),
		"to", fmt.Sprintf("(%.1f,%.1f)", act.Position.X, act.Position.Y))

	return nil
}
