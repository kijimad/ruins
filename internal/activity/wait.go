package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// WaitBehavior はBehaviorの実装
type WaitBehavior struct {
	Duration consts.Turn
	Reason   string
}

// Info はBehaviorの実装
func (wb *WaitBehavior) Info() Info {
	return Info{
		Name:            "待機",
		Description:     "指定した時間だけ待機する",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 500,
	}
}

// Name はBehaviorの実装
func (wb *WaitBehavior) Name() gc.BehaviorName {
	return gc.BehaviorWait
}

// BuildActivity はBehaviorの実装。待機は作業ではなく時間経過なので、
// 待機回数を Required に据え、DoTurn で毎ターン 1 ずつ注ぐ。
func (wb *WaitBehavior) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	turns := int(wb.Duration)
	if turns <= 0 {
		turns = 1
	}
	return NewActivity(wb, turns)
}

// Validate は待機アクティビティの検証を行う
func (wb *WaitBehavior) Validate(comp *gc.Activity, _ ecs.Entity, _ w.World) error {
	// 待機は基本的に常に実行可能
	// ただし、最低限のチェックは行う

	// 待機回数が妥当かチェック
	if comp.Required <= 0 {
		return fmt.Errorf("待機回数が無効です")
	}

	return nil
}

// Start は待機開始時の処理を実行する
func (wb *WaitBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("待機開始", "actor", actor, "reason", wb.Reason, "required", comp.Required)
	return nil
}

// DoTurn は待機アクティビティの1ターン分の処理を実行する
func (wb *WaitBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 長い待機は敵が近づいたら中断する。1回だけのターン送りとAIの手番調整は
	// その場で完結する行動なので対象にしない
	if comp.Required > 1 && !isAreaSafe(actor, world) {
		Cancel(comp, "周囲に敵がいるため待機を中断")
		return nil
	}

	// 1ターン進行
	comp.Accumulated++
	log.Debug("待機進行", "progress", GetProgressPercent(comp))

	// 完了チェック
	if comp.Accumulated >= comp.Required {
		Complete(comp)
	}

	return nil
}

// Finish は待機完了時の処理を実行する
func (wb *WaitBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("待機完了", "actor", actor)

	// 複数ターン待機の場合のみログを表示する
	if comp.Required > 1 && world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Append("待機を終了した").
			Log()
	}

	return nil
}

// Canceled は待機キャンセル時の処理を実行する
func (wb *WaitBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("待機キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}
