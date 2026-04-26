package activity

import (
	"fmt"
	"math"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/skill"
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

// AttackActivity はBehaviorの実装
type AttackActivity struct{}

// Info はBehaviorの実装
func (aa *AttackActivity) Info() Info {
	return Info{
		Name:            "攻撃",
		Description:     "敵を攻撃する",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (aa *AttackActivity) Name() gc.BehaviorName {
	return gc.BehaviorAttack
}

// Validate はBehaviorの実装
func (aa *AttackActivity) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		return ErrAttackTargetNotSet
	}

	if actor.HasComponent(world.Components.Dead) {
		return ErrAttackerDead
	}

	if !comp.Target.HasComponent(world.Components.GridElement) {
		return ErrAttackTargetNotExists
	}

	if comp.Target.HasComponent(world.Components.Dead) {
		return ErrAttackTargetDead
	}

	if !aa.isInRange(actor, *comp.Target, world) {
		return ErrAttackOutOfRange
	}

	if !aa.canPerformAttack(actor, world) {
		return ErrAttackNoWeapon
	}

	return nil
}

// Start はBehaviorの実装
func (aa *AttackActivity) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("攻撃開始", "actor", actor, "target", *comp.Target)
	return nil
}

// DoTurn はBehaviorの実装
func (aa *AttackActivity) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if comp.Target == nil {
		Cancel(comp, "攻撃対象が設定されていません")
		return ErrAttackTargetNotSet
	}

	if !aa.canAttack(comp, actor, world) {
		Cancel(comp, "攻撃できません")
		return ErrAttackTargetInvalid
	}

	if err := aa.performAttack(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("攻撃エラー: %s", err.Error()))
		return err
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (aa *AttackActivity) Finish(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("攻撃アクティビティ完了",
		"actor", actor,
		"target", *comp.Target)

	return nil
}

// Canceled はBehaviorの実装
func (aa *AttackActivity) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("攻撃キャンセル", "actor", actor, "reason", comp.CancelReason)
	return nil
}

func (aa *AttackActivity) performAttack(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	target := *comp.Target

	log.Debug("攻撃実行", "attacker", actor, "target", target)

	attack, attackMethodName, err := getAttackParams(actor, world)
	if err != nil {
		return fmt.Errorf("攻撃パラメータの取得に失敗: %w", err)
	}

	applyAttackDamage(actor, target, world, attack, attackMethodName, 0, 0)
	return nil
}

func (aa *AttackActivity) canAttack(comp *gc.Activity, actor ecs.Entity, world w.World) bool {
	if comp.Target == nil {
		return false
	}

	if err := aa.Validate(comp, actor, world); err != nil {
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
	abils := world.Components.Abilities.Get(attacker)
	return abils != nil
}

// getBareHandsAttack は素手武器の攻撃パラメータを取得する
func getBareHandsAttack(world w.World) (gc.Attacker, string, error) {
	rawMaster := world.Resources.RawMaster
	bareHandsSpec, err := rawMaster.NewWeaponSpec("素手")
	if err != nil {
		return nil, "", fmt.Errorf("素手武器が見つかりません: %w", err)
	}
	if bareHandsSpec.Melee == nil {
		return nil, "", fmt.Errorf("素手武器にMeleeコンポーネントがありません")
	}
	return bareHandsSpec.Melee, "素手", nil
}

// getAttackParams は攻撃者の武器から攻撃パラメータと攻撃方法名を取得する
// 戻り値: (攻撃パラメータ, 攻撃方法名, エラー)
func getAttackParams(attacker ecs.Entity, world w.World) (gc.Attacker, string, error) {
	// プレイヤーの場合: 装備武器から攻撃パラメータを取得
	if attacker.HasComponent(world.Components.Player) {
		// 選択中の武器スロット番号（1-5）から配列インデックスに変換
		selectedSlot := world.Resources.Dungeon.SelectedWeaponSlot
		weaponIndex := selectedSlot - 1 // 1-based to 0-based
		if weaponIndex < 0 || weaponIndex >= 5 {
			return nil, "", fmt.Errorf("無効な武器スロット番号: %d", selectedSlot)
		}

		weapons := worldhelper.GetWeapons(world, attacker)
		weapon := weapons[weaponIndex]
		if weapon != nil {
			// 装備している武器から攻撃パラメータを取得
			attack, weaponName, err := worldhelper.GetMeleeFromWeapon(world, *weapon)
			if err == nil && attack != nil {
				return attack, weaponName, nil
			}
		}

		// 武器が装備されていない場合は素手武器を使用
		return getBareHandsAttack(world)
	}

	// 敵の場合: CommandTableから攻撃パラメータを取得
	if attacker.HasComponent(world.Components.CommandTable) {
		attack, weaponName, err := worldhelper.GetAttackFromCommandTable(world, attacker)
		if err == nil && attack != nil {
			return attack, weaponName, nil
		}

		// CommandTableから取得できない場合は素手武器を使用
		return getBareHandsAttack(world)
	}

	return nil, "", fmt.Errorf("攻撃パラメータを取得できません: 攻撃者にPlayerまたはCommandTableコンポーネントがありません")
}

// getSkillMult は事前計算済みのスキル倍率(%)を返す。
// isDamageがtrueならWeaponDamage、falseならWeaponAccuracyを参照する。
// Effectsコンポーネントを持たないエンティティでは100(等倍)を返す。
func getSkillMult(entity ecs.Entity, attack gc.Attacker, world w.World, isDamage bool) int {
	if attack == nil {
		return 100
	}
	if !entity.HasComponent(world.Components.CharModifiers) {
		return 100
	}
	effects := world.Components.CharModifiers.Get(entity).(*gc.CharModifiers)
	skillID, ok := gc.WeaponSkillID(attack.GetAttackCategory())
	if !ok {
		return 100
	}
	var mults map[gc.SkillID]int
	if isDamage {
		mults = effects.WeaponDamage
	} else {
		mults = effects.WeaponAccuracy
	}
	if mult, ok := mults[skillID]; ok {
		return mult
	}
	return 100
}

// applyElementResist は事前計算済みの元素耐性倍率でダメージを軽減する
func applyElementResist(damage int, target ecs.Entity, element gc.ElementType, world w.World) int {
	if !target.HasComponent(world.Components.CharModifiers) {
		return damage
	}
	effects := world.Components.CharModifiers.Get(target).(*gc.CharModifiers)
	mult, ok := effects.ElementResist[element]
	if !ok {
		return damage
	}
	reduced := damage * mult / 100
	if reduced < MinDamage {
		reduced = MinDamage
	}
	return reduced
}

// applyAttackDamage はダメージ適用・ログ出力・スキル成長・死亡処理を一括で行う共通関数。
// ShootActivityからも使用される
func applyAttackDamage(actor, target ecs.Entity, world w.World, attack gc.Attacker, attackMethodName string, hitRateModifier int, damageModifier int) {
	hit, criticalHit := rollHitCheckWithModifier(actor, target, world, attack, hitRateModifier)
	if !hit {
		logAttackResult(actor, target, world, false, false, 0, attackMethodName)
		worldhelper.SpawnVisualEffect(target, gc.NewMissEffect(), world)
		return
	}

	damage := calculateDamage(actor, target, world, attack, criticalHit, damageModifier)
	if damage < 0 {
		damage = 0
	}

	pools := world.Components.Pools.Get(target).(*gc.Pools)
	beforeHP := pools.HP.Current
	pools.HP.Current -= damage
	if pools.HP.Current < 0 {
		pools.HP.Current = 0
	}

	logAttackResult(actor, target, world, true, criticalHit, damage, attackMethodName)
	growWeaponSkill(actor, world, attack)
	worldhelper.SpawnVisualEffect(target, gc.NewDamageEffect(damage), world)

	if pools.HP.Current <= 0 && beforeHP > 0 {
		// 死亡時はすべてのアクティビティをキャンセルする
		if worldhelper.GetActivity(world, target) != nil {
			CancelActivity(target, "死亡", world)
		}
		target.AddComponent(world.Components.Dead, &gc.Dead{})
		logDeath(world, target)
	} else if comp := worldhelper.GetActivity(world, target); comp != nil && CanInterrupt(comp) {
		// 被ダメージで中断可能なアクティビティをキャンセルする
		CancelActivity(target, "攻撃を受けた", world)
	}
}

// calculateHitRate は命中率を算出する。ダイスロールなしの純粋な計算で、UI表示と命中判定の両方で使用する
func calculateHitRate(attacker, target ecs.Entity, world w.World, attack gc.Attacker, modifier int) int {
	attackerAbils := world.Components.Abilities.Get(attacker).(*gc.Abilities)
	targetAbils := world.Components.Abilities.Get(target).(*gc.Abilities)

	hitRate := BaseHitRate + (attackerAbils.Dexterity.Total-targetAbils.Agility.Total)*HitRatePerStatPoint
	hitRate += getWeaponAccuracyFromAttack(attack)
	hitRate = hitRate * getSkillMult(attacker, attack, world, false) / 100
	hitRate += modifier

	if hitRate > MaxHitRate {
		hitRate = MaxHitRate
	}
	if hitRate < MinHitRate {
		hitRate = MinHitRate
	}

	return hitRate
}

// rollHitCheckWithModifier は命中判定を行う。modifierは追加の命中率補正（負値でペナルティ）
func rollHitCheckWithModifier(attacker, target ecs.Entity, world w.World, attack gc.Attacker, modifier int) (hit bool, critical bool) {
	hitRate := calculateHitRate(attacker, target, world, attack, modifier)

	roll := world.Config.RNG.IntN(DiceMax) + 1
	hit = roll <= hitRate
	critical = roll <= CriticalHitThreshold

	return hit, critical
}

// getWeaponAccuracyFromAttack はAttackerから命中率補正を取得する
func getWeaponAccuracyFromAttack(attack gc.Attacker) int {
	if attack == nil {
		return 0
	}
	return attack.GetAccuracy() - BaseHitRate
}

// calculateDamage はダメージ計算を行う
func calculateDamage(attacker, target ecs.Entity, world w.World, attack gc.Attacker, critical bool, damageModifier int) int {
	attackerAbils := world.Components.Abilities.Get(attacker).(*gc.Abilities)

	baseAbil := attackerAbils.Strength.Total
	if attack != nil && attack.GetAttackCategory().Range == gc.AttackRangeRanged {
		baseAbil = attackerAbils.Sensation.Total
	}

	targetAbils := world.Components.Abilities.Get(target).(*gc.Abilities)
	targetDefense := targetAbils.Defense.Total

	baseDamage := baseAbil + world.Config.RNG.IntN(DamageRandomRange) + 1

	if attack != nil {
		baseDamage += attack.GetDamage()
	}
	baseDamage += damageModifier

	baseDamage = baseDamage * getSkillMult(attacker, attack, world, true) / 100

	if critical {
		baseDamage = baseDamage * CriticalDamageMultiplier / CriticalDamageBase
	}

	if attack != nil && attack.GetElement() != gc.ElementTypeNone {
		baseDamage = applyElementResist(baseDamage, target, attack.GetElement(), world)
	}

	finalDamage := baseDamage - targetDefense
	if finalDamage < MinDamage {
		finalDamage = MinDamage
	}

	return finalDamage
}

// growWeaponSkill は攻撃成功時に武器スキルの経験値を加算する
func growWeaponSkill(actor ecs.Entity, world w.World, attack gc.Attacker) {
	if attack == nil {
		return
	}
	skillsComp := world.Components.Skills.Get(actor)
	if skillsComp == nil {
		return
	}
	skills := skillsComp.(*gc.Skills)

	skillID, ok := gc.WeaponSkillID(attack.GetAttackCategory())
	if !ok {
		return
	}
	s := skills.Get(skillID)

	abilsComp := world.Components.Abilities.Get(actor)
	if abilsComp == nil {
		return
	}
	abils := abilsComp.(*gc.Abilities)
	ablID := gc.SkillAbilityID(skillID)

	if skill.GainExp(s, abils.ValueOf(ablID)) {
		actor.AddComponent(world.Components.StatsChanged, &gc.StatsChanged{})

		actorName := worldhelper.GetEntityName(actor, world)
		gamelog.New(gamelog.FieldLog).
			Append(fmt.Sprintf("%s のスキルが上がった！（%s Lv%d）", actorName, string(skillID), s.Value)).
			Log()
	}
}

// logAttackResult は攻撃結果をログに出力する
func logAttackResult(attacker, target ecs.Entity, world w.World, hit bool, critical bool, damage int, attackMethodName string) {
	if !attacker.HasComponent(world.Components.Player) && !target.HasComponent(world.Components.Player) {
		return
	}

	attackerName := worldhelper.GetEntityName(attacker, world)
	targetName := worldhelper.GetEntityName(target, world)

	gamelog.New(gamelog.FieldLog).
		Build(func(l *gamelog.Logger) {
			worldhelper.AppendNameWithColor(l, attacker, attackerName, world)
		}).
		Append(" は ").
		Build(func(l *gamelog.Logger) {
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
func logDeath(world w.World, target ecs.Entity) {
	targetName := worldhelper.GetEntityName(target, world)

	gamelog.New(gamelog.FieldLog).
		Build(func(l *gamelog.Logger) {
			worldhelper.AppendNameWithColor(l, target, targetName, world)
		}).
		Append(" は倒れた。").
		Log()
}
