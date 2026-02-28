package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ExecuteInteraction は相互作用の種類に応じたアクティビティを実行する
func ExecuteInteraction(manager *ActivityManager, actor ecs.Entity, interactable ecs.Entity, world w.World) (*ActionResult, error) {
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
		return executeDoor(manager, actor, interactable, world)
	case gc.TalkInteraction:
		return executeTalk(manager, actor, interactable, world)
	case gc.ItemInteraction:
		return executeItem(manager, actor, world)
	case gc.MeleeInteraction:
		return executeMelee(manager, actor, interactable, world)
		// TODO: 消す
	case gc.TestTriggerInteraction:
		return executeTestTrigger(content)
	default:
		// TODO: エラーにする
		// 未知の型はテスト用やカスタム拡張の可能性があるため、警告のみで成功を返す
		logger.New(logger.CategoryAction).Warn("未知の相互作用タイプ", "type", fmt.Sprintf("%T", comp.Data))
		return &ActionResult{Success: true, ActivityName: "Unknown", Message: "相互作用実行"}, nil
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
	return &ActionResult{Success: true, ActivityName: "Portal", Message: "ポータル移動"}, nil
}

func executeDungeonGate(world w.World) (*ActionResult, error) {
	if err := world.Resources.Dungeon.RequestStateChange(resources.OpenDungeonSelectEvent{}); err != nil {
		return nil, fmt.Errorf("ダンジョン選択状態変更要求エラー: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: "DungeonGate", Message: "ダンジョンゲート発動"}, nil
}

func executeDoor(manager *ActivityManager, actor ecs.Entity, doorEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !doorEntity.HasComponent(world.Components.Door) {
		return nil, fmt.Errorf("DoorInteractionだがDoorコンポーネントがない")
	}

	door := world.Components.Door.Get(doorEntity).(*gc.Door)
	params := ActionParams{
		Actor:  actor,
		Target: &doorEntity,
	}

	if door.IsOpen {
		return manager.Execute(&CloseDoorActivity{}, params, world)
	}
	return manager.Execute(&OpenDoorActivity{}, params, world)
}

func executeTalk(manager *ActivityManager, actor ecs.Entity, npcEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !npcEntity.HasComponent(world.Components.Dialog) {
		return nil, fmt.Errorf("TalkInteractionですがDialogコンポーネントがありません")
	}

	params := ActionParams{
		Actor:  actor,
		Target: &npcEntity,
	}

	result, err := manager.Execute(&TalkActivity{}, params, world)
	if err != nil {
		return nil, fmt.Errorf("会話アクション失敗: %w", err)
	}

	if result != nil && result.Success {
		dialog := world.Components.Dialog.Get(npcEntity).(*gc.Dialog)
		if err := world.Resources.Dungeon.RequestStateChange(resources.ShowDialogEvent{
			MessageKey:    dialog.MessageKey,
			SpeakerEntity: npcEntity,
		}); err != nil {
			return nil, fmt.Errorf("会話状態変更要求エラー: %w", err)
		}
	}
	return result, nil
}

func executeItem(manager *ActivityManager, actor ecs.Entity, world w.World) (*ActionResult, error) {
	params := ActionParams{
		Actor: actor,
	}
	result, err := manager.Execute(&PickupActivity{}, params, world)
	if err != nil {
		logger.New(logger.CategoryAction).Warn("アイテム拾得アクション失敗", "error", err)
	}
	return result, err
}

func executeMelee(manager *ActivityManager, actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	params := ActionParams{
		Actor:  actor,
		Target: &target,
	}
	return manager.Execute(&AttackActivity{}, params, world)
}

func executeTestTrigger(content gc.TestTriggerInteraction) (*ActionResult, error) {
	if content.Executed != nil {
		*content.Executed = true
	}
	return &ActionResult{Success: true, ActivityName: "TestTrigger", Message: "テストトリガー実行"}, nil
}
