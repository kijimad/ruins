package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InteractionActivateActivity は相互作用を発動するActivity
type InteractionActivateActivity struct {
	InteractableEntity ecs.Entity
}

// Info はActivityInterfaceの実装
func (ia *InteractionActivateActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "相互作用発動",
		Description:     "相互作用を発動する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 0,
		TotalRequiredAP: 0,
	}
}

// String はActivityInterfaceの実装
func (ia *InteractionActivateActivity) String() string {
	return "InteractionActivate"
}

// Validate は相互作用発動アクティビティの検証を行う
func (ia *InteractionActivateActivity) Validate(_ *Activity, world w.World) error {
	// InteractableEntityが存在するかチェック
	if !ia.InteractableEntity.HasComponent(world.Components.Interactable) {
		return fmt.Errorf("指定されたエンティティはInteractableを持っていません")
	}

	// Interactableの設定が有効かチェック
	interactable := world.Components.Interactable.Get(ia.InteractableEntity).(*gc.Interactable)
	config := interactable.Data.Config()

	// ActivationRangeの検証
	if err := config.ActivationRange.Valid(); err != nil {
		return fmt.Errorf("無効なActivationRange: %w", err)
	}

	// ActivationWayの検証
	if err := config.ActivationWay.Valid(); err != nil {
		return fmt.Errorf("無効なActivationWay: %w", err)
	}

	return nil
}

// Start は相互作用発動開始時の処理を実行する
func (ia *InteractionActivateActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("相互作用発動開始", "actor", act.Actor, "interactable", ia.InteractableEntity)
	return nil
}

// DoTurn は相互作用発動アクティビティの1ターン分の処理を実行する
func (ia *InteractionActivateActivity) DoTurn(act *Activity, world w.World) error {
	interactable := world.Components.Interactable.Get(ia.InteractableEntity).(*gc.Interactable)

	// 相互作用の種類に応じた処理を実行
	var interactionErr error
	switch content := interactable.Data.(type) {
	case gc.BridgeInteraction:
		ia.executeBridge(act, world, content)
	case gc.DoorInteraction:
		ia.executeDoor(act, world, content)
	case gc.TalkInteraction:
		ia.executeTalk(act, world, content)
	case gc.ItemInteraction:
		ia.executeItem(act, world, content)
	case gc.MeleeInteraction:
		ia.executeMelee(act, world, content)
	case gc.TestTriggerInteraction:
		ia.executeTestTrigger(act, world, content)
	default:
		interactionErr = fmt.Errorf("未知の相互作用タイプ: %T", interactable)
		act.Cancel(fmt.Sprintf("相互作用発動エラー: %s", interactionErr.Error()))
	}

	if interactionErr != nil {
		return interactionErr
	}

	act.Complete()
	return nil
}

// Finish は相互作用発動完了時の処理を実行する
func (ia *InteractionActivateActivity) Finish(act *Activity, _ w.World) error {
	act.Logger.Debug("相互作用発動アクティビティ完了", "actor", act.Actor)
	return nil
}

// Canceled は相互作用発動キャンセル時の処理を実行する
func (ia *InteractionActivateActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("相互作用発動キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}

// executeBridge は橋を渡る相互作用を実行する
func (ia *InteractionActivateActivity) executeBridge(act *Activity, world w.World, bridge gc.BridgeInteraction) {
	// NextFloorSeed を Dungeon リソースに保存
	world.Resources.Dungeon.NextFloorSeed = bridge.NextFloorSeed
	world.Resources.Dungeon.SelectedBridgeID = bridge.BridgeID

	// WarpNextEvent を発行
	world.Resources.Dungeon.SetStateEvent(resources.WarpNextEvent{})

	act.Logger.Debug("橋を渡る", "actor", act.Actor, "bridgeID", bridge.BridgeID, "nextSeed", bridge.NextFloorSeed)

	// ログ出力
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Magic(fmt.Sprintf("橋%sを渡った。", bridge.BridgeID)).
			Log()
	}
}

// executeDoor はドア相互作用を実行する
func (ia *InteractionActivateActivity) executeDoor(act *Activity, world w.World, _ gc.DoorInteraction) {
	if !ia.InteractableEntity.HasComponent(world.Components.Door) {
		act.Logger.Warn("DoorInteractionだがDoorコンポーネントがない", "entity", ia.InteractableEntity)
		return
	}

	door := world.Components.Door.Get(ia.InteractableEntity).(*gc.Door)

	// ドアの状態に応じて開閉アクティビティを実行
	var doorActivity ActivityInterface
	if door.IsOpen {
		doorActivity = &CloseDoorActivity{}
	} else {
		doorActivity = &OpenDoorActivity{}
	}

	params := ActionParams{
		Actor:  act.Actor,
		Target: &ia.InteractableEntity,
	}

	manager := NewActivityManager(act.Logger)
	_, err := manager.Execute(doorActivity, params, world)
	if err != nil {
		act.Logger.Warn("ドアアクション失敗", "error", err)
	}
}

// executeTalk は会話相互作用を実行する
func (ia *InteractionActivateActivity) executeTalk(act *Activity, world w.World, _ gc.TalkInteraction) {
	if !ia.InteractableEntity.HasComponent(world.Components.Dialog) {
		act.Logger.Warn("TalkInteractionだがDialogコンポーネントがない", "entity", ia.InteractableEntity)
		return
	}

	params := ActionParams{
		Actor:  act.Actor,
		Target: &ia.InteractableEntity,
	}

	manager := NewActivityManager(act.Logger)
	result, err := manager.Execute(&TalkActivity{}, params, world)
	if err != nil {
		act.Logger.Warn("会話アクション失敗", "error", err)
		return
	}

	// 会話成功時は会話メッセージを表示するStateEventを設定
	if result != nil && result.Success {
		dialog := world.Components.Dialog.Get(ia.InteractableEntity).(*gc.Dialog)
		world.Resources.Dungeon.SetStateEvent(resources.ShowDialogEvent{
			MessageKey:    dialog.MessageKey,
			SpeakerEntity: ia.InteractableEntity,
		})
	}
}

// executeItem はアイテム拾得相互作用を実行する
func (ia *InteractionActivateActivity) executeItem(act *Activity, world w.World, _ gc.ItemInteraction) {
	params := ActionParams{
		Actor: act.Actor,
	}

	manager := NewActivityManager(act.Logger)
	_, err := manager.Execute(&PickupActivity{}, params, world)
	if err != nil {
		act.Logger.Warn("アイテム拾得アクション失敗", "error", err)
	}
}

// executeMelee は近接攻撃相互作用を実行する
func (ia *InteractionActivateActivity) executeMelee(act *Activity, world w.World, _ gc.MeleeInteraction) {
	params := ActionParams{
		Actor:  act.Actor,
		Target: &ia.InteractableEntity, // 相互作用可能エンティティを攻撃対象とする
	}

	manager := NewActivityManager(act.Logger)
	_, err := manager.Execute(&AttackActivity{}, params, world)
	if err != nil {
		act.Logger.Warn("近接攻撃アクション失敗", "error", err)
	}
}

// executeTestTrigger はテスト用トリガー相互作用を実行する
func (ia *InteractionActivateActivity) executeTestTrigger(act *Activity, _ w.World, content gc.TestTriggerInteraction) {
	act.Logger.Debug("テストトリガー実行", "actor", act.Actor, "trigger", ia.InteractableEntity)
	// Executedフラグをtrueに設定
	if content.Executed != nil {
		*content.Executed = true
	}
}
