package activity

import (
	"fmt"
	"strconv"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/skill"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 攻撃システムの定数
const (
	MeleeAttackRange = 1.5 // 近接攻撃の最大射程（斜めも考慮）
)

// MeleeBehavior はBehaviorの実装
type MeleeBehavior struct{}

// Info はBehaviorの実装
func (ab *MeleeBehavior) Info() Info {
	return Info{
		Name:            "Attack",
		Description:     "Attack an enemy",
		Interruptible:   false,
		Resumable:       false,
		ActionPointCost: consts.StandardActionCost,
		TotalRequiredAP: 0,
	}
}

// Name はBehaviorの実装
func (ab *MeleeBehavior) Name() gc.BehaviorName {
	return gc.BehaviorMelee
}

// NewMeleeActivity は攻撃対象を指定して攻撃アクティビティを組む。
func NewMeleeActivity(target ecs.Entity) *gc.Activity {
	comp := NewActivity(gc.BehaviorMelee, 0)
	comp.Params = &gc.MeleeParams{Target: target}
	return comp
}

// Validate はBehaviorの実装
func (ab *MeleeBehavior) Validate(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.MeleeParams)
	if !ok {
		return ErrParamsTypeMismatch
	}

	if world.Components.Dead.Has(actor) {
		return ErrAttackerDead
	}

	// 近接は隣接歩き込み専用で、対象選択が生存・隣接・攻撃能力を保証する。以降で弾かれるのは
	// 選択後の消失など通常プレイで起きない不変条件違反なのでシステムエラーとする。
	// ゼロ値・死亡エンティティは Ark の Has でパニックするため先に弾く
	if !world.ECS.Alive(p.Target) {
		return fmt.Errorf("target does not exist")
	}

	if !world.Components.GridElement.Has(p.Target) {
		return fmt.Errorf("target has no position")
	}

	if world.Components.Dead.Has(p.Target) {
		return fmt.Errorf("target is already dead")
	}

	if !ab.isInRange(actor, p.Target, world) {
		return fmt.Errorf("target is out of melee range")
	}

	if !ab.canPerformAttack(actor, world) {
		return fmt.Errorf("attacker has no attack means")
	}

	return nil
}

// Start はBehaviorの実装
func (ab *MeleeBehavior) Start(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	if p, ok := comp.Params.(*gc.MeleeParams); ok {
		log.Debug("attack started", "actor", actor, "target", p.Target)
	}
	return nil
}

// DoTurn はBehaviorの実装
func (ab *MeleeBehavior) DoTurn(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	if _, ok := comp.Params.(*gc.MeleeParams); !ok {
		Cancel(comp, "attack target is not set")
		return ErrParamsTypeMismatch
	}

	if !ab.canAttack(comp, actor, world) {
		Cancel(comp, "cannot attack")
		return ErrAttackTargetInvalid
	}

	if err := ab.performAttack(comp, actor, world); err != nil {
		Cancel(comp, fmt.Sprintf("attack error: %s", err.Error()))
		return err
	}

	Complete(comp)
	return nil
}

// Finish はBehaviorの実装
func (ab *MeleeBehavior) Finish(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	p, ok := comp.Params.(*gc.MeleeParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	log.Debug("attack activity finished",
		"actor", actor,
		"target", p.Target)

	return nil
}

// Canceled はBehaviorの実装
func (ab *MeleeBehavior) Canceled(comp *gc.Activity, actor ecs.Entity, _ w.World) error {
	log.Debug("attack canceled", "actor", actor, "reason", comp.CancelReason)
	return nil
}

func (ab *MeleeBehavior) performAttack(comp *gc.Activity, actor ecs.Entity, world w.World) error {
	p, ok := comp.Params.(*gc.MeleeParams)
	if !ok {
		return ErrParamsTypeMismatch
	}
	target := p.Target

	log.Debug("performing attack", "attacker", actor, "target", target)

	attack, attackMethodName, err := getAttackParams(actor, world)
	if err != nil {
		return fmt.Errorf("failed to get attack parameters: %w", err)
	}

	return applyAttackDamage(actor, target, world, attack, attackMethodName, 0, 0)
}

func (ab *MeleeBehavior) canAttack(comp *gc.Activity, actor ecs.Entity, world w.World) bool {
	if _, ok := comp.Params.(*gc.MeleeParams); !ok {
		return false
	}

	if err := ab.Validate(comp, actor, world); err != nil {
		return false
	}

	return true
}

func (ab *MeleeBehavior) isInRange(attacker, target ecs.Entity, world w.World) bool {
	if !world.Components.GridElement.Has(attacker) {
		return false
	}
	attackerGrid := world.Components.GridElement.Get(attacker)

	if !world.Components.GridElement.Has(target) {
		return false
	}
	targetGrid := world.Components.GridElement.Get(target)

	distance := geometry.Distance(float64(attackerGrid.X), float64(attackerGrid.Y), float64(targetGrid.X), float64(targetGrid.Y))

	// TODO: 遠距離武器の場合は射程を武器から取得
	return distance <= MeleeAttackRange
}

func (ab *MeleeBehavior) canPerformAttack(attacker ecs.Entity, world w.World) bool {
	// TODO: 装備武器のチェック
	abils := world.Components.Abilities.Get(attacker)
	return abils != nil
}

// getBareHandsAttack は素手武器の攻撃パラメータを取得する
func getBareHandsAttack(world w.World) (gc.Attacker, string, error) {
	bareHandsSpec, err := raw.NewWeaponSpec(world.Resources.RawMaster, "bare_hands")
	if err != nil {
		return nil, "", fmt.Errorf("bare hands weapon not found: %w", err)
	}
	if bareHandsSpec.Melee == nil {
		return nil, "", fmt.Errorf("bare hands weapon has no Melee component")
	}
	if bareHandsSpec.Name == nil {
		return nil, "", fmt.Errorf("bare hands weapon has no Name component")
	}
	// 攻撃方法名は素手武器の表示名。武器と同じく raw の英語 name を返し、表示側の logAttackResult が
	// query.T で訳す。表示 name なので raw の翻訳カバレッジゲートが訳の有無を検証する。
	return bareHandsSpec.Melee, bareHandsSpec.Name.Name, nil
}

// getAttackParams は攻撃者の武器から攻撃パラメータと攻撃方法名を取得する
// 戻り値: (攻撃パラメータ, 攻撃方法名, エラー)
func getAttackParams(attacker ecs.Entity, world w.World) (gc.Attacker, string, error) {
	// プレイヤーの場合: 装備武器から攻撃パラメータを取得
	if world.Components.Player.Has(attacker) {
		// 選択中の武器スロット番号（1-5）から配列インデックスに変換
		selectedSlot := query.GetWeaponSelection(world).Slot
		weaponIndex := selectedSlot - 1 // 1-based to 0-based
		if weaponIndex < 0 || weaponIndex >= 5 {
			return nil, "", fmt.Errorf("invalid weapon slot number: %d", selectedSlot)
		}

		weapons := query.GetWeapons(world, attacker)
		weapon := weapons[weaponIndex]
		if weapon != nil {
			// 装備している武器から攻撃パラメータを取得
			attack, weaponName, err := query.GetMeleeFromWeapon(world, *weapon)
			if err == nil && attack != nil {
				return attack, weaponName, nil
			}
		}

		// 武器が装備されていない場合は素手武器を使用
		return getBareHandsAttack(world)
	}

	// 敵の場合: CommandTableから攻撃パラメータを取得
	if world.Components.CommandTable.Has(attacker) {
		attack, weaponName, err := query.GetAttackFromCommandTable(world, attacker)
		if err == nil && attack != nil {
			return attack, weaponName, nil
		}

		// CommandTableから取得できない場合は素手武器を使用
		return getBareHandsAttack(world)
	}

	return nil, "", fmt.Errorf("cannot get attack parameters: attacker has neither Player nor CommandTable component")
}

// getSkillMult は事前計算済みのスキル倍率(%)を返す。
// isDamageがtrueならWeaponDamage、falseならWeaponAccuracyを参照する。
// Effectsコンポーネントを持たないエンティティでは100(等倍)を返す。
func getSkillMult(entity ecs.Entity, attack gc.Attacker, world w.World, isDamage bool) consts.Percent {
	if attack == nil {
		return consts.PercentBase
	}
	if !world.Components.CharModifiers.Has(entity) {
		return consts.PercentBase
	}
	effects := world.Components.CharModifiers.Get(entity)
	skillID, ok := gc.WeaponSkillID(attack.GetAttackCategory())
	if !ok {
		return consts.PercentBase
	}
	var mults map[gc.SkillID]consts.Percent
	if isDamage {
		mults = effects.WeaponDamage
	} else {
		mults = effects.WeaponAccuracy
	}
	if mult, ok := mults[skillID]; ok {
		return mult
	}
	return consts.PercentBase
}

// applyElementResist は事前計算済みの元素耐性倍率でダメージを軽減する
func applyElementResist(damage int, target ecs.Entity, element gc.ElementType, world w.World) int {
	if !world.Components.CharModifiers.Has(target) {
		return damage
	}
	effects := world.Components.CharModifiers.Get(target)
	mult, ok := effects.ElementResist[element]
	if !ok {
		return damage
	}
	reduced := max(mult.ApplyInt(damage), formula.MinDamage)
	return reduced
}

// applyAttackDamage はダメージ適用・ログ出力・スキル成長・死亡処理を一括で行う共通関数。
// ShootBehaviorからも使用される
func applyAttackDamage(actor, target ecs.Entity, world w.World, attack gc.Attacker, attackMethodName string, hitRateModifier int, damageModifier int) error {
	if attack == nil {
		return fmt.Errorf("attack must not be nil")
	}

	hit, criticalHit := rollHitCheckWithModifier(actor, target, world, attack, hitRateModifier)
	if !hit {
		logAttackResult(actor, target, world, false, false, 0, attackMethodName)
		lifecycle.SpawnVisualEffect(target, gc.NewMissEffect(), world)
		return nil
	}

	damage := max(calculateDamage(actor, target, world, attack, criticalHit, damageModifier), 0)

	logAttackResult(actor, target, world, true, criticalHit, damage, attackMethodName)
	growWeaponSkill(actor, world, attack)
	lifecycle.SpawnVisualEffect(target, gc.NewDamageEffect(damage), world)
	gameaction.ApplyDamage(world, target, damage, actor)
	// HP を削った後、HealthStatus を持つ対象へ確率で怪我を付ける
	applyInjury(actor, target, world, attack)

	// 被ダメージで中断可能なアクティビティをキャンセルする
	if comp := query.GetActivity(world, target); comp != nil && CanInterrupt(comp) {
		CancelActivity(target, "took an attack", world)
	}

	return nil
}

// calculateHitRate は命中率を算出する。ダイスロールなしの純粋な計算で、UI表示と命中判定の両方で使用する
func calculateHitRate(attacker, target ecs.Entity, world w.World, attack gc.Attacker, modifier int) int {
	if !world.Components.Abilities.Has(attacker) {
		return formula.BaseHitRate
	}
	attackerAbils := world.Components.Abilities.Get(attacker)

	// Abilitiesを持たないターゲットには自動命中する
	targetAgility := 0
	if world.Components.Abilities.Has(target) {
		targetAbilsComp := world.Components.Abilities.Get(target)
		targetAgility = targetAbilsComp.Agility.Total
	}

	// 基礎命中は反射神経 DEX と対象の回避の対抗。武器習熟の能力値は WeaponAccuracy 側が畳むのでここには足さない
	hitRate := formula.BaseHitRate + (attackerAbils.Dexterity.Total-targetAgility)*formula.HitRatePerStatPoint
	hitRate += getWeaponAccuracyFromAttack(attack)
	hitRate = getSkillMult(attacker, attack, world, false).ApplyInt(hitRate)
	hitRate += modifier

	// 体調由来の命中低下を掛ける。CharModifiers を持たない攻撃者は身体機能ペナルティを受けず等倍
	if world.Components.CharModifiers.Has(attacker) {
		aim := world.Components.CharModifiers.Get(attacker).AccuracyCapacity(attack.GetAttackCategory())
		hitRate = aim.ApplyInt(hitRate)
	}

	return formula.ClampHitRate(hitRate)
}

// rollHitCheckWithModifier は命中判定を行う。modifierは追加の命中率補正（負値でペナルティ）
func rollHitCheckWithModifier(attacker, target ecs.Entity, world w.World, attack gc.Attacker, modifier int) (hit bool, critical bool) {
	hitRate := calculateHitRate(attacker, target, world, attack, modifier)

	roll := world.Resources.Config.RNG.IntN(formula.DiceMax) + 1
	hit = roll <= hitRate
	critical = roll <= formula.CriticalHitThreshold

	return hit, critical
}

// getWeaponAccuracyFromAttack はAttackerから命中率補正を取得する
func getWeaponAccuracyFromAttack(attack gc.Attacker) int {
	return attack.GetAccuracy() - formula.BaseHitRate
}

// calculateDamage はダメージ計算を行う
func calculateDamage(attacker, target ecs.Entity, world w.World, attack gc.Attacker, critical bool, damageModifier int) int {
	if !world.Components.Abilities.Has(attacker) {
		return 0
	}
	attackerAbils := world.Components.Abilities.Get(attacker)

	baseAbil := attackerAbils.Strength.Total
	if attack.GetAttackCategory().Range == gc.AttackRangeRanged {
		baseAbil = attackerAbils.Sensation.Total
	}

	targetDefense := 0
	if world.Components.Abilities.Has(target) {
		targetAbilsComp := world.Components.Abilities.Get(target)
		targetDefense = targetAbilsComp.Defense.Total
	}

	baseDamage := baseAbil + world.Resources.Config.RNG.IntN(formula.DamageRandomRange) + 1
	baseDamage += attack.GetDamage()
	baseDamage += damageModifier

	baseDamage = getSkillMult(attacker, attack, world, true).ApplyInt(baseDamage)

	if critical {
		baseDamage = formula.ApplyCritical(baseDamage)
	}

	if attack.GetElement() != gc.ElementTypeNone {
		baseDamage = applyElementResist(baseDamage, target, attack.GetElement(), world)
	}

	finalDamage := max(baseDamage-targetDefense, formula.MinDamage)

	return finalDamage
}

// growWeaponSkill は攻撃成功時に武器スキルの経験値を加算する
func growWeaponSkill(actor ecs.Entity, world w.World, attack gc.Attacker) {
	if !world.Components.Skills.Has(actor) {
		return
	}
	skills := world.Components.Skills.Get(actor)

	skillID, ok := gc.WeaponSkillID(attack.GetAttackCategory())
	if !ok {
		return
	}
	s := skills.Get(skillID)

	if !world.Components.Abilities.Has(actor) {
		return
	}
	abils := world.Components.Abilities.Get(actor)
	ablID := gc.SkillAbilityID(skillID)

	if skill.GainExp(s, abils.ValueOf(ablID)) {
		// 同一ターン内の別処理が既にマーカーを付けていることがあるため、二重付与を避ける
		if !world.Components.StatsChanged.Has(actor) {
			world.Components.StatsChanged.Add(actor, &gc.StatsChanged{})
		}

		actorName := query.GetEntityName(actor, world)
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s's skill rose! (%s Lv%d)", actorName, string(skillID), s.Value)).
			Log()
	}
}

// logAttackResult は攻撃結果をログに出力する
func logAttackResult(attacker, target ecs.Entity, world w.World, hit bool, critical bool, damage int, attackMethodName string) {
	attackerRelevant := query.IsAlly(world, attacker)
	targetRelevant := query.IsAlly(world, target)
	if !attackerRelevant && !targetRelevant {
		return
	}

	attackerName := query.GetEntityName(attacker, world)
	targetName := query.GetEntityName(target, world)
	attackerMarkup := query.NameMarkup(attacker, attackerName, world)
	targetMarkup := query.NameMarkup(target, targetName, world)
	damageStr := strconv.Itoa(damage)

	logger := gamelog.New(query.GetGameLog(world))
	withMethod := attackMethodName != ""
	// 攻撃方法名は武器名や素手の英語原文なので、表示前に現在言語へ訳す。
	methodName := attackMethodName
	if withMethod {
		methodName = query.T(world, attackMethodName)
	}
	switch {
	case !hit:
		if withMethod {
			logger.Markup(query.T(world, "%s used %s to attack %s but missed.", attackerMarkup, methodName, targetMarkup))
		} else {
			logger.Markup(query.T(world, "%s attacked %s but missed.", attackerMarkup, targetMarkup))
		}
	case critical:
		if withMethod {
			logger.Markup(query.T(world, "%s used %s to score a critical hit on %s and dealt %s damage!", attackerMarkup, methodName, targetMarkup, damageStr))
		} else {
			logger.Markup(query.T(world, "%s scored a critical hit on %s and dealt %s damage!", attackerMarkup, targetMarkup, damageStr))
		}
	default:
		if withMethod {
			logger.Markup(query.T(world, "%s used %s to attack %s and dealt %s damage.", attackerMarkup, methodName, targetMarkup, damageStr))
		} else {
			logger.Markup(query.T(world, "%s attacked %s and dealt %s damage.", attackerMarkup, targetMarkup, damageStr))
		}
	}
	logger.Log()
}
