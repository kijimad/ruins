package actions

import (
	"fmt"
	"math"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// 攻撃システムの定数
const (
	// 射程・距離関連
	MeleeAttackRange = 1.5 // 近接攻撃の最大射程（斜めも考慮）

	// 命中率関連
	BaseHitRate          = 80 // 基本命中率（%）
	HitRatePerStatPoint  = 2  // 器用度と敏捷度の差1点あたりの命中率変化（%）
	MaxHitRate           = 95 // 最大命中率（%）
	MinHitRate           = 5  // 最小命中率（%）
	CriticalHitThreshold = 5  // クリティカルヒット判定しきい値（%以下）

	// ダメージ関連
	DamageRandomRange        = 6 // ダメージのランダム要素（1-6）
	CriticalDamageMultiplier = 3 // クリティカルダメージ倍率の分子
	CriticalDamageBase       = 2 // クリティカルダメージ倍率の分母（3/2 = 1.5倍）
	MinDamage                = 1 // 最低保証ダメージ

	// 確率計算関連
	DiceMax = 100 // ダイス最大値（1-100）
)

// AttackActivity はActivityInterfaceの実装
type AttackActivity struct{}

// Info はActivityInterfaceの実装
func (aa *AttackActivity) Info() ActivityInfo {
	return ActivityInfo{
		Name:            "攻撃",
		Description:     "敵を攻撃する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: 100,
		TotalRequiredAP: 100,
	}
}

// String はActivityInterfaceの実装
func (aa *AttackActivity) String() string {
	return "Attack"
}

// Validate はActivityInterfaceの実装
func (aa *AttackActivity) Validate(act *Activity, world w.World) error {
	if act.Target == nil {
		return ErrAttackTargetNotSet
	}

	if act.Actor.HasComponent(world.Components.Dead) {
		return ErrAttackerDead
	}

	if !act.Target.HasComponent(world.Components.GridElement) {
		return ErrAttackTargetNotExists
	}

	if act.Target.HasComponent(world.Components.Dead) {
		return ErrAttackTargetDead
	}

	if !aa.isInRange(act.Actor, *act.Target, world) {
		return ErrAttackOutOfRange
	}

	if !aa.canPerformAttack(act.Actor, world) {
		return ErrAttackNoWeapon
	}

	return nil
}

// Start はActivityInterfaceの実装
func (aa *AttackActivity) Start(act *Activity, _ w.World) error {
	act.Logger.Debug("攻撃開始", "actor", act.Actor, "target", *act.Target)
	return nil
}

// DoTurn はActivityInterfaceの実装
func (aa *AttackActivity) DoTurn(act *Activity, world w.World) error {
	if act.Target == nil {
		act.Cancel("攻撃対象が設定されていません")
		return ErrAttackTargetNotSet
	}

	if !aa.canAttack(act, world) {
		act.Cancel("攻撃できません")
		return ErrAttackTargetInvalid
	}

	if err := aa.performAttack(act, world); err != nil {
		act.Cancel(fmt.Sprintf("攻撃エラー: %s", err.Error()))
		return err
	}

	act.Complete()
	return nil
}

// Finish はActivityInterfaceの実装
func (aa *AttackActivity) Finish(act *Activity, _ w.World) error {
	act.Logger.Debug("攻撃アクティビティ完了",
		"actor", act.Actor,
		"target", *act.Target)

	return nil
}

// Canceled はActivityInterfaceの実装
func (aa *AttackActivity) Canceled(act *Activity, _ w.World) error {
	act.Logger.Debug("攻撃キャンセル", "actor", act.Actor, "reason", act.CancelReason)
	return nil
}

func (aa *AttackActivity) performAttack(act *Activity, world w.World) error {
	attacker := act.Actor
	target := *act.Target

	act.Logger.Debug("攻撃実行", "attacker", attacker, "target", target)

	// 攻撃方法を取得
	_, attackMethodName, err := aa.getAttackParams(attacker, world)
	if err != nil {
		return fmt.Errorf("攻撃パラメータの取得に失敗: %w", err)
	}

	hit, criticalHit := aa.rollHitCheck(attacker, target, world)
	if !hit {
		aa.logAttackResult(attacker, target, world, false, false, 0, attackMethodName)
		return nil
	}

	damage := aa.calculateDamage(attacker, target, world, criticalHit)
	if damage < 0 {
		damage = 0
	}

	// ダメージを適用
	pools := world.Components.Pools.Get(target).(*gc.Pools)
	beforeHP := pools.HP.Current
	pools.HP.Current -= damage
	if pools.HP.Current < 0 {
		pools.HP.Current = 0
	}

	// 攻撃とダメージを1行でログ出力
	aa.logAttackResult(attacker, target, world, true, criticalHit, damage, attackMethodName)

	// 死亡チェックと死亡ログ
	if pools.HP.Current <= 0 && beforeHP > 0 {
		target.AddComponent(world.Components.Dead, &gc.Dead{})
		aa.logDeath(world, target)
	}

	return nil
}

func (aa *AttackActivity) canAttack(act *Activity, world w.World) bool {
	if act.Target == nil {
		return false
	}

	if err := aa.Validate(act, world); err != nil {
		return false
	}

	return true
}

func (aa *AttackActivity) isInRange(attacker, target ecs.Entity, world w.World) bool {
	attackerGrid := world.Components.GridElement.Get(attacker)
	if attackerGrid == nil {
		return false
	}

	targetGrid := world.Components.GridElement.Get(target)
	if targetGrid == nil {
		return false
	}

	attackerPos := attackerGrid.(*gc.GridElement)
	targetPos := targetGrid.(*gc.GridElement)

	dx := float64(attackerPos.X - targetPos.X)
	dy := float64(attackerPos.Y - targetPos.Y)
	distance := math.Sqrt(dx*dx + dy*dy)

	// TODO: 遠距離武器の場合は射程を武器から取得
	return distance <= MeleeAttackRange
}

func (aa *AttackActivity) canPerformAttack(attacker ecs.Entity, world w.World) bool {
	// TODO: 装備武器のチェック
	attrs := world.Components.Attributes.Get(attacker)
	return attrs != nil
}

func (aa *AttackActivity) rollHitCheck(attacker, target ecs.Entity, world w.World) (hit bool, critical bool) {
	attackerAttrs := world.Components.Attributes.Get(attacker).(*gc.Attributes)
	attackerDexterity := attackerAttrs.Dexterity.Total

	targetAttrs := world.Components.Attributes.Get(target).(*gc.Attributes)
	targetAgility := targetAttrs.Agility.Total

	baseHitRate := BaseHitRate + (attackerDexterity-targetAgility)*HitRatePerStatPoint

	weaponAccuracy := aa.getWeaponAccuracy(attacker, world)
	baseHitRate += weaponAccuracy

	if baseHitRate > MaxHitRate {
		baseHitRate = MaxHitRate
	}
	if baseHitRate < MinHitRate {
		baseHitRate = MinHitRate
	}

	roll := rand.IntN(DiceMax) + 1
	hit = roll <= baseHitRate
	critical = roll <= CriticalHitThreshold

	return hit, critical
}

func (aa *AttackActivity) calculateDamage(attacker, target ecs.Entity, world w.World, critical bool) int {
	attackerAttrs := world.Components.Attributes.Get(attacker).(*gc.Attributes)
	attackerStrength := attackerAttrs.Strength.Total

	targetAttrs := world.Components.Attributes.Get(target).(*gc.Attributes)
	targetDefense := targetAttrs.Defense.Total

	baseDamage := attackerStrength + rand.IntN(DamageRandomRange) + 1

	weaponDamage := aa.getWeaponDamage(attacker, world)
	baseDamage += weaponDamage

	if critical {
		baseDamage = baseDamage * CriticalDamageMultiplier / CriticalDamageBase
	}

	finalDamage := baseDamage - targetDefense
	if finalDamage < MinDamage {
		finalDamage = MinDamage
	}

	return finalDamage
}

// getWeaponDamage は攻撃者の武器から攻撃力を取得する
func (aa *AttackActivity) getWeaponDamage(attacker ecs.Entity, world w.World) int {
	attack, _, err := aa.getAttackParams(attacker, world)
	if err != nil || attack == nil {
		return 0
	}
	return attack.Damage
}

// getWeaponAccuracy は攻撃者の武器から命中率を取得する
func (aa *AttackActivity) getWeaponAccuracy(attacker ecs.Entity, world w.World) int {
	attack, _, err := aa.getAttackParams(attacker, world)
	if err != nil || attack == nil {
		return 0
	}
	// Accuracyは0-100なので、BaseHitRateとの差分を返す
	return attack.Accuracy - BaseHitRate
}

// getBareHandsAttack は素手武器の攻撃パラメータを取得する
func (aa *AttackActivity) getBareHandsAttack(world w.World) (*gc.Attack, string, error) {
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	bareHandsSpec, err := rawMaster.NewWeaponSpec("素手")
	if err != nil {
		return nil, "", fmt.Errorf("素手武器が見つかりません: %w", err)
	}
	if bareHandsSpec.Attack == nil {
		return nil, "", fmt.Errorf("素手武器にAttackコンポーネントがありません")
	}
	return bareHandsSpec.Attack, "素手", nil
}

// getAttackParams は攻撃者の武器から攻撃パラメータと攻撃方法名を取得する
// 戻り値: (攻撃パラメータ, 攻撃方法名, エラー)
func (aa *AttackActivity) getAttackParams(attacker ecs.Entity, world w.World) (*gc.Attack, string, error) {
	// プレイヤーの場合: 装備武器から攻撃パラメータを取得
	if attacker.HasComponent(world.Components.Player) {
		// 近接武器スロットから武器を取得
		meleeWeapon := worldhelper.GetMeleeWeapon(world, attacker)

		if meleeWeapon != nil {
			// 装備している武器から攻撃パラメータを取得
			attack, weaponName, err := worldhelper.GetAttackFromWeapon(world, *meleeWeapon)
			if err == nil && attack != nil {
				return attack, weaponName, nil
			}
		}

		// 装備していない場合は素手武器を使用
		return aa.getBareHandsAttack(world)
	}

	// 敵の場合: CommandTableから攻撃パラメータを取得
	if attacker.HasComponent(world.Components.CommandTable) {
		attack, weaponName, err := worldhelper.GetAttackFromCommandTable(world, attacker)
		if err == nil && attack != nil {
			return attack, weaponName, nil
		}

		// CommandTableから取得できない場合は素手武器を使用
		return aa.getBareHandsAttack(world)
	}

	return nil, "", fmt.Errorf("攻撃パラメータを取得できません: 攻撃者にPlayerまたはCommandTableコンポーネントがありません")
}

// logAttackResult は攻撃結果をログに出力する（ダメージも含む）
func (aa *AttackActivity) logAttackResult(attacker, target ecs.Entity, world w.World, hit bool, critical bool, damage int, attackMethodName string) {
	// プレイヤーが関わる攻撃のみログ出力
	if !isPlayerActivity(&Activity{Actor: attacker}, world) && !isPlayerActivity(&Activity{Actor: target}, world) {
		return
	}

	// 攻撃者名とターゲット名を取得
	attackerName := worldhelper.GetEntityName(attacker, world)
	targetName := worldhelper.GetEntityName(target, world)

	gamelog.New(gamelog.FieldLog).
		Build(func(l *gamelog.Logger) {
			worldhelper.AppendNameWithColor(l, attacker, attackerName, world)
		}).
		Append(" は ").
		Build(func(l *gamelog.Logger) {
			// 攻撃方法がある場合は表示
			if attackMethodName != "" {
				l.Append(attackMethodName).Append(" で ")
			}
			worldhelper.AppendNameWithColor(l, target, targetName, world)
		}).
		Build(func(l *gamelog.Logger) {
			if !hit {
				l.Append(" を攻撃したが外れた。")
			} else if critical {
				l.Append(fmt.Sprintf(" にクリティカルヒットし、%d のダメージを与えた！", damage))
			} else {
				l.Append(fmt.Sprintf(" を攻撃し、%d のダメージを与えた。", damage))
			}
		}).
		Log()
}

// logDeath は死亡ログを出力する
func (aa *AttackActivity) logDeath(world w.World, target ecs.Entity) {
	targetName := worldhelper.GetEntityName(target, world)

	gamelog.New(gamelog.FieldLog).
		Build(func(l *gamelog.Logger) {
			worldhelper.AppendNameWithColor(l, target, targetName, world)
		}).
		Append(" は倒れた。").
		Log()
}
