package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ExecuteInteraction は相互作用の種類に応じたアクティビティを実行する
func ExecuteInteraction(actor ecs.Entity, interactable ecs.Entity, world w.World) (*ActionResult, error) {
	if !interactable.HasComponent(world.Components.Interactable) {
		return nil, fmt.Errorf("指定されたエンティティはInteractableを持っていません")
	}

	comp := world.Components.Interactable.Get(interactable).(*gc.Interactable)
	config := comp.Data.Config()

	if err := config.ActivationRange.Valid(); err != nil {
		return nil, fmt.Errorf("無効なActivationRange: %w", err)
	}
	if err := config.ActivationWay.Valid(); err != nil {
		return nil, fmt.Errorf("無効なActivationWay: %w", err)
	}

	switch content := comp.Data.(type) {
	case gc.PortalInteraction:
		return executePortal(world, content)
	case gc.DungeonGateInteraction:
		return executeDungeonGate(world)
	case gc.DoorInteraction:
		return executeDoor(actor, interactable, world)
	case gc.DoorLockInteraction:
		return executeDoorLock(world)
	case gc.TalkInteraction:
		return executeTalk(actor, interactable, world)
	case gc.ItemInteraction:
		return executeItem(actor, world)
	case gc.MeleeInteraction:
		return executeMelee(actor, interactable, world)
	default:
		return nil, fmt.Errorf("未知の相互作用タイプ: %T", comp.Data)
	}
}

func executePortal(world w.World, portal gc.PortalInteraction) (*ActionResult, error) {
	switch portal.PortalType {
	case gc.PortalTypeNext:
		if err := world.Resources.Dungeon.RequestStateChange(resources.WarpNextEvent{}); err != nil {
			return nil, fmt.Errorf("次フロアワープ状態変更要求エラー: %w", err)
		}
	case gc.PortalTypeTown:
		if err := world.Resources.Dungeon.RequestStateChange(resources.WarpEscapeEvent{}); err != nil {
			return nil, fmt.Errorf("街帰還状態変更要求エラー: %w", err)
		}
	default:
		return nil, fmt.Errorf("未知のポータルタイプ: %s", portal.PortalType)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorPortal, Message: "ポータル移動"}, nil
}

func executeDungeonGate(world w.World) (*ActionResult, error) {
	if err := world.Resources.Dungeon.RequestStateChange(resources.OpenDungeonSelectEvent{}); err != nil {
		return nil, fmt.Errorf("ダンジョン選択状態変更要求エラー: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorDungeonGate, Message: "ダンジョンゲート発動"}, nil
}

func executeDoor(actor ecs.Entity, doorEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !doorEntity.HasComponent(world.Components.Door) {
		return nil, fmt.Errorf("DoorInteractionだがDoorコンポーネントがない")
	}

	door := world.Components.Door.Get(doorEntity).(*gc.Door)
	params := ActionParams{
		Actor:  actor,
		Target: &doorEntity,
	}

	if door.IsOpen {
		return Execute(&CloseDoorActivity{}, params, world)
	}
	return Execute(&OpenDoorActivity{}, params, world)
}

func executeDoorLock(world w.World) (*ActionResult, error) {
	if worldhelper.LockAllDoors(world) > 0 {
		gamelog.New(gamelog.FieldLog).
			Append("どこかで扉が閉じたようだ。").
			Log()
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorDoorLock, Message: "扉ロック"}, nil
}

func executeTalk(actor ecs.Entity, npcEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !npcEntity.HasComponent(world.Components.Dialog) {
		return nil, fmt.Errorf("TalkInteractionですがDialogコンポーネントがありません")
	}

	params := ActionParams{
		Actor:  actor,
		Target: &npcEntity,
	}

	result, err := Execute(&TalkActivity{}, params, world)
	if err != nil {
		return nil, fmt.Errorf("会話アクション失敗: %w", err)
	}

	return result, nil
}

func executeItem(actor ecs.Entity, world w.World) (*ActionResult, error) {
	params := ActionParams{
		Actor: actor,
	}
	result, err := Execute(&PickupActivity{}, params, world)
	return result, err
}

func executeMelee(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	params := ActionParams{
		Actor:  actor,
		Target: &target,
	}
	return Execute(&AttackActivity{}, params, world)
}
