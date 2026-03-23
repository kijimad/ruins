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

	// 攻撃方法を取得
	attack, attackMethodName, err := aa.getAttackParams(actor, world)
	if err != nil {
		return fmt.Errorf("攻撃パラメータの取得に失敗: %w", err)
	}

	hit, criticalHit := aa.rollHitCheck(actor, target, world, attack)
	if !hit {
		aa.logAttackResult(actor, target, world, false, false, 0, attackMethodName)
		worldhelper.SpawnVisualEffect(target, gc.NewMissEffect(), world)
		return nil
	}

	damage := aa.calculateDamage(actor, target, world, attack, criticalHit)
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
	aa.logAttackResult(actor, target, world, true, criticalHit, damage, attackMethodName)

	// 攻撃成功時にスキル成長
	aa.growWeaponSkill(actor, world, attack)

	// ダメージエフェクトを生成
	worldhelper.SpawnVisualEffect(target, gc.NewDamageEffect(damage), world)

	// 死亡チェックと死亡ログ
	if pools.HP.Current <= 0 && beforeHP > 0 {
		target.AddComponent(world.Components.Dead, &gc.Dead{})
		aa.logDeath(world, target)
	}

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

func (aa *AttackActivity) rollHitCheck(attacker, target ecs.Entity, world w.World, attack *gc.Attack) (hit bool, critical bool) {
	attackerAbils := world.Components.Abilities.Get(attacker).(*gc.Abilities)
	attackerDexterity := attackerAbils.Dexterity.Total

	targetAbils := world.Components.Abilities.Get(target).(*gc.Abilities)
	targetAgility := targetAbils.Agility.Total

	baseHitRate := BaseHitRate + (attackerDexterity-targetAgility)*HitRatePerStatPoint

	weaponAccuracy := aa.getWeaponAccuracy(attacker, world)
	baseHitRate += weaponAccuracy

	// 武器スキルによる命中倍率を適用
	baseHitRate = baseHitRate * getSkillMult(attacker, attack, world, false) / 100

	if baseHitRate > MaxHitRate {
		baseHitRate = MaxHitRate
	}
	if baseHitRate < MinHitRate {
		baseHitRate = MinHitRate
	}

	roll := world.Config.RNG.IntN(DiceMax) + 1
	hit = roll <= baseHitRate
	critical = roll <= CriticalHitThreshold

	return hit, critical
}

func (aa *AttackActivity) calculateDamage(attacker, target ecs.Entity, world w.World, attack *gc.Attack, critical bool) int {
	attackerAbils := world.Components.Abilities.Get(attacker).(*gc.Abilities)

	// 武器の射程に応じて基礎能力値を切り替える
	baseAbil := attackerAbils.Strength.Total
	if attack != nil && attack.AttackCategory.Range == gc.AttackRangeRanged {
		baseAbil = attackerAbils.Sensation.Total
	}

	targetAbils := world.Components.Abilities.Get(target).(*gc.Abilities)
	targetDefense := targetAbils.Defense.Total

	baseDamage := baseAbil + world.Config.RNG.IntN(DamageRandomRange) + 1

	weaponDamage := aa.getWeaponDamage(attacker, world)
	baseDamage += weaponDamage

	// 武器スキルによるダメージ倍率を適用
	baseDamage = baseDamage * getSkillMult(attacker, attack, world, true) / 100

	if critical {
		baseDamage = baseDamage * CriticalDamageMultiplier / CriticalDamageBase
	}

	// 元素耐性による軽減
	if attack != nil && attack.Element != gc.ElementTypeNone {
		baseDamage = applyElementResist(baseDamage, target, attack.Element, world)
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
	rawMaster := world.Resources.RawMaster
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
			attack, weaponName, err := worldhelper.GetAttackFromWeapon(world, *weapon)
			if err == nil && attack != nil {
				return attack, weaponName, nil
			}
		}

		// 武器が装備されていない場合は素手武器を使用
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
	if !attacker.HasComponent(world.Components.Player) && !target.HasComponent(world.Components.Player) {
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

// getSkillMult は事前計算済みのスキル倍率(%)を返す。
// isDamageがtrueならWeaponDamage、falseならWeaponAccuracyを参照する。
// Effectsコンポーネントを持たないエンティティでは100(等倍)を返す。
func getSkillMult(entity ecs.Entity, attack *gc.Attack, world w.World, isDamage bool) int {
	if attack == nil {
		return 100
	}
	if !entity.HasComponent(world.Components.CharModifiers) {
		return 100
	}
	effects := world.Components.CharModifiers.Get(entity).(*gc.CharModifiers)
	skillID, ok := gc.WeaponSkillID(attack.AttackCategory)
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

// growWeaponSkill は攻撃成功時に武器スキルの経験値を加算する。
// Skillsコンポーネントを持たないエンティティではスキップする。
func (aa *AttackActivity) growWeaponSkill(actor ecs.Entity, world w.World, attack *gc.Attack) {
	if attack == nil {
		return
	}
	skillsComp := world.Components.Skills.Get(actor)
	if skillsComp == nil {
		return
	}
	skills := skillsComp.(*gc.Skills)

	skillID, ok := gc.WeaponSkillID(attack.AttackCategory)
	if !ok {
		return
	}
	s := skills.Get(skillID)

	// 対応する能力値を取得して成長速度に反映する
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
