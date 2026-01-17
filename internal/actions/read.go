package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
)

// ReadActivity はActivityInterfaceの実装
type ReadActivity struct{}

// Info はActivityInterfaceの実装
func (ra *ReadActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "読書",
		Description:     "本を読んでスキルを習得する",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: 100,
		TotalRequiredAP: 2000,
	}
}

// String はActivityInterfaceの実装
func (ra *ReadActivity) String() string {
	return "Read"
}

// Validate は読書アクティビティの検証を行う
// Validate はActivityInterfaceの実装
func (ra *ReadActivity) Validate(act *Activity, world w.World) error {
	// 読書対象（本）が必要
	if act.Target == nil {
		return fmt.Errorf("読書には本が必要です")
	}

	// 本が実際に存在し、読書可能かをチェック
	targetEntity := *act.Target

	// Itemコンポーネントを持つかチェック
	if !targetEntity.HasComponent(world.Components.Item) {
		return fmt.Errorf("読書対象がアイテムではありません")
	}

	// バックパック内にあるかチェック
	if !targetEntity.HasComponent(world.Components.ItemLocationInPlayerBackpack) {
		return fmt.Errorf("本がバックパック内にありません")
	}

	// TODO: より詳細な読書可能チェック
	// - 十分な明るさがあるか
	// - 読書スキルが適切か
	// - 本が読める状態か（破損していないか）

	return nil
}

// Start は読書開始時の処理を実行する
// Start はActivityInterfaceの実装
func (ra *ReadActivity) Start(act *Activity, world w.World) error {
	act.Logger.Debug("読書開始", "actor", act.Actor, "target", *act.Target, "duration", act.TurnsLeft)

	// 読書開始メッセージ
	targetEntity := *act.Target
	itemName := "本"
	if nameComp := world.Components.Name.Get(targetEntity); nameComp != nil {
		name := nameComp.(*gc.Name)
		itemName = name.Name
	}

	// プレイヤーの場合のみ読書開始メッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("読書を開始した: ").
			ItemName(itemName).
			Log()
	}

	return nil
}

// DoTurn は読書アクティビティの1ターン分の処理を実行する
// DoTurn はActivityInterfaceの実装
func (ra *ReadActivity) DoTurn(act *Activity, world w.World) error {
	// 読書条件を再チェック
	if err := ra.Validate(act, world); err != nil {
		act.Cancel(fmt.Sprintf("読書条件が満たされません: %s", err.Error()))
		return err
	}

	// 基本のターン処理
	if act.TurnsLeft <= 0 {
		act.Complete()
		return nil
	}

	// 1ターン進行
	act.TurnsLeft--
	act.Logger.Debug("読書進行",
		"turns_left", act.TurnsLeft,
		"progress", act.GetProgressPercent())

	// 読書効果処理
	if err := ra.performReading(act, world); err != nil {
		act.Logger.Warn("読書効果処理エラー", "error", err.Error())
	}

	// 完了チェック
	if act.TurnsLeft <= 0 {
		act.Complete()
		return nil
	}

	return nil
}

// Finish は読書完了時の処理を実行する
// Finish はActivityInterfaceの実装
func (ra *ReadActivity) Finish(act *Activity, world w.World) error {
	act.Logger.Debug("読書完了", "actor", act.Actor)

	// 完了メッセージ
	targetEntity := *act.Target
	itemName := "本"
	if nameComp := world.Components.Name.Get(targetEntity); nameComp != nil {
		name := nameComp.(*gc.Name)
		itemName = name.Name
	}

	// プレイヤーの場合のみ読書完了メッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("読書を完了した: ").
			ItemName(itemName).
			Log()
	}

	// TODO: 読書完了による効果
	// - スキル向上
	// - 知識獲得
	// - レシピ習得など

	return nil
}

// Canceled は読書キャンセル時の処理を実行する
// Canceled はActivityInterfaceの実装
func (ra *ReadActivity) Canceled(act *Activity, world w.World) error {
	// プレイヤーの場合のみ中断時のメッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("読書が中断された: ").
			Append(act.CancelReason).
			Log()
	}

	act.Logger.Debug("読書中断", "reason", act.CancelReason, "progress", act.GetProgressPercent())
	return nil
}

// performReading は読書効果処理を実行する
func (ra *ReadActivity) performReading(act *Activity, world w.World) error {
	// 読書対象の有効性チェック
	if act.Target == nil {
		return fmt.Errorf("読書対象が指定されていません")
	}

	// TODO: 読書による効果実装
	// - スキル経験値の獲得
	// - 知識ポイントの蓄積
	// - 特殊効果の発動

	// プレイヤーの場合のみ10ターン毎に集中メッセージを表示
	if isPlayerActivity(act, world) && act.TurnsTotal-act.TurnsLeft > 0 && (act.TurnsTotal-act.TurnsLeft)%10 == 0 {
		gamelog.New(gamelog.FieldLog).
			Append("読書に集中している...").
			Log()
	}

	return nil
}
