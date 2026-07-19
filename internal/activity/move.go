package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// CanMoveTo は指定位置に移動可能かチェックする。
// fromは移動元の座標で、斜め移動時の壁すり抜け防止に使用する
func CanMoveTo(world w.World, to, from consts.Coord[int], movingEntity ecs.Entity) bool {
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
	dx := to.X - from.X
	dy := to.Y - from.Y
	if dx != 0 && dy != 0 {
		if si.IsBlockPass(from.X+dx, from.Y) && si.IsBlockPass(from.X, from.Y+dy) {
			return false
		}
	}

	if si.IsBlockPass(to.X, to.Y) {
		return false
	}

	// キャラクターがいるタイルへは、位置交換できる相手の場合のみ移動可能
	if target, ok := si.CharacterAt(to.X, to.Y); ok {
		return CanSwapPosition(world, movingEntity, target)
	}

	return true
}

// frontAllowsMoveTo はローカル X が寒波前線の進入不可ライン以西でないかを返す。
//
// 進入不可ラインは極低温ゾーン西端 ColdZoneWest。ここより西は破棄され進入もできない。
// 極低温ゾーン自体（ライン東〜前線東端）へは進入できる。踏み込むと凍える。
// ゾーン判定は SeamlessBand のメソッドに集約している。前線が無効な通常ダンジョンでは常に許可する。
func frontAllowsMoveTo(world w.World, localX int) bool {
	sb := query.GetDungeon(world).SeamlessBand
	if !sb.Front.Active {
		return true
	}
	return !sb.Front.IsWestOfFront(sb.LocalToAbsX(consts.Tile(localX)))
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

// MoveActivity はBehaviorの実装
type MoveActivity struct {
	Destination gc.GridElement
}

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

// BuildActivity はBehaviorの実装
func (ma *MoveActivity) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	comp, err := NewActivity(ma, 1)
	if err != nil {
		return nil, err
	}
	comp.Destination = &ma.Destination
	return comp, nil
}

// Validate はBehaviorの実装
func (ma *MoveActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Destination == nil {
		return ErrMoveTargetNotSet
	}

	dest := consts.Coord[int]{X: int(comp.Destination.X), Y: int(comp.Destination.Y)}
	if dest.X < 0 || dest.Y < 0 {
		return ErrMoveTargetCoordInvalid
	}

	if !world.Components.GridElement.Has(actor) {
		return ErrMoveNoGridElement
	}
	gridElement := world.Components.GridElement.Get(actor)
	if !CanMoveTo(world, dest, consts.Coord[int]{X: int(gridElement.X), Y: int(gridElement.Y)}, actor) {
		return ErrMoveTargetInvalid
	}

	// 所持重量が最大の1.5倍を超えていたら動けない
	if world.Components.WeightCapacity.Has(actor) {
		cw := world.Components.WeightCapacity.Get(actor)
		overweightLimit := cw.Max * 1.5
		if cw.Current > overweightLimit {
			if world.Components.Player.Has(actor) {
				gamelog.New(query.GetGameLog(world)).
					Warning("重すぎて動けない").
					Log()
			}
			return ErrMoveOverweight
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
	if !world.Components.GridElement.Has(actor) {
		Cancel(comp, "移動できません（位置情報なし）")
		return ErrMoveTargetInvalid
	}
	gridElement := world.Components.GridElement.Get(actor)

	// 移動可能かチェック
	grid := gridElement
	to := consts.Coord[int]{X: int(comp.Destination.X), Y: int(comp.Destination.Y)}
	from := consts.Coord[int]{X: int(grid.X), Y: int(grid.Y)}
	if !CanMoveTo(world, to, from, actor) {
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
	if comp.Destination != nil && world.Components.Player.Has(actor) {
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
	if !world.Components.GridElement.Has(actor) {
		return ErrGridElementNotFound
	}
	gridElement := world.Components.GridElement.Get(actor)

	grid := gridElement
	oldX, oldY := int(grid.X), int(grid.Y)
	destX, destY := int(comp.Destination.X), int(comp.Destination.Y)

	// 味方キャラクターのいるタイルに移動する場合、位置を入れ替える
	swapped, didSwap := swapAllyIfNeeded(world, actor, oldX, oldY, destX, destY)

	grid.X = comp.Destination.X
	grid.Y = comp.Destination.Y

	// 空間インデックスを増分更新する（無効化→全再構築のチャーンを避け、
	// 同一ターン内で後続のAIが移動先を正しく判定できるようにする）。
	// 入れ替えが起きた場合は相手キャラの位置(dest→old)も更新する。
	// 更新順は問わない（MoveCharacter が自分自身のときだけ from を削除するため）。
	query.UpdateCharacterPositionInIndex(world, actor, oldX, oldY, destX, destY)
	if didSwap {
		query.UpdateCharacterPositionInIndex(world, swapped, destX, destY, oldX, oldY)
	}

	progressHunger(actor, world)

	log.Debug("移動完了",
		"actor", actor,
		"from", fmt.Sprintf("(%d,%d)", oldX, oldY),
		"to", fmt.Sprintf("(%d,%d)", destX, destY))

	return nil
}

// swapAllyIfNeeded はプレイヤーが隊員のいるタイルに移動する際に位置を入れ替える。
// 入れ替えた相手と、入れ替えが発生したかを返す
func swapAllyIfNeeded(world w.World, actor ecs.Entity, fromX, fromY, toX, toY int) (ecs.Entity, bool) {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return ecs.Entity{}, false
	}
	target, ok := si.CharacterAt(toX, toY)
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
	targetGrid.X = consts.Tile(fromX)
	targetGrid.Y = consts.Tile(fromY)

	log.Debug("味方と位置入れ替え",
		"target", target,
		"from", fmt.Sprintf("(%d,%d)", toX, toY),
		"to", fmt.Sprintf("(%d,%d)", fromX, fromY))

	return target, true
}
