package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// RestActivity はBehaviorの実装
type RestActivity struct{}

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

// Validate は休息アクティビティの検証を行う
func (ra *RestActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// 周囲の安全性をチェック
	if !ra.isSafe(actor, world) {
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
	if !ra.isSafe(actor, world) {
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
		log.Warn("HP回復処理エラー", "error", err.Error())
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
		gamelog.New(gamelog.FieldLog).
			Append("十分な休息を取って体力を回復した").
			Log()
	}

	// 最終的なHP回復（ボーナス）
	poolsComponent := world.Components.Pools.Get(actor)
	if poolsComponent != nil {
		pools := poolsComponent.(*gc.Pools)
		if pools.HP.Current < pools.HP.Max {
			bonusHealing := 5 / 2 // 完了ボーナス
			pools.HP.Current += bonusHealing
			if pools.HP.Current > pools.HP.Max {
				pools.HP.Current = pools.HP.Max
			}

			gamelog.New(gamelog.FieldLog).
				Append("完全な休息により追加で ").
				Append(fmt.Sprintf("%d", bonusHealing)).
				Append(" HP回復した").
				Log()
		}

		// SPも少し回復
		if pools.SP.Current < pools.SP.Max {
			bonusStamina := 10
			pools.SP.Current += bonusStamina
			if pools.SP.Current > pools.SP.Max {
				pools.SP.Current = pools.SP.Max
			}

			log.Debug("スタミナ回復", "bonus", bonusStamina, "current", pools.SP.Current)
		}
	}

	return nil
}

// Canceled は休息キャンセル時の処理を実行する
func (ra *RestActivity) Canceled(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// プレイヤーの場合のみ中断時のメッセージを表示
	if actor.HasComponent(world.Components.Player) {
		gamelog.New(gamelog.FieldLog).
			Append("休息が中断された: ").
			Append(comp.CancelReason).
			Log()
	}

	log.Debug("休息中断", "reason", comp.CancelReason, "progress", GetProgressPercent(comp))
	return nil
}

// performHealing はHP回復処理を実行する
func (ra *RestActivity) performHealing(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	// Poolsコンポーネントを取得
	poolsComponent := world.Components.Pools.Get(actor)
	if poolsComponent == nil {
		// HPコンポーネントがない場合はスキップ（エラーにしない）
		return nil
	}

	pools, ok := poolsComponent.(*gc.Pools)
	if !ok {
		return fmt.Errorf("Poolsコンポーネントの型変換に失敗しました")
	}
	if pools.HP.Current >= pools.HP.Max {
		// 既に満タンの場合は早期完了
		Complete(comp)
		return nil
	}

	// 直接HP回復（1ターンあたり5HP）
	healAmount := 5
	beforeHP := pools.HP.Current
	pools.HP.Current += healAmount
	if pools.HP.Current > pools.HP.Max {
		pools.HP.Current = pools.HP.Max
	}
	actualHealing := pools.HP.Current - beforeHP

	// 5ターン毎にゲームログ出力（プレイヤーの場合のみ）
	if actor.HasComponent(world.Components.Player) && comp.TurnsTotal-comp.TurnsLeft > 0 && (comp.TurnsTotal-comp.TurnsLeft)%5 == 0 {
		gamelog.New(gamelog.FieldLog).
			Append(fmt.Sprintf("HPが %d 回復した。", actualHealing)).
			Log()
	}

	log.Debug("HP回復", "actor", actor, "amount", actualHealing)
	return nil
}

// isSafe は周囲が安全かをチェックする
func (ra *RestActivity) isSafe(actor ecs.Entity, world w.World) bool {
	// プレイヤーの位置を取得
	gridElement := world.Components.GridElement.Get(actor)
	if gridElement == nil {
		return false
	}

	playerGrid := gridElement.(*gc.GridElement)
	playerX, playerY := int(playerGrid.X), int(playerGrid.Y)

	// 近くに敵がいないかチェック（3x3の範囲）
	safeRadius := 1
	hasEnemies := false

	world.Manager.Join(
		world.Components.FactionEnemy,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		enemyGrid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		enemyX, enemyY := int(enemyGrid.X), int(enemyGrid.Y)

		// 距離チェック
		dx, dy := enemyX-playerX, enemyY-playerY
		if dx >= -safeRadius && dx <= safeRadius && dy >= -safeRadius && dy <= safeRadius {
			hasEnemies = true
		}
	}))

	return !hasEnemies
}
