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

// Info はBehaviorの実装
func (rb *RestBehavior) Info() Info {
	return Info{
		Name:            "休息",
		Description:     "体力を回復するために休息する",
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
		return fmt.Errorf("周囲に敵がいるため休息できません")
	}

	// 必要量が妥当かチェック
	if comp.Progress.Max <= 0 {
		return fmt.Errorf("休息の必要量が無効です")
	}

	return nil
}

// Start は休息開始時の処理を実行する
func (rb *RestBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("休息開始", "actor", actor, "required", comp.Progress.Max)
	return nil
}

// DoTurn は休息アクティビティの1ターン分の処理を実行する
func (rb *RestBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 周囲の安全性をチェック
	if !isAreaSafe(actor, world) {
		Cancel(comp, "周囲に敵がいるため休息を中断")
		return fmt.Errorf("周囲に敵がいるため休息できません")
	}

	// 今ターンのAPを注ぐ。APが高いほど速く休息が進む
	comp.Progress.Current += perTurnAP(actor, world)
	log.Debug("休息進行", "progress", GetProgressPercent(comp))

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
	log.Debug("休息完了", "actor", actor)

	// プレイヤーの場合のみ完了メッセージを表示
	if world.Components.Player.Has(actor) {
		gamelog.New(query.GetGameLog(world)).
			Append("十分な休息を取って体力を回復した").
			Log()
	}

	// 最終的なHP回復（ボーナス）
	if world.Components.HP.Has(actor) {
		hp := world.Components.HP.Get(actor)
		if hp.Current < hp.Max {
			bonusHealing := 2
			hp.Current += bonusHealing
			if hp.Current > hp.Max {
				hp.Current = hp.Max
			}

			gamelog.New(query.GetGameLog(world)).
				Append("完全な休息により追加で ").
				Append(fmt.Sprintf("%d", bonusHealing)).
				Append(" HP回復した").
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
			Append("休息が中断された: ").
			Append(comp.CancelReason).
			Log()
	}

	log.Debug("休息中断", "reason", comp.CancelReason, "progress", GetProgressPercent(comp))
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

	// 直接HP回復（1ターンあたり5HP）
	healAmount := 5
	beforeHP := hp.Current
	hp.Current += healAmount
	if hp.Current > hp.Max {
		hp.Current = hp.Max
	}
	actualHealing := hp.Current - beforeHP

	log.Debug("HP回復", "actor", actor, "amount", actualHealing)
	return nil
}
