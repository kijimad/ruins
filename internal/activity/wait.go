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

// BuildActivity はBehaviorの実装
func (wb *WaitBehavior) BuildActivity(_ ecs.Entity, _ w.World) (*gc.Activity, error) {
	duration := wb.Duration
	if duration <= 0 {
		duration = 1
	}
	comp, err := NewActivity(wb, duration)
	if err != nil {
		return nil, err
	}
	return comp, nil
}

// Validate は待機アクティビティの検証を行う
func (wb *WaitBehavior) Validate(comp *gc.Activity, _ ecs.Entity, _ w.World) error {
	// 待機は基本的に常に実行可能
	// ただし、最低限のチェックは行う

	// 待機時間が妥当かチェック
	if comp.TurnsTotal <= 0 {
		return fmt.Errorf("待機時間が無効です")
	}

	return nil
}

// Start は待機開始時の処理を実行する
func (wb *WaitBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("待機開始", "actor", actor, "reason", wb.Reason, "duration", comp.TurnsLeft)
	return nil
}

// DoTurn は待機アクティビティの1ターン分の処理を実行する
func (wb *WaitBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 長い待機は敵が近づいたら中断する。1ターンのターン送りとAIの手番調整は
	// その場で完結する行動なので対象にしない
	if comp.TurnsTotal > 1 && !isAreaSafe(actor, world) {
		Cancel(comp, "周囲に敵がいるため待機を中断")
		return nil
	}

	// 環境を観察
	wb.observeEnvironment(comp, actor, world)

	// 基本のターン処理
	if comp.TurnsLeft <= 0 {
		Complete(comp)
		return nil
	}

	// 1ターン進行
	comp.TurnsLeft--
	log.Debug("待機進行",
		"turns_left", comp.TurnsLeft,
		"progress", GetProgressPercent(comp))

	// 完了チェック
	if comp.TurnsLeft <= 0 {
		Complete(comp)
		return nil
	}

	return nil
}

// Finish は待機完了時の処理を実行する
func (wb *WaitBehavior) Finish(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("待機完了", "actor", actor)

	// 複数ターン待機の場合のみログを表示する
	if comp.TurnsTotal > 1 && world.Components.Player.Has(actor) {
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

// observeEnvironment は環境観察処理を実行する
func (wb *WaitBehavior) observeEnvironment(comp *gc.Activity, actor ecs.Entity, _ w.World) {
	// 待機中の環境観察（5ターン毎）
	if (comp.TurnsTotal-comp.TurnsLeft)%5 == 0 {
		// TODO: 環境観察の実装
		// - 周囲の敵の発見
		// - アイテムの発見
		// - 天候の変化など
		log.Debug("環境観察", "actor", actor)
	}
}
