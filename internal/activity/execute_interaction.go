package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/raw"
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
		return executePortal(world, gc.WarpDescendEvent(), "次フロアワープ状態変更要求エラー", "ポータル移動")
	case gc.InteractionPortalPrev:
		return executePortal(world, gc.WarpAscendEvent(), "前フロアワープ状態変更要求エラー", "ポータル移動")
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
	case gc.InteractionDisassemble:
		return executeDisassemble(actor, target, world)
	case gc.InteractionEnterCube:
		// 入る対象のキューブ本体を載せて運ぶ。退場時の戻り先解決に使う
		return executePortal(world, gc.WarpCubeEnterEvent(target), "キューブ入場状態変更要求エラー", "キューブに入る")
	case gc.InteractionExitCube:
		return executePortal(world, gc.WarpCubeExitEvent(), "キューブ退場状態変更要求エラー", "キューブから出る")
	case gc.InteractionPullCube:
		return executePullCube(actor, target, world)
	case gc.InteractionCubePanel:
		return executePortal(world, gc.OpenCubePanelEvent(), "コントロールパネル状態変更要求エラー", "コントロールパネルを開いた")
	}
	// default を置かず exhaustive に全種別を強制する。未知入力は raw/save 由来でありうるので
	// panic せず error で loud に落とす
	return nil, fmt.Errorf("未知の相互作用タイプ: %s", interaction)
}

// executePortal は状態機械へ遷移リクエストを投げる。実際の SwapTo や画面遷移は状態機械が担う。
// ポータル・キューブの入退場・コントロールパネルなど、リクエストを1つ出して終わる相互作用が共通で使う。
// failMsg はリクエスト失敗時にエラーへ前置する文脈、successMsg は成功時に ActionResult へ載せる表示。
func executePortal(world w.World, event gc.StateChangeRequest, failMsg, successMsg string) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, event); err != nil {
		return nil, fmt.Errorf("%s: %w", failMsg, err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorPortal, Message: successMsg}, nil
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

// executePullCube はキューブを自分の側へ引く。後退スペースが無いなど引けないときは、
// 致命エラーにせずログで知らせて no-op にする。プレイヤーのできない操作は異常系でないため、
// Execute を呼ぶ前に可否を判定してから分岐する。
func executePullCube(actor ecs.Entity, cube ecs.Entity, world w.World) (*ActionResult, error) {
	if !canPullCube(world, actor, cube) {
		gamelog.New(query.GetGameLog(world)).Append("引くスペースがない。").Log()
		return &ActionResult{Success: false, ActivityName: gc.BehaviorPull, Message: "引けない"}, nil
	}
	return Execute(NewPullBehavior(cube), actor, world)
}

func executeDoor(actor ecs.Entity, doorEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.Door.Has(doorEntity) {
		return nil, fmt.Errorf("DoorInteractionだがDoorコンポーネントがない")
	}

	door := world.Components.Door.Get(doorEntity)

	if door.IsOpen {
		return Execute(&CloseDoorBehavior{Target: doorEntity}, actor, world)
	}
	return Execute(&OpenDoorBehavior{Target: doorEntity}, actor, world)
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

	result, err := Execute(&TalkBehavior{Target: npcEntity}, actor, world)
	if err != nil {
		return nil, fmt.Errorf("会話アクション失敗: %w", err)
	}

	return result, nil
}

func executeItem(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(&PickupBehavior{Target: &target}, actor, world)
}

func executeItemAll(actor ecs.Entity, world w.World) (*ActionResult, error) {
	if !world.Components.GridElement.Has(actor) {
		return nil, fmt.Errorf("位置情報が見つかりません")
	}
	gridElement := world.Components.GridElement.Get(actor)
	return Execute(&PickupBehavior{Destination: &gc.GridElement{Coord: gridElement.Coord}}, actor, world)
}

func executeStorage(storageEntity ecs.Entity, world w.World) (*ActionResult, error) {
	if err := lifecycle.RequestStateChange(world, gc.OpenStorageEvent(storageEntity)); err != nil {
		return nil, fmt.Errorf("収納メニュー状態変更要求エラー: %w", err)
	}
	return &ActionResult{Success: true, ActivityName: gc.BehaviorStorage, Message: "収納を開いた"}, nil
}

func executeMelee(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	return Execute(&AttackBehavior{Target: target}, actor, world)
}

// executeDisassemble は工具の有無を先に確かめ、無ければエラーでなくログで知らせる。
// 工具不足はプレイヤーの通常操作で起きる状態であり、異常系ではないため
func executeDisassemble(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	name := query.GetEntityName(target, world)
	def, ok := raw.FindDisassembly(world.Resources.RawMaster, name)
	if !ok {
		return nil, fmt.Errorf("対象は分解定義を持っていません")
	}
	if _, _, ok := FindBestDisassemblyTool(world, actor, def.ToolCategory); !ok {
		gamelog.New(query.GetGameLog(world)).
			ItemName(name).
			Append("を分解できる工具を持っていない").
			Log()
		return &ActionResult{Success: false, ActivityName: gc.BehaviorDisassemble, Message: "工具がない"}, nil
	}
	return Execute(&DisassembleBehavior{Target: target}, actor, world)
}
