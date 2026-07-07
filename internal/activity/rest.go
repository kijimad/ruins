package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// RestActivity はBehaviorの実装
type RestActivity struct {
	Duration int
}

// Info はBehaviorの実装
func (ra *RestActivity) Info() Info {
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
func (ra *RestActivity) Name() gc.BehaviorName {
	return gc.BehaviorRest
}

// BuildActivity はBehaviorの実装
func (ra *RestActivity) BuildActivity(actor ecs.Entity, world w.World) (*gc.Activity, error) {
	duration := ra.Duration
	if duration <= 0 {
		characterAP, err := getEntityMaxAP(actor, world)
		if err != nil {
			return nil, err
		}
		duration = CalculateRequiredTurns(ra, characterAP)
	}
	comp, err := NewActivity(ra, duration)
	if err != nil {
		return nil, err
	}
	return comp, nil
}

// Validate は休息アクティビティの検証を行う
func (ra *RestActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 周囲の安全性をチェック
	if !isAreaSafe(actor, world) {
		return fmt.Errorf("周囲に敵がいるため休息できません")
	}

	// 休息時間が妥当かチェック
	if comp.TurnsTotal <= 0 {
		return fmt.Errorf("休息時間が無効です")
	}

	return nil
}

// Start は休息開始時の処理を実行する
func (ra *RestActivity) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("休息開始", "actor", actor, "duration", comp.TurnsLeft)
	return nil
}

// DoTurn は休息アクティビティの1ターン分の処理を実行する
func (ra *RestActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 周囲の安全性をチェック
	if !isAreaSafe(actor, world) {
		Cancel(comp, "周囲に敵がいるため休息を中断")
		return fmt.Errorf("周囲に敵がいるため休息できません")
	}

	// 基本のターン処理
	if comp.TurnsLeft <= 0 {
		Complete(comp)
		return nil
	}

	// 1ターン進行
	comp.TurnsLeft--
	log.Debug("休息進行",
		"turns_left", comp.TurnsLeft,
		"progress", GetProgressPercent(comp))

	// HP回復処理
	if err := ra.performHealing(comp, actor, world); err != nil {
		return err
	}

	// 完了チェック
	if comp.TurnsLeft <= 0 {
		Complete(comp)
		return nil
	}

	return nil
}

// Finish は休息完了時の処理を実行する
func (ra *RestActivity) Finish(_ *gc.Activity, actor ecs.Entity, world w.World) error {
	log.Debug("休息完了", "actor", actor)

	// プレイヤーの場合のみ完了メッセージを表示
	if actor.HasComponent(world.Components.Player) {
		gamelog.New(query.GetGameLog(world)).
			Append("十分な休息を取って体力を回復した").
			Log()
	}

	// 最終的なHP回復（ボーナス）
	hpComponent := world.Components.HP.Get(actor)
	if hpComponent != nil {
		hp := hpComponent.(*gc.HP)
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
func (ra *RestActivity) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// プレイヤーの場合のみ中断時のメッセージを表示
	if actor.HasComponent(world.Components.Player) {
		gamelog.New(query.GetGameLog(world)).
			Append("休息が中断された: ").
			Append(comp.CancelReason).
			Log()
	}

	log.Debug("休息中断", "reason", comp.CancelReason, "progress", GetProgressPercent(comp))
	return nil
}

// performHealing はHP回復処理を実行する
func (ra *RestActivity) performHealing(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	hpComponent := world.Components.HP.Get(actor)
	if hpComponent == nil {
		return nil
	}

	hp, ok := hpComponent.(*gc.HP)
	if !ok {
		return fmt.Errorf("HPコンポーネントの型変換に失敗しました")
	}
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

	// 5ターン毎にゲームログ出力（プレイヤーの場合のみ）
	if actor.HasComponent(world.Components.Player) && comp.TurnsTotal-comp.TurnsLeft > 0 && (comp.TurnsTotal-comp.TurnsLeft)%5 == 0 {
		gamelog.New(query.GetGameLog(world)).
			Append(fmt.Sprintf("HPが %d 回復した。", actualHealing)).
			Log()
	}

	log.Debug("HP回復", "actor", actor, "amount", actualHealing)
	return nil
}
