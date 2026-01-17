package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
)

// TalkActivity は会話アクティビティ
type TalkActivity struct{}

// Info はActivityInterfaceの実装
func (ta *TalkActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "会話",
		Description:     "NPCと会話する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 100,
		TotalRequiredAP: 100,
	}
}

// String はActivityInterfaceの実装
func (ta *TalkActivity) String() string {
	return "Talk"
}

// Validate は会話アクティビティの検証を行う
func (ta *TalkActivity) Validate(act *Activity, world w.World) error {
	if act.Target == nil {
		return fmt.Errorf("会話対象が指定されていません")
	}

	targetEntity := *act.Target

	// Dialogコンポーネントを持っているか確認
	if !targetEntity.HasComponent(world.Components.Dialog) {
		return fmt.Errorf("対象エンティティは会話できません")
	}

	// FactionNeutralを持っているか確認
	if !targetEntity.HasComponent(world.Components.FactionNeutral) {
		return fmt.Errorf("対象エンティティは中立派閥ではありません")
	}

	return nil
}

// Start は会話開始時の処理を実行する
func (ta *TalkActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("会話開始", "actor", act.Actor)
	return nil
}

// DoTurn は会話アクティビティの1ターン分の処理を実行する
func (ta *TalkActivity) DoTurn(act *Activity, world w.World) error {
	targetEntity := *act.Target

	dialogComp := world.Components.Dialog.Get(targetEntity).(*gc.Dialog)
	if dialogComp == nil {
		act.Cancel("会話データが取得できません")
		return fmt.Errorf("会話データが取得できません")
	}

	// Nameコンポーネントから話者名を取得
	if !targetEntity.HasComponent(world.Components.Name) {
		act.Cancel("対象エンティティにNameコンポーネントがありません")
		return fmt.Errorf("対象エンティティにNameコンポーネントがありません")
	}
	nameComp := world.Components.Name.Get(targetEntity).(*gc.Name)
	speakerName := nameComp.Name

	act.Logger.Debug("会話実行", "messageKey", dialogComp.MessageKey, "speaker", speakerName)

	// 会話メッセージの表示はstateで行うため、ここでは完了のみ
	act.Complete()
	return nil
}

// Finish は会話完了時の処理を実行する
func (ta *TalkActivity) Finish(act *Activity, world w.World) error {
	act.Logger.Debug("会話アクティビティ完了", "actor", act.Actor)

	// プレイヤーの場合のみメッセージを表示
	if isPlayerActivity(act, world) {
		targetEntity := *act.Target
		if !targetEntity.HasComponent(world.Components.Name) {
			return fmt.Errorf("対象エンティティにNameコンポーネントがありません")
		}
		nameComp := world.Components.Name.Get(targetEntity).(*gc.Name)

		gamelog.New(gamelog.FieldLog).
			Append(nameComp.Name + "と話した。").
			Log()
	}

	return nil
}

// Canceled は会話キャンセル時の処理を実行する
func (ta *TalkActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("会話キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}
