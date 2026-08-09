package activity

import (
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
		Name:            "Wait",
		Description:     "Wait for a specified amount of time",
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

	// 待機回数が妥当かチェック。構築時に必ず正の値を据えるため、ここで非正なのは不変条件違反
	if comp.Progress.Max <= 0 {
		return ErrWaitInvalidDuration
	}

	return nil
}

// Start は待機開始時の処理を実行する
func (wb *WaitBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("wait started", "actor", actor, "required", comp.Progress.Max)
	return nil
}

// DoTurn は待機アクティビティの1ターン分の処理を実行する
func (wb *WaitBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 長い待機は敵が近づいたら中断する。1回だけのターン送りとAIの手番調整は
	// その場で完結する行動なので対象にしない
	if comp.Progress.Max > 1 && !isAreaSafe(actor, world) {
		Cancel(comp, "wait interrupted because enemies are nearby")
		return nil
	}

	// 1ターン進行
	comp.Progress.Current++
	log.Debug("wait progressing", "progress", GetProgressPercent(comp))

	// 完了チェック
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}

	return nil
}

// Finish は待機完了時の処理を実行する
func (wb *WaitBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("wait finished", "actor", actor)

	// 複数ターン待機の場合のみログを表示する
	if comp.Progress.Max > 1 && world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Finished waiting")).
			Log()
	}

	return nil
}

// Canceled は待機キャンセル時の処理を実行する
func (wb *WaitBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("wait canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}
