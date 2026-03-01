package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/logger"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TalkActivity は会話アクティビティ
type TalkActivity struct{}

// Info はBehaviorの実装
func (ta *TalkActivity) Info() Info {
	return Info{
		Name:            "会話",
		Description:     "NPCと会話する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (ta *TalkActivity) Name() gc.BehaviorName {
	return gc.BehaviorTalk
}

// Validate は会話アクティビティの検証を行う
func (ta *TalkActivity) Validate(comp *gc.CurrentActivity, _ ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return fmt.Errorf("会話対象が指定されていません")
	}

	targetEntity := *comp.Target

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
func (ta *TalkActivity) Start(_ *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("会話開始", "actor", actor)
	return nil
}

// DoTurn は会話アクティビティの1ターン分の処理を実行する
func (ta *TalkActivity) DoTurn(comp *gc.CurrentActivity, _ ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)
	targetEntity := *comp.Target

	dialogComp := world.Components.Dialog.Get(targetEntity).(*gc.Dialog)
	if dialogComp == nil {
		Cancel(comp, "会話データが取得できません")
		return fmt.Errorf("会話データが取得できません")
	}

	// Nameコンポーネントから話者名を取得
	if !targetEntity.HasComponent(world.Components.Name) {
		Cancel(comp, "対象エンティティにNameコンポーネントがありません")
		return fmt.Errorf("対象エンティティにNameコンポーネントがありません")
	}
	nameComp := world.Components.Name.Get(targetEntity).(*gc.Name)
	speakerName := nameComp.Name

	log.Debug("会話実行", "messageKey", dialogComp.MessageKey, "speaker", speakerName)

	// 会話メッセージの表示はstateで行うため、ここでは完了のみ
	Complete(comp)
	return nil
}

// Finish は会話完了時の処理を実行する
func (ta *TalkActivity) Finish(comp *gc.CurrentActivity, actor ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("会話アクティビティ完了", "actor", actor)

	if comp.Target == nil {
		return nil
	}

	targetEntity := *comp.Target

	// プレイヤーの場合のみメッセージを表示
	if actor.HasComponent(world.Components.Player) {
		if !targetEntity.HasComponent(world.Components.Name) {
			return fmt.Errorf("対象エンティティにNameコンポーネントがありません")
		}
		nameComp := world.Components.Name.Get(targetEntity).(*gc.Name)

		gamelog.New(gamelog.FieldLog).
			Append(nameComp.Name + "と話した。").
			Log()

		// 会話ダイアログを表示
		if targetEntity.HasComponent(world.Components.Dialog) {
			dialog := world.Components.Dialog.Get(targetEntity).(*gc.Dialog)
			if err := world.Resources.Dungeon.RequestStateChange(resources.ShowDialogEvent{
				MessageKey:    dialog.MessageKey,
				SpeakerEntity: targetEntity,
			}); err != nil {
				return fmt.Errorf("会話状態変更要求エラー: %w", err)
			}
		}
	}

	return nil
}

// Canceled は会話キャンセル時の処理を実行する
func (ta *TalkActivity) Canceled(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("会話キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}
