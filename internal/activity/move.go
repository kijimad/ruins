package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// CanMoveTo は指定位置に移動可能かチェックする。
// fromは移動元の座標で、斜め移動時の壁すり抜け防止に使用する
func CanMoveTo(world w.World, to, from consts.Coord[consts.Tile], _ ecs.Entity) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}

	if to.X < 0 || to.Y < 0 || to.X >= si.MapWidth || to.Y >= si.MapHeight {
		return false
	}

	// 斜め移動の場合、隣接する直交2方向が両方ブロックされていれば通行不可
	d := to.Sub(from)
	if d.X != 0 && d.Y != 0 {
		if si.IsBlockPass(from.Add(consts.Coord[consts.Tile]{X: d.X})) && si.IsBlockPass(from.Add(consts.Coord[consts.Tile]{Y: d.Y})) {
			return false
		}
	}

	if si.IsBlockPass(to) {
		return false
	}

	// キャラクターがいるタイルへは移動できない
	if _, ok := si.CharacterAt(to); ok {
		return false
	}

	return true
}

// MoveBehavior はBehaviorの実装
type MoveBehavior struct{}

// Info はBehaviorの実装
func (mb *MoveBehavior) Info() Info {
	return Info{
		Name:            "Move",
		Description:     "Move to an adjacent tile",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (mb *MoveBehavior) Name() gc.BehaviorName {
	return gc.BehaviorMove
}

// NewMoveActivity は移動先を指定して移動アクティビティを組む。
func NewMoveActivity(destination gc.GridElement) *gc.Activity {
	comp := NewActivity(gc.BehaviorMove, 0)
	comp.Params = &gc.MoveParams{Destination: destination}
	return comp
}

// Validate はBehaviorの実装
func (mb *MoveBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.MoveParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	if p.Destination.X < 0 || p.Destination.Y < 0 {
		return fmt.Errorf("destination coordinate is invalid")
	}

	if !world.Components.GridElement.Has(actor) {
		return fmt.Errorf("GridElement not found on moving actor")
	}
	gridElement := world.Components.GridElement.Get(actor)
	if !CanMoveTo(world, p.Destination.Coord, gridElement.Coord, actor) {
		return fmt.Errorf("destination is not movable")
	}

	// 所持重量が最大の1.5倍を超えていたら動けない
	if world.Components.WeightCapacity.Has(actor) {
		cw := world.Components.WeightCapacity.Get(actor)
		overweightLimit := cw.Max * 3 / 2
		if cw.Current > overweightLimit {
			return &UserError{Msg: query.T(world, "Too heavy to move")}
		}
	}

	return nil
}

// Start はBehaviorの実装
func (mb *MoveBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	if p, ok := comp.Params.(*gc.MoveParams); ok {
		log.Debug("move started", "actor", actor, "destination", p.Destination)
	}
	return nil
}

// DoTurn はBehaviorの実装
func (mb *MoveBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.MoveParams)
	if !ok {
		Cancel(comp, "move destination is not set")
		return ErrParamsTypeMismatch
	}

	// GridElementの存在確認
	if !world.Components.GridElement.Has(actor) {
		Cancel(comp, "cannot move (no position)")
		return ErrMoveTargetInvalid
	}
	gridElement := world.Components.GridElement.Get(actor)

	// 移動可能かチェック
	grid := gridElement
	to := p.Destination.Coord
	from := grid.Coord
	if !CanMoveTo(world, to, from, actor) {
		Cancel(comp, "cannot move")
		return ErrMoveTargetInvalid
	}

	if err := mb.performMove(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("move error: %s", err.Error()))
		return err
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (mb *MoveBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("move activity finished", "actor", actor)

	// プレイヤーの場合のみ移動先のタイルイベントをチェック
	if p, ok := comp.Params.(*gc.MoveParams); ok && world.Components.Player.Has(actor) {
		showTileInteractionMessage(world, &p.Destination)
	}

	return nil
}

// Canceled はBehaviorの実装
func (mb *MoveBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("move canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

func (mb *MoveBehavior) performMove(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.MoveParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	if !world.Components.GridElement.Has(actor) {
		return ErrGridElementNotFound
	}
	grid := world.Components.GridElement.Get(actor)
	old := grid.Coord
	dest := p.Destination.Coord

	grid.Coord = dest

	// 空間インデックスを増分更新する（無効化→全再構築のチャーンを避け、
	// 同一ターン内で後続のAIが移動先を正しく判定できるようにする）。
	query.UpdateCharacterPositionInIndex(world, actor, old, dest)

	log.Debug("move finished",
		"actor", actor,
		"from", old.String(),
		"to", dest.String())

	return nil
}
