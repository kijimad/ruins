package actions

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// RestActivity はActivityInterfaceの実装
type RestActivity struct{}

// Info はActivityInterfaceの実装
func (ra *RestActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "休息",
		Description:     "体力を回復するために休息する",
		Interruptible:   true,
		Resumable:       true,
		ActionPointCost: 100,
		TotalRequiredAP: 1000,
	}
}

// String はActivityInterfaceの実装
func (ra *RestActivity) String() string {
	return "Rest"
}

// Validate は休息アクティビティの検証を行う
// Validate はActivityInterfaceの実装
func (ra *RestActivity) Validate(act *Activity, world w.World) error {
	// 周囲の安全性をチェック
	if !ra.isSafe(act, world) {
		return fmt.Errorf("周囲に敵がいるため休息できません")
	}

	// 休息時間が妥当かチェック
	if act.TurnsTotal <= 0 {
		return fmt.Errorf("休息時間が無効です")
	}

	return nil
}

// Start は休息開始時の処理を実行する
// Start はActivityInterfaceの実装
func (ra *RestActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("休息開始", "actor", act.Actor, "duration", act.TurnsLeft)
	return nil
}

// DoTurn は休息アクティビティの1ターン分の処理を実行する
// DoTurn はActivityInterfaceの実装
func (ra *RestActivity) DoTurn(act *Activity, world w.World) error {
	// 周囲の安全性をチェック
	if !ra.isSafe(act, world) {
		act.Cancel("周囲に敵がいるため休息を中断")
		return fmt.Errorf("周囲に敵がいるため休息できません")
	}

	// 基本のターン処理
	if act.TurnsLeft <= 0 {
		act.Complete()
		return nil
	}

	// 1ターン進行
	act.TurnsLeft--
	act.Logger.Debug("休息進行",
		"turns_left", act.TurnsLeft,
		"progress", act.GetProgressPercent())

	// HP回復処理
	if err := ra.performHealing(act, world); err != nil {
		act.Logger.Warn("HP回復処理エラー", "error", err.Error())
	}

	// 完了チェック
	if act.TurnsLeft <= 0 {
		act.Complete()
		return nil
	}

	return nil
}

// Finish は休息完了時の処理を実行する
// Finish はActivityInterfaceの実装
func (ra *RestActivity) Finish(act *Activity, world w.World) error {
	act.Logger.Debug("休息完了", "actor", act.Actor)

	// プレイヤーの場合のみ完了メッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("十分な休息を取って体力を回復した").
			Log()
	}

	// 最終的なHP回復（ボーナス）
	poolsComponent := world.Components.Pools.Get(act.Actor)
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

			act.Logger.Debug("スタミナ回復", "bonus", bonusStamina, "current", pools.SP.Current)
		}
	}

	return nil
}

// Canceled は休息キャンセル時の処理を実行する
// Canceled はActivityInterfaceの実装
func (ra *RestActivity) Canceled(act *Activity, world w.World) error {
	// プレイヤーの場合のみ中断時のメッセージを表示
	if isPlayerActivity(act, world) {
		gamelog.New(gamelog.FieldLog).
			Append("休息が中断された: ").
			Append(act.CancelReason).
			Log()
	}

	act.Logger.Debug("休息中断", "reason", act.CancelReason, "progress", act.GetProgressPercent())
	return nil
}

// performHealing はHP回復処理を実行する
func (ra *RestActivity) performHealing(act *Activity, world w.World) error {
	// アクターの有効性チェック
	if act.Actor == 0 {
		return fmt.Errorf("休息するエンティティが指定されていません")
	}

	// Poolsコンポーネントを取得
	poolsComponent := world.Components.Pools.Get(act.Actor)
	if poolsComponent == nil {
		// HPコンポーネントがない場合はスキップ（エラーにしない）
		return nil
	}

	pools := poolsComponent.(*gc.Pools)
	if pools.HP.Current >= pools.HP.Max {
		// 既に満タンの場合は早期完了
		act.Complete()
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
	if isPlayerActivity(act, world) && act.TurnsTotal-act.TurnsLeft > 0 && (act.TurnsTotal-act.TurnsLeft)%5 == 0 {
		gamelog.New(gamelog.FieldLog).
			Append(fmt.Sprintf("HPが %d 回復した。", actualHealing)).
			Log()
	}

	act.Logger.Debug("HP回復", "actor", act.Actor, "amount", actualHealing)
	return nil
}

// isSafe は周囲が安全かをチェックする
func (ra *RestActivity) isSafe(act *Activity, world w.World) bool {
	// プレイヤーの位置を取得
	gridElement := world.Components.GridElement.Get(act.Actor)
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
