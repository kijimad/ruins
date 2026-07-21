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
func ExecuteInteraction(actor ecs.Entity, target ecs.Entity, interaction gc.InteractionKind, world w.World) (*ActionResult, error) {
	config := interaction.Config()

	if err := config.ActivationRange.Valid(); err != nil {
		return nil, fmt.Errorf("無効なActivationRange: %w", err)
	}
	if err := config.ActivationWay.Valid(); err != nil {
		return nil, fmt.Errorf("無効なActivationWay: %w", err)
	}

	switch interaction {
	case gc.InteractionPortalNext:
		// 共存方式の下り。同一 State 内 swapTo で現階を退避し、再訪で復元できる
		return executePortal(world, gc.WarpDescendEvent(), "次フロアワープ状態変更要求エラー")
	case gc.InteractionPortalPrev:
		return executePortal(world, gc.WarpAscendEvent(), "前フロアワープ状態変更要求エラー")
	case gc.InteractionDungeonEnter:
		return executeDungeonEnter(target, world)
	case gc.InteractionDoor:
		return executeDoor(actor, target, world)
	case gc.InteractionDoorLock:
		return executeDoorLock(world)
	case gc.InteractionTalk:
		return executeTalk(actor, target, world)
	case gc.InteractionItem:
		return executeItem(actor, target, world)
	case gc.InteractionItemAll:
		return executeItemAll(actor, world)
	case gc.InteractionStorage:
		return executeStorage(target, world)
	case gc.InteractionMelee:
		return executeMelee(actor, target, world)
	}
	// default を置かず exhaustive に全種別を強制する。未知入力は raw/save 由来でありうるので
	// panic せず error で loud に落とす
	return nil, fmt.Errorf("未知の相互作用タイプ: %s", interaction)
}

func executePortal(world w.World, event gc.StateChangeRequest, errMsg string) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, event); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorPortal, Message: "ポータル移動"}, nil
}

// executeDungeonEnter は遺跡入口の進入先を入口プロップの DungeonEntrance から読み、遺跡進入を要求する。
// 入口ごとに進入先が違うため、イベントに定義名を載せて運ぶ。
func executeDungeonEnter(target ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.DungeonEntrance.Has(target) {
		return nil, fmt.Errorf("遺跡入口に進入先の遺跡定義がありません")
	}
	defName := world.Components.DungeonEntrance.Get(target).DefinitionName
	if err := lifecycle.RequestStateChange(world, gc.WarpDungeonEnterEvent(defName)); err != nil {
		return nil, fmt.Errorf("遺跡進入状態変更要求エラー: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorPortal, Message: "遺跡進入"}, nil
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
	if !world.Components.GridElement.Has(actor) {
		return nil, fmt.Errorf("位置情報が見つかりません")
	}
	gridElement := world.Components.GridElement.Get(actor)
	destination := gc.GridElement{Coord: gridElement.Coord}
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
