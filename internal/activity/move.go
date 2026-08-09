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
func CanMoveTo(world w.World, to, from consts.Coord[consts.Tile], movingEntity ecs.Entity) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}

	if to.X < 0 || to.Y < 0 || to.X >= si.MapWidth || to.Y >= si.MapHeight {
		return false
	}

	// 寒波前線の進入不可ライン（極低温ゾーン西端）以西へは移動できない。
	// 一方向の空間的強制。前線が無効な通常ダンジョンでは影響しない
	if !frontAllowsMoveTo(world, to.X) {
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

	// キャラクターがいるタイルへは、位置交換できる相手の場合のみ移動可能
	if target, ok := si.CharacterAt(to); ok {
		return CanSwapPosition(world, movingEntity, target)
	}

	return true
}

// frontAllowsMoveTo はローカル X が寒波前線の進入不可ライン以西でないかを返す。
//
// 進入不可ラインは極低温ゾーン西端 ColdZoneWest。ここより西は破棄され進入もできない。
// 極低温ゾーン自体（ライン東〜前線東端）へは進入できる。踏み込むと凍える。
// ゾーン判定は SeamlessBand のメソッドに集約している。
//
// 前線はオーバーワールド固有。帯データは遺跡進入で退避され現ステージから外れるため、現ステージが
// 帯データを持つか、すなわちオーバーワールドにいるかで先に gate する。遺跡では常に許可する。
func frontAllowsMoveTo(world w.World, localX consts.Tile) bool {
	if !query.IsOnOverworld(world) {
		return true
	}
	sb := *query.GetSeamlessBand(world)
	if !sb.Front.Active {
		return true
	}
	return !sb.Front.IsWestOfFront(sb.LocalToAbsX(localX))
}

// CanSwapPosition はmoverがtargetと位置交換できるかを判定する。
// プレイヤーだけが隊員と位置交換できる
func CanSwapPosition(world w.World, mover, target ecs.Entity) bool {
	if world.Components.Player.Has(mover) {
		return world.Components.SquadMember.Has(target)
	}
	// 隊員は他のキャラクターをブロックとして扱う。
	// 隊員同士の位置交換を許可すると、互いに交換し続けて前進できなくなる
	return false
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
		return ErrMoveTargetNotSet
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
		return ErrMoveTargetNotSet
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
		return ErrMoveTargetNotSet
	}
	if !world.Components.GridElement.Has(actor) {
		return ErrGridElementNotFound
	}
	grid := world.Components.GridElement.Get(actor)
	old := grid.Coord
	dest := p.Destination.Coord

	// 味方キャラクターのいるタイルに移動する場合、位置を入れ替える
	swapped, didSwap := swapAllyIfNeeded(world, actor, old, dest)

	grid.Coord = dest

	// 空間インデックスを増分更新する（無効化→全再構築のチャーンを避け、
	// 同一ターン内で後続のAIが移動先を正しく判定できるようにする）。
	// 入れ替えが起きた場合は相手キャラの位置(dest→old)も更新する。
	// 更新順は問わない（MoveCharacter が自分自身のときだけ from を削除するため）。
	query.UpdateCharacterPositionInIndex(world, actor, old, dest)
	if didSwap {
		query.UpdateCharacterPositionInIndex(world, swapped, dest, old)
	}

	log.Debug("move finished",
		"actor", actor,
		"from", old.String(),
		"to", dest.String())

	return nil
}

// swapAllyIfNeeded はプレイヤーが隊員のいるタイルに移動する際に位置を入れ替える。
// 入れ替えた相手と、入れ替えが発生したかを返す
func swapAllyIfNeeded(world w.World, actor ecs.Entity, from, to consts.Coord[consts.Tile]) (ecs.Entity, bool) {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return ecs.Entity{}, false
	}
	target, ok := si.CharacterAt(to)
	if !ok {
		return ecs.Entity{}, false
	}
	if !CanSwapPosition(world, actor, target) {
		return ecs.Entity{}, false
	}
	if !world.Components.GridElement.Has(target) {
		return ecs.Entity{}, false
	}
	targetGrid := world.Components.GridElement.Get(target)
	targetGrid.Coord = from

	// 位置入れ替えなので味方は actor と逆向きに動く。味方視点では to から from へ移る
	log.Debug("swapped position with ally",
		"target", target,
		"from", to.String(),
		"to", from.String())

	return target, true
}
