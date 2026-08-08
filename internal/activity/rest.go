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

// RestBehavior はBehaviorの実装
type RestBehavior struct{}

const (
	restHealPerTurn       = 5 // 1ターンあたりの直接HP回復量
	restFullRestBonusHeal = 2 // 完全休息を終えたときの追加HP回復量
)

// Info はBehaviorの実装
func (rb *RestBehavior) Info() Info {
	return Info{
		Name:            "Rest",
		Description:     "Rest to recover health",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 1000,
	}
}

// Name はBehaviorの実装
func (rb *RestBehavior) Name() gc.BehaviorName {
	return gc.BehaviorRest
}

// NewRestActivity は休息アクティビティを組む。必要総APは Info の TotalRequiredAP。
func NewRestActivity() *gc.Activity {
	return NewActivity(gc.BehaviorRest, (&RestBehavior{}).Info().TotalRequiredAP)
}

// Validate は休息アクティビティの検証を行う
func (rb *RestBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 周囲の安全性をチェック
	if !isAreaSafe(actor, world) {
		return fmt.Errorf("cannot rest because enemies are nearby")
	}

	// 必要量が妥当かチェック
	if comp.Progress.Max <= 0 {
		return fmt.Errorf("rest requirement amount is invalid")
	}

	return nil
}

// Start は休息開始時の処理を実行する
func (rb *RestBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("rest started", "actor", actor, "required", comp.Progress.Max)
	return nil
}

// DoTurn は休息アクティビティの1ターン分の処理を実行する
func (rb *RestBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 周囲の安全性をチェック
	if !isAreaSafe(actor, world) {
		Cancel(comp, "rest interrupted because enemies are nearby")
		return fmt.Errorf("cannot rest because enemies are nearby")
	}

	// 今ターンのAPを注ぐ。APが高いほど速く休息が進む
	comp.Progress.Current += perTurnAP(actor, world)
	log.Debug("rest progressing", "progress", GetProgressPercent(comp))

	// HP回復処理。満タンなら早期完了する
	if err := rb.performHealing(comp, actor, world); err != nil {
		return err
	}

	// 必要量に達したら完了
	if comp.Progress.Current >= comp.Progress.Max {
		Complete(comp)
	}

	return nil
}

// Finish は休息完了時の処理を実行する
func (rb *RestBehavior) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("rest finished", "actor", actor)

	// プレイヤーの場合のみ完了メッセージを表示
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Rested well and recovered health")).
			Log()
	}

	// 最終的なHP回復（ボーナス）
	if world.Components.HP.Has(actor) {
		hp := world.Components.HP.Get(actor)
		if hp.Current < hp.Max {
			bonusHealing := restFullRestBonusHeal
			hp.Current += bonusHealing
			if hp.Current > hp.Max {
				hp.Current = hp.Max
			}

			gamelog.New(query.GetGameLog(world)).
				Markup(query.T(world, "Full rest recovered an additional %d HP", bonusHealing)).
				Log()
		}
	}

	return nil
}

// Canceled は休息キャンセル時の処理を実行する
func (rb *RestBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// プレイヤーの場合のみ中断時のメッセージを表示
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Rest interrupted: %s", query.T(world, comp.CancelReason))).
			Log()
	}

	log.Debug("rest interrupted", "reason", comp.CancelReason, "progress", GetProgressPercent(comp))
	return nil
}

// performHealing はHP回復処理を実行する
func (rb *RestBehavior) performHealing(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if !world.Components.HP.Has(actor) {
		return nil
	}
	hp := world.Components.HP.Get(actor)
	if hp.Current >= hp.Max {
		// 既に満タンの場合は早期完了
		Complete(comp)
		return nil
	}

	healAmount := restHealPerTurn
	beforeHP := hp.Current
	hp.Current += healAmount
	if hp.Current > hp.Max {
		hp.Current = hp.Max
	}
	actualHealing := hp.Current - beforeHP

	log.Debug("HP recovered", "actor", actor, "amount", actualHealing)
	return nil
}
