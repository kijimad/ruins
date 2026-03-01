package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// WaitActivity はBehaviorの実装
type WaitActivity struct{}

// Info はBehaviorの実装
func (wa *WaitActivity) Info() Info {
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
func (wa *WaitActivity) Name() gc.BehaviorName {
	return gc.BehaviorWait
}

// Validate は待機アクティビティの検証を行う
func (wa *WaitActivity) Validate(comp *gc.CurrentActivity, _ ecs.Entity, _ w.World) error {
	// 待機は基本的に常に実行可能
	// ただし、最低限のチェックは行う

	// 待機時間が妥当かチェック
	if comp.TurnsTotal <= 0 {
		return fmt.Errorf("待機時間が無効です")
	}

	return nil
}

// Start は待機開始時の処理を実行する
func (wa *WaitActivity) Start(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	reason := "時間を過ごすため"
	log.Debug("待機開始", "actor", actor, "reason", reason, "duration", comp.TurnsLeft)
	return nil
}

// DoTurn は待機アクティビティの1ターン分の処理を実行する
func (wa *WaitActivity) DoTurn(comp *gc.CurrentActivity, actor ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)

	// 環境を観察
	wa.observeEnvironment(comp, actor, world)

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
func (wa *WaitActivity) Finish(_ *gc.CurrentActivity, actor ecs.Entity, world w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("待機完了", "actor", actor)

	// TODO: 1ターン待機の場合も出るのは微妙な感じがする
	// プレイヤーの場合のみ待機完了メッセージを表示
	if actor.HasComponent(world.Components.Player) {
		gamelog.New(gamelog.FieldLog).
			Append("待機を終了した").
			Log()
	}

	return nil
}

// Canceled は待機キャンセル時の処理を実行する
func (wa *WaitActivity) Canceled(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) error {
	log := logger.New(logger.CategoryAction)
	log.Debug("待機キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

// observeEnvironment は環境観察処理を実行する
func (wa *WaitActivity) observeEnvironment(comp *gc.CurrentActivity, actor ecs.Entity, _ w.World) {
	log := logger.New(logger.CategoryAction)
	// 待機中の環境観察（5ターン毎）
	if (comp.TurnsTotal-comp.TurnsLeft)%5 == 0 {
		// TODO: 環境観察の実装
		// - 周囲の敵の発見
		// - アイテムの発見
		// - 天候の変化など
		log.Debug("環境観察", "actor", actor)
	}
}
