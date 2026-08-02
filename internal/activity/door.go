package activity

import (
	"fmt"

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
		Name:            "扉開閉",
		Description:     "扉を開く",
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
	comp := newActivity(gc.BehaviorOpenDoor, 0)
	comp.Target = &target
	return comp
}

// Validate は扉開閉アクティビティの検証を行う
func (odb *OpenDoorBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("扉エンティティが指定されていません")
	}

	targetEntity := *comp.Target

	// ゼロ値・死亡エンティティはArkのHasでパニックするため先に弾く
	if !world.ECS.Alive(targetEntity) {
		return fmt.Errorf("対象エンティティは扉ではありません")
	}

	// Doorコンポーネントを持っているか確認
	if !world.Components.Door.Has(targetEntity) {
		return fmt.Errorf("対象エンティティは扉ではありません")
	}

	return nil
}

// Start は扉開閉開始時の処理を実行する
func (odb *OpenDoorBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("扉開閉開始", "actor", actor)
	return nil
}

// DoTurn は扉開閉アクティビティの1ターン分の処理を実行する
func (odb *OpenDoorBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	targetEntity := *comp.Target

	if !world.Components.Door.Has(targetEntity) {
		Cancel(comp, "扉コンポーネントが取得できません")
		return fmt.Errorf("扉コンポーネントが取得できません")
	}
	raw := world.Components.Door.Get(targetEntity)
	doorComp := raw

	if doorComp.Locked {
		gamelog.New(query.GetGameLog(world)).
			Append("扉はロックされている。").
			Log()
		Cancel(comp, "扉はロックされている")
		return nil
	}

	// 扉を開く
	if !doorComp.IsOpen {
		if err := lifecycle.OpenDoor(world, targetEntity); err != nil {
			Cancel(comp, fmt.Sprintf("扉を開けません: %v", err))
			return err
		}

		log.Debug("扉を開きました", "door", targetEntity)

		// 視界の更新が必要
		query.GetVisionState(world).RequestUpdate()
	}

	Complete(comp)
	return nil
}

// Finish は扉開閉完了時の処理を実行する
func (odb *OpenDoorBehavior) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("扉開閉アクティビティ完了", "actor", actor)

	// プレイヤーの場合のみメッセージを表示
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Append("扉を開いた。").
			Log()
	}

	return nil
}

// Canceled は扉開閉キャンセル時の処理を実行する
func (odb *OpenDoorBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("扉開閉キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// CloseDoorBehavior はBehaviorの実装
type CloseDoorBehavior struct{}

// Info はBehaviorの実装
func (cdb *CloseDoorBehavior) Info() Info {
	return Info{
		Name:            "扉閉鎖",
		Description:     "扉を閉じる",
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
	comp := newActivity(gc.BehaviorCloseDoor, 0)
	comp.Target = &target
	return comp
}

// Validate は扉閉鎖アクティビティの検証を行う
func (cdb *CloseDoorBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("扉エンティティが指定されていません")
	}

	targetEntity := *comp.Target

	// ゼロ値・死亡エンティティはArkのHasでパニックするため先に弾く
	if !world.ECS.Alive(targetEntity) {
		return fmt.Errorf("対象エンティティは扉ではありません")
	}

	// Doorコンポーネントを持っているか確認
	if !world.Components.Door.Has(targetEntity) {
		return fmt.Errorf("対象エンティティは扉ではありません")
	}

	return nil
}

// Start は扉閉鎖開始時の処理を実行する
func (cdb *CloseDoorBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("扉閉鎖開始", "actor", actor)
	return nil
}

// DoTurn は扉閉鎖アクティビティの1ターン分の処理を実行する
func (cdb *CloseDoorBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	targetEntity := *comp.Target

	if !world.Components.Door.Has(targetEntity) {
		Cancel(comp, "扉コンポーネントが取得できません")
		return fmt.Errorf("扉コンポーネントが取得できません")
	}
	raw := world.Components.Door.Get(targetEntity)
	doorComp := raw

	if doorComp.Locked {
		Cancel(comp, "扉はロックされている")
		return nil
	}

	// 扉を閉じる
	if doorComp.IsOpen {
		if err := lifecycle.CloseDoor(world, targetEntity); err != nil {
			Cancel(comp, fmt.Sprintf("扉を閉じられません: %v", err))
			return err
		}

		log.Debug("扉を閉じました", "door", targetEntity)

		// 視界の更新が必要であることをマーク（BlockViewが変更されたため）
		query.GetVisionState(world).RequestUpdate()
	}

	Complete(comp)
	return nil
}

// Finish は扉閉鎖完了時の処理を実行する
func (cdb *CloseDoorBehavior) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("扉閉鎖アクティビティ完了", "actor", actor)

	// プレイヤーの場合のみメッセージを表示
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Append("扉を閉じた。").
			Log()
	}

	return nil
}

// Canceled は扉閉鎖キャンセル時の処理を実行する
func (cdb *CloseDoorBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("扉閉鎖キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}
