package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// TalkBehavior は会話アクティビティ
type TalkBehavior struct{}

// Info はBehaviorの実装
func (tb *TalkBehavior) Info() Info {
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
func (tb *TalkBehavior) Name() gc.BehaviorName {
	return gc.BehaviorTalk
}

// NewTalkActivity は会話対象を指定して会話アクティビティを組む。
func NewTalkActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorTalk, 0)
	comp.Params = &gc.TargetParams{Target: target}
	return comp
}

// Validate は会話アクティビティの検証を行う
func (tb *TalkBehavior) Validate(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.TargetParams)
	if !ok {
		return fmt.Errorf("会話対象が指定されていません")
	}

	targetEntity := p.Target

	// Dialogコンポーネントを持っているか確認
	if !world.Components.Dialog.Has(targetEntity) {
		return fmt.Errorf("対象エンティティは会話できません")
	}

	// 中立派閥か確認
	if !query.IsNeutral(world, targetEntity) {
		return fmt.Errorf("対象エンティティは中立派閥ではありません")
	}

	return nil
}

// Start は会話開始時の処理を実行する
func (tb *TalkBehavior) Start(_ *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("会話開始", "actor", actor)
	return nil
}

// DoTurn は会話アクティビティの1ターン分の処理を実行する
func (tb *TalkBehavior) DoTurn(comp *gc.Activity, _ ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.TargetParams)
	if !ok {
		Cancel(comp, "会話対象が指定されていません")
		return fmt.Errorf("会話対象が指定されていません")
	}
	targetEntity := p.Target

	if !world.Components.Dialog.Has(targetEntity) {
		Cancel(comp, "会話データが取得できません")
		return fmt.Errorf("会話データが取得できません")
	}
	dialogComp := world.Components.Dialog.Get(targetEntity)

	// Nameコンポーネントから話者名を取得
	if !world.Components.Name.Has(targetEntity) {
		Cancel(comp, "対象エンティティにNameコンポーネントがありません")
		return fmt.Errorf("対象エンティティにNameコンポーネントがありません")
	}
	nameComp := world.Components.Name.Get(targetEntity)
	speakerName := nameComp.Name

	log.Debug("会話実行", "messageKey", dialogComp.MessageKey, "speaker", speakerName)

	// 会話メッセージの表示はstateで行うため、ここでは完了のみ
	Complete(comp)
	return nil
}

// Finish は会話完了時の処理を実行する
func (tb *TalkBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("会話アクティビティ完了", "actor", actor)

	p, ok := comp.Params.(*gc.TargetParams)
	if !ok {
		return nil
	}

	targetEntity := p.Target

	// プレイヤーの場合のみメッセージを表示
	if world.Components.Player.Has(actor) {
		if !world.Components.Name.Has(targetEntity) {
			return fmt.Errorf("対象エンティティにNameコンポーネントがありません")
		}
		nameComp := world.Components.Name.Get(targetEntity)

		gamelog.New(query.GetGameLog(world)).
			Append(nameComp.Name + "と話した。").
			Log()

		// 会話ダイアログを表示
		if world.Components.Dialog.Has(targetEntity) {
			dialog := world.Components.Dialog.Get(targetEntity)
			if err := lifecycle.RequestStateChange(world, gc.ShowDialogEvent(dialog.MessageKey, targetEntity)); err != nil {
				return fmt.Errorf("会話状態変更要求エラー: %w", err)
			}
		}
	}

	return nil
}

// Canceled は会話キャンセル時の処理を実行する
func (tb *TalkBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("会話キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}
