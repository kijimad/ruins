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
type WaitBehavior struct{}

// Info はBehaviorの実装
func (wb *WaitBehavior) Info() Info {
	return Info{
		Name:            "待機",
		Description:     "指定した時間だけ待機する",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: consts.StandardActionCost,
	}
}

// Name はBehaviorの実装
func (wb *WaitBehavior) Name() gc.BehaviorName {
	return gc.BehaviorWait
}

// NewWaitActivity は待機回数を指定して待機アクティビティを組む。待機は時間経過なので
// Progress.Max に待機回数を直接据える。
func NewWaitActivity(duration consts.Turn) *gc.Activity {
	turns := int(duration)
	if turns <= 0 {
		turns = 1
	}
	return NewActivity(gc.BehaviorWait, turns)
}

// Validate は待機アクティビティの検証を行う
func (wb *WaitBehavior) Validate(comp *gc.Activity, _ ecs.Entity, _ w.World) error {
	// 待機は基本的に常に実行可能
	// ただし、最低限のチェックは行う

	// 待機回数が妥当かチェック
	if comp.Progress.Max <= 0 {
		return fmt.Errorf("待機回数が無効です")
	}

	return nil
}

// Start は待機開始時の処理を実行する
func (wb *WaitBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("待機開始", "actor", actor, "required", comp.Progress.Max)
	return nil
}

// DoTurn は待機アクティビティの1ターン分の処理を実行する
func (wb *WaitBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 長い待機は敵が近づいたら中断する。1回だけのターン送りとAIの手番調整は
	// その場で完結する行動なので対象にしない
	if comp.Progress.Max > 1 && !isAreaSafe(actor, world) {
		Cancel(comp, "周囲に敵がいるため待機を中断")
		return nil
	}

	// 1ターン進行
	comp.Progress.Current++
	log.Debug("待機進行", "progress", GetProgressPercent(comp))

	// 完了チェック
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}

	return nil
}

// Finish は待機完了時の処理を実行する
func (wb *WaitBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("待機完了", "actor", actor)

	// 複数ターン待機の場合のみログを表示する
	if comp.Progress.Max > 1 && world.Components.Player.Has(actor) {
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
