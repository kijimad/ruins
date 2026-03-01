package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// OpenDoorActivity はBehaviorの実装
type OpenDoorActivity struct{}

// Info はBehaviorの実装
func (oda *OpenDoorActivity) Info() Info {
	return Info{
		Name:            "ドア開閉",
		Description:     "ドアを開く",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (oda *OpenDoorActivity) Name() gc.BehaviorName {
	return gc.BehaviorOpenDoor
}

// Validate はドア開閉アクティビティの検証を行う
func (oda *OpenDoorActivity) Validate(comp *gc.CurrentActivity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("ドアエンティティが指定されていません")
	}

	targetEntity := *comp.Target

	// Doorコンポーネントを持っているか確認
	if !targetEntity.HasComponent(world.Components.Door) {
		return fmt.Errorf("対象エンティティはドアではありません")
	}

	return nil
}

// Start はドア開閉開始時の処理を実行する
func (oda *OpenDoorActivity) Start(_ *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("ドア開閉開始", "actor", actor)
	return nil
}

// DoTurn はドア開閉アクティビティの1ターン分の処理を実行する
func (oda *OpenDoorActivity) DoTurn(comp *gc.CurrentActivity, _ ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)
	targetEntity := *comp.Target

	doorComp := world.Components.Door.Get(targetEntity).(*gc.Door)
	if doorComp == nil {
		Cancel(comp, "ドアコンポーネントが取得できません")
		return fmt.Errorf("ドアコンポーネントが取得できません")
	}

	// ドアを開く
	if !doorComp.IsOpen {
		if err := worldhelper.OpenDoor(world, targetEntity); err != nil {
			Cancel(comp, fmt.Sprintf("ドアを開けません: %v", err))
			return err
		}

		log.Debug("ドアを開きました", "door", targetEntity)

		// 視界の更新が必要
		world.Resources.Dungeon.NeedsForceUpdate = true
	}

	Complete(comp)
	return nil
}

// Finish はドア開閉完了時の処理を実行する
func (oda *OpenDoorActivity) Finish(_ *gc.CurrentActivity, actor ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("ドア開閉アクティビティ完了", "actor", actor)

	// プレイヤーの場合のみメッセージを表示
	if actor.HasComponent(world.Components.Player) {
		gamelog.New(gamelog.FieldLog).
			Append("ドアを開いた。").
			Log()
	}

	return nil
}

// Canceled はドア開閉キャンセル時の処理を実行する
func (oda *OpenDoorActivity) Canceled(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("ドア開閉キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// CloseDoorActivity はBehaviorの実装
type CloseDoorActivity struct{}

// Info はBehaviorの実装
func (cda *CloseDoorActivity) Info() Info {
	return Info{
		Name:            "ドア閉鎖",
		Description:     "ドアを閉じる",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (cda *CloseDoorActivity) Name() gc.BehaviorName {
	return gc.BehaviorCloseDoor
}

// Validate はドア閉鎖アクティビティの検証を行う
func (cda *CloseDoorActivity) Validate(comp *gc.CurrentActivity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("ドアエンティティが指定されていません")
	}

	targetEntity := *comp.Target

	// Doorコンポーネントを持っているか確認
	if !targetEntity.HasComponent(world.Components.Door) {
		return fmt.Errorf("対象エンティティはドアではありません")
	}

	return nil
}

// Start はドア閉鎖開始時の処理を実行する
func (cda *CloseDoorActivity) Start(_ *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("ドア閉鎖開始", "actor", actor)
	return nil
}

// DoTurn はドア閉鎖アクティビティの1ターン分の処理を実行する
func (cda *CloseDoorActivity) DoTurn(comp *gc.CurrentActivity, _ ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)
	targetEntity := *comp.Target

	doorComp := world.Components.Door.Get(targetEntity).(*gc.Door)
	if doorComp == nil {
		Cancel(comp, "ドアコンポーネントが取得できません")
		return fmt.Errorf("ドアコンポーネントが取得できません")
	}

	// ドアを閉じる
	if doorComp.IsOpen {
		if err := worldhelper.CloseDoor(world, targetEntity); err != nil {
			Cancel(comp, fmt.Sprintf("ドアを閉じられません: %v", err))
			return err
		}

		log.Debug("ドアを閉じました", "door", targetEntity)

		// 視界の更新が必要であることをマーク（BlockViewが変更されたため）
		world.Resources.Dungeon.NeedsForceUpdate = true
	}

	Complete(comp)
	return nil
}

// Finish はドア閉鎖完了時の処理を実行する
func (cda *CloseDoorActivity) Finish(_ *gc.CurrentActivity, actor ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("ドア閉鎖アクティビティ完了", "actor", actor)

	// プレイヤーの場合のみメッセージを表示
	if actor.HasComponent(world.Components.Player) {
		gamelog.New(gamelog.FieldLog).
			Append("ドアを閉じた。").
			Log()
	}

	return nil
}

// Canceled はドア閉鎖キャンセル時の処理を実行する
func (cda *CloseDoorActivity) Canceled(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("ドア閉鎖キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}
