package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ExecuteInteraction は相互作用の種類に応じたアクティビティを実行する。
// interactionには実行すべき具体的なInteractionDataを渡す
func ExecuteInteraction(actor ecs.Entity, target ecs.Entity, interaction gc.InteractionData, world w.World) (*ActionResult, error) {
	config := interaction.Config()

	if err := config.ActivationRange.Valid(); err != nil {
		return nil, fmt.Errorf("無効なActivationRange: %w", err)
	}
	if err := config.ActivationWay.Valid(); err != nil {
		return nil, fmt.Errorf("無効なActivationWay: %w", err)
	}

	switch content := interaction.(type) {
	case gc.PortalInteraction:
		return executePortal(world, content)
	case gc.DungeonGateInteraction:
		return executeDungeonGate(world)
	case gc.DoorInteraction:
		return executeDoor(actor, target, world)
	case gc.DoorLockInteraction:
		return executeDoorLock(world)
	case gc.TalkInteraction:
		return executeTalk(actor, target, world)
	case gc.ItemInteraction:
		return executeItem(actor, target, world)
	case gc.ItemAllInteraction:
		return executeItemAll(actor, world)
	case gc.StorageInteraction:
		return executeStorage(target, world)
	case gc.MeleeInteraction:
		return executeMelee(actor, target, world)
	default:
		return nil, fmt.Errorf("未知の相互作用タイプ: %T", interaction)
	}
}

func executePortal(world w.World, portal gc.PortalInteraction) (*ActionResult, error) {
	switch portal.PortalType {
	case gc.PortalTypeNext:
		if err := lifecycle.RequestStateChange(world, gc.WarpNextEvent()); err != nil {
			return nil, fmt.Errorf("次フロアワープ状態変更要求エラー: %w", err)
		}
	case gc.PortalTypeTown:
		if err := lifecycle.RequestStateChange(world, gc.WarpEscapeEvent()); err != nil {
			return nil, fmt.Errorf("街帰還状態変更要求エラー: %w", err)
		}
	default:
		return nil, fmt.Errorf("未知のポータルタイプ: %s", portal.PortalType)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorPortal, Message: "ポータル移動"}, nil
}

func executeDungeonGate(world w.World) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, gc.OpenDungeonSelectEvent()); err != nil {
		return nil, fmt.Errorf("ダンジョン選択状態変更要求エラー: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorDungeonGate, Message: "ダンジョンゲート発動"}, nil
}

func executeDoor(actor ecs.Entity, doorEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.Door.Has(doorEntity) {
		return nil, fmt.Errorf("DoorInteractionだがDoorコンポーネントがない")
	}

	door := world.Components.Door.Get(doorEntity)

	if door.IsOpen {
		return Execute(&CloseDoorActivity{Target: doorEntity}, actor, world)
	}
	return Execute(&OpenDoorActivity{Target: doorEntity}, actor, world)
}

func executeDoorLock(world w.World) (*ActionResult, error) {
	if lifecycle.LockAllDoors(world) > 0 {
		gamelog.New(query.GetGameLog(world)).
			Append("どこかで扉が閉じたようだ。").
			Log()
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorDoorLock, Message: "扉ロック"}, nil
}

func executeTalk(actor ecs.Entity, npcEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.Dialog.Has(npcEntity) {
		return nil, fmt.Errorf("TalkInteractionですがDialogコンポーネントがありません")
	}

	result, err := Execute(&TalkActivity{Target: npcEntity}, actor, world)
	if err != nil {
		return nil, fmt.Errorf("会話アクション失敗: %w", err)
	}

	return result, nil
}

func executeItem(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(&PickupActivity{Target: &target}, actor, world)
}

func executeItemAll(actor ecs.Entity, world w.World) (*ActionResult, error) {
	gridElement := world.Components.GridElement.Get(actor)
	if gridElement == nil {
		return nil, fmt.Errorf("位置情報が見つかりません")
	}
	playerGrid := gridElement
	destination := gc.GridElement{X: playerGrid.X, Y: playerGrid.Y}
	return Execute(&PickupActivity{Destination: &destination}, actor, world)
}

func executeStorage(storageEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, gc.OpenStorageEvent(storageEntity)); err != nil {
		return nil, fmt.Errorf("収納メニュー状態変更要求エラー: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorStorage, Message: "収納を開いた"}, nil
}

func executeMelee(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(&AttackActivity{Target: target}, actor, world)
}
