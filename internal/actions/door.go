package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
)

// OpenDoorActivity はActivityInterfaceの実装
type OpenDoorActivity struct{}

// Info はActivityInterfaceの実装
func (oda *OpenDoorActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "ドア開閉",
		Description:     "ドアを開く",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 100,
		TotalRequiredAP: 100,
	}
}

// String はActivityInterfaceの実装
func (oda *OpenDoorActivity) String() string {
	return "OpenDoor"
}

// Validate はドア開閉アクティビティの検証を行う
func (oda *OpenDoorActivity) Validate(act *Activity, world w.World) error {
	if act.Target == nil {
		return fmt.Errorf("ドアエンティティが指定されていません")
	}

	targetEntity := *act.Target

	// Doorコンポーネントを持っているか確認
	if !targetEntity.HasComponent(world.Components.Door) {
		return fmt.Errorf("対象エンティティはドアではありません")
	}

	return nil
}

// Start はドア開閉開始時の処理を実行する
func (oda *OpenDoorActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("ドア開閉開始", "actor", act.Actor)
	return nil
}

// DoTurn はドア開閉アクティビティの1ターン分の処理を実行する
func (oda *OpenDoorActivity) DoTurn(act *Activity, world w.World) error {
	targetEntity := *act.Target

	doorComp := world.Components.Door.Get(targetEntity).(*gc.Door)
	if doorComp == nil {
		act.Cancel("ドアコンポーネントが取得できません")
		return fmt.Errorf("ドアコンポーネントが取得できません")
	}

	// ドアを開く
	if !doorComp.IsOpen {
		doorComp.IsOpen = true

		// BlockPass と BlockView を削除（通行可能・視線が通るようになる）
		if targetEntity.HasComponent(world.Components.BlockPass) {
			targetEntity.RemoveComponent(world.Components.BlockPass)
		}
		if targetEntity.HasComponent(world.Components.BlockView) {
			targetEntity.RemoveComponent(world.Components.BlockView)
		}

		// スプライトを開いた状態に変更
		if targetEntity.HasComponent(world.Components.SpriteRender) {
			spriteRender := world.Components.SpriteRender.Get(targetEntity).(*gc.SpriteRender)
			if doorComp.Orientation == gc.DoorOrientationHorizontal {
				spriteRender.SpriteKey = "door_horizontal_open"
			} else {
				spriteRender.SpriteKey = "door_vertical_open"
			}
		}

		act.Logger.Debug("ドアを開きました", "door", targetEntity)

		// 視界の更新が必要
		world.Resources.Dungeon.NeedsForceUpdate = true
	}

	act.Complete()
	return nil
}

// Finish はドア開閉完了時の処理を実行する
func (oda *OpenDoorActivity) Finish(act *Activity, world w.World) error {
	act.Logger.Debug("ドア開閉アクティビティ完了", "actor", act.Actor)

	// プレイヤーの場合のみメッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("ドアを開いた。").
			Log()
	}

	return nil
}

// Canceled はドア開閉キャンセル時の処理を実行する
func (oda *OpenDoorActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("ドア開閉キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}

// CloseDoorActivity はActivityInterfaceの実装
type CloseDoorActivity struct{}

// Info はActivityInterfaceの実装
func (cda *CloseDoorActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "ドア閉鎖",
		Description:     "ドアを閉じる",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 100,
		TotalRequiredAP: 100,
	}
}

// String はActivityInterfaceの実装
func (cda *CloseDoorActivity) String() string {
	return "CloseDoor"
}

// Validate はドア閉鎖アクティビティの検証を行う
func (cda *CloseDoorActivity) Validate(act *Activity, world w.World) error {
	if act.Target == nil {
		return fmt.Errorf("ドアエンティティが指定されていません")
	}

	targetEntity := *act.Target

	// Doorコンポーネントを持っているか確認
	if !targetEntity.HasComponent(world.Components.Door) {
		return fmt.Errorf("対象エンティティはドアではありません")
	}

	return nil
}

// Start はドア閉鎖開始時の処理を実行する
func (cda *CloseDoorActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("ドア閉鎖開始", "actor", act.Actor)
	return nil
}

// DoTurn はドア閉鎖アクティビティの1ターン分の処理を実行する
func (cda *CloseDoorActivity) DoTurn(act *Activity, world w.World) error {
	targetEntity := *act.Target

	doorComp := world.Components.Door.Get(targetEntity).(*gc.Door)
	if doorComp == nil {
		act.Cancel("ドアコンポーネントが取得できません")
		return fmt.Errorf("ドアコンポーネントが取得できません")
	}

	// ドアを閉じる
	if doorComp.IsOpen {
		doorComp.IsOpen = false

		// BlockPass と BlockView を追加（通行不可・視線が通らなくなる）
		if !targetEntity.HasComponent(world.Components.BlockPass) {
			targetEntity.AddComponent(world.Components.BlockPass, &gc.BlockPass{})
		}
		if !targetEntity.HasComponent(world.Components.BlockView) {
			targetEntity.AddComponent(world.Components.BlockView, &gc.BlockView{})
		}

		// スプライトを閉じた状態に変更
		if targetEntity.HasComponent(world.Components.SpriteRender) {
			spriteRender := world.Components.SpriteRender.Get(targetEntity).(*gc.SpriteRender)
			if doorComp.Orientation == gc.DoorOrientationHorizontal {
				spriteRender.SpriteKey = "door_horizontal_closed"
			} else {
				spriteRender.SpriteKey = "door_vertical_closed"
			}
		}

		act.Logger.Debug("ドアを閉じました", "door", targetEntity)

		// 視界の更新が必要であることをマーク（BlockViewが変更されたため）
		world.Resources.Dungeon.NeedsForceUpdate = true
	}

	act.Complete()
	return nil
}

// Finish はドア閉鎖完了時の処理を実行する
func (cda *CloseDoorActivity) Finish(act *Activity, world w.World) error {
	act.Logger.Debug("ドア閉鎖アクティビティ完了", "actor", act.Actor)

	// プレイヤーの場合のみメッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("ドアを閉じた。").
			Log()
	}

	return nil
}

// Canceled はドア閉鎖キャンセル時の処理を実行する
func (cda *CloseDoorActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("ドア閉鎖キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}
