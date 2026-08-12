package activity

import (
	"fmt"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// OpenDoorBehavior はBehaviorの実装
type OpenDoorBehavior struct{}

// Info はBehaviorの実装
func (odb *OpenDoorBehavior) Info() Info {
	return Info{
		Name:            "Open/Close Door",
		Description:     "Open a door",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (odb *OpenDoorBehavior) Name() gc.BehaviorName {
	return gc.BehaviorOpenDoor
}

// NewOpenDoorActivity は対象扉を指定して開扉アクティビティを組む。
func NewOpenDoorActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorOpenDoor, 0)
	comp.Params = &gc.OpenDoorParams{Target: target}
	return comp
}

// Validate は扉開閉アクティビティの検証を行う
func (odb *OpenDoorBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.OpenDoorParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	targetEntity := p.Target

	// ゼロ値・死亡エンティティはArkのHasでパニックするため先に弾く
	if !world.ECS.Alive(targetEntity) {
		return fmt.Errorf("target is not alive")
	}

	// Doorコンポーネントを持っているか確認
	if !world.Components.Door.Has(targetEntity) {
		return fmt.Errorf("target is not a door")
	}

	return nil
}

// Start は扉開閉開始時の処理を実行する
func (odb *OpenDoorBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("open door started", "actor", actor)
	return nil
}

// DoTurn は扉開閉アクティビティの1ターン分の処理を実行する
func (odb *OpenDoorBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.OpenDoorParams)
	if !ok {
		Cancel(comp, "door entity is not set")
		return ErrParamsTypeMismatch
	}
	targetEntity := p.Target

	if !world.Components.Door.Has(targetEntity) {
		Cancel(comp, "cannot get door component")
		return fmt.Errorf("cannot get door component")
	}
	raw := world.Components.Door.Get(targetEntity)
	doorComp := raw

	// 扉を開く
	if !doorComp.IsOpen {
		if err := lifecycle.OpenDoor(world, targetEntity); err != nil {
			Cancel(comp, fmt.Sprintf("cannot open door: %v", err))
			return err
		}

		log.Debug("door opened", "door", targetEntity)

		// 視界の更新が必要
		query.GetVisionState(world).RequestUpdate()
	}

	Complete(comp)
	return nil
}

// Finish は扉開閉完了時の処理を実行する
func (odb *OpenDoorBehavior) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("open door activity finished", "actor", actor)

	// プレイヤーの場合のみメッセージを表示
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Opened the door.")).
			Log()
	}

	return nil
}

// Canceled は扉開閉キャンセル時の処理を実行する
func (odb *OpenDoorBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("open door canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// CloseDoorBehavior はBehaviorの実装
type CloseDoorBehavior struct{}

// Info はBehaviorの実装
func (cdb *CloseDoorBehavior) Info() Info {
	return Info{
		Name:            "Close Door",
		Description:     "Close a door",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (cdb *CloseDoorBehavior) Name() gc.BehaviorName {
	return gc.BehaviorCloseDoor
}

// NewCloseDoorActivity は対象扉を指定して閉扉アクティビティを組む。
func NewCloseDoorActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorCloseDoor, 0)
	comp.Params = &gc.CloseDoorParams{Target: target}
	return comp
}

// Validate は扉閉鎖アクティビティの検証を行う
func (cdb *CloseDoorBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.CloseDoorParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	targetEntity := p.Target

	// ゼロ値・死亡エンティティはArkのHasでパニックするため先に弾く
	if !world.ECS.Alive(targetEntity) {
		return fmt.Errorf("target is not alive")
	}

	// Doorコンポーネントを持っているか確認
	if !world.Components.Door.Has(targetEntity) {
		return fmt.Errorf("target is not a door")
	}

	// 扉のマスにキャラクターかフィールドアイテムがいれば閉じられない。人や物の上に扉を閉じない。
	// 扉相互作用は隣接マスから発動するため、閉じる本人がこの判定に当たることはない
	if world.Components.GridElement.Has(targetEntity) {
		coord := world.Components.GridElement.Get(targetEntity).Coord
		if doorTileOccupied(world, coord) {
			return &UserError{Msg: query.T(world, "Something is in the doorway.")}
		}
	}

	return nil
}

// doorTileOccupied は扉の座標にキャラクターかフィールドアイテムがいるかを返す。
// キャラクターは空間インデックスの定義を再利用し、アイテムは LocationOnField で判定する
func doorTileOccupied(world w.World, coord consts.Coord[consts.Tile]) bool {
	if si := query.GetSpatialIndex(world); si != nil {
		if _, ok := si.CharacterAt(coord); ok {
			return true
		}
	}
	return slices.ContainsFunc(query.GetEntitiesAt(world, coord.X, coord.Y), world.Components.LocationOnField.Has)
}

// Start は扉閉鎖開始時の処理を実行する
func (cdb *CloseDoorBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("close door started", "actor", actor)
	return nil
}

// DoTurn は扉閉鎖アクティビティの1ターン分の処理を実行する
func (cdb *CloseDoorBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.CloseDoorParams)
	if !ok {
		Cancel(comp, "door entity is not set")
		return ErrParamsTypeMismatch
	}
	targetEntity := p.Target

	if !world.Components.Door.Has(targetEntity) {
		Cancel(comp, "cannot get door component")
		return fmt.Errorf("cannot get door component")
	}
	raw := world.Components.Door.Get(targetEntity)
	doorComp := raw

	// 扉を閉じる
	if doorComp.IsOpen {
		if err := lifecycle.CloseDoor(world, targetEntity); err != nil {
			Cancel(comp, fmt.Sprintf("cannot close door: %v", err))
			return err
		}

		log.Debug("door closed", "door", targetEntity)

		// 視界の更新が必要であることをマーク（BlockViewが変更されたため）
		query.GetVisionState(world).RequestUpdate()
	}

	Complete(comp)
	return nil
}

// Finish は扉閉鎖完了時の処理を実行する
func (cdb *CloseDoorBehavior) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("close door activity finished", "actor", actor)

	// プレイヤーの場合のみメッセージを表示
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Closed the door.")).
			Log()
	}

	return nil
}

// Canceled は扉閉鎖キャンセル時の処理を実行する
func (cdb *CloseDoorBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("close door canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}
