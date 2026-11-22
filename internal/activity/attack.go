package activity

import (
	"fmt"
	"strconv"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/formula"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
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

	// misuseChargeMultiplier は本来の用途以外に使ったときに残量が減る倍率
	misuseChargeMultiplier = 2
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
		return ErrParamsTypeMismatch
	}

	if !ab.canAttack(comp, actor, world) {
		return ErrAttackTargetInvalid
	}

	if err := ab.performAttack(comp, actor, world); err != nil {
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

	attack, attackMethodName, weapon, err := getAttackParams(actor, world)
	if err != nil {
		return fmt.Errorf("failed to get attack parameters: %w", err)
	}

	return applyAttackDamage(actor, target, world, weapon, attack, attackMethodName, 0, 0)
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
func getBareHandsAttack(world w.World) (gc.Attacker, i18n.Msgid, error) {
	melee, name, err := raw.BodyAttackMelee(world.Resources.RawMaster, raw.BareHands)
	if err != nil {
		return nil, "", fmt.Errorf("bare hands attack not found: %w", err)
	}
	return melee, name, nil
}

// getAttackParams は攻撃者の武器から攻撃パラメータと攻撃方法名と武器の実体を取得する。
// 素手や敵の攻撃表の攻撃は武器の実体を持たないので gc.InvalidEntity を返す
func getAttackParams(attacker ecs.Entity, world w.World) (gc.Attacker, i18n.Msgid, ecs.Entity, error) {
	if world.Components.Player.Has(attacker) {
		weapon, ok := query.SelectedHeld(world, attacker)
		if !ok || !world.Components.Melee.Has(weapon) {
			attack, name, err := getBareHandsAttack(world)
			return attack, name, gc.InvalidEntity, err
		}
		attack, name, err := query.GetMeleeFromWeapon(world, weapon)
		return attack, name, weapon, err
	}

	if world.Components.CommandTable.Has(attacker) {
		attack, name, err := query.GetAttackFromCommandTable(world, attacker)
		return attack, name, gc.InvalidEntity, err
	}

	return nil, "", gc.InvalidEntity, ErrNoAttackSource
}

// getSkillMult はスキル倍率(%)を返す。
// isDamageがtrueならダメージ倍率、falseなら命中倍率を参照する
func getSkillMult(entity ecs.Entity, attack gc.Attacker, world w.World, isDamage bool) consts.Percent {
	if attack == nil {
		return consts.PercentBase
	}
	weapon, ok := world.Resources.SkillDefs.Weapon(attack.GetAttackCategory())
	if !ok {
		return consts.PercentBase
	}
	key := weapon.Damage
	if !isDamage {
		key = weapon.Accuracy
	}
	return query.ProficiencyValue(world, entity, key)
}

// elementMult は対象の属性 element への耐性の倍率を返す。耐性を持たない属性なら等倍
func elementMult(target ecs.Entity, element rawschema.Element, world w.World) consts.Percent {
	key, ok := world.Resources.SkillDefs.Resist(element)
	if !ok {
		return consts.PercentBase
	}
	return query.ProficiencyValue(world, target, key)
}

// applyAttackDamage は命中を判定し、当たれば applyHit で与える。外れたらログと見た目の効果を出す。
// 近接・射撃・投げるが使う。weapon は攻撃に使った武器の実体で、残量を持てば当たっても外れても最後に減らす。
// 武器の実体を持たない攻撃は gc.InvalidEntity を渡す
func applyAttackDamage(actor, target ecs.Entity, world w.World, weapon ecs.Entity, attack gc.Attacker, attackMethodName i18n.Msgid, hitRateModifier int, damageModifier int) error {
	if attack == nil {
		return fmt.Errorf("attack must not be nil")
	}
	// 減らす量は殴る前の対象で決め、武器は最後に減らす。先に消すと、壊したときの産出が武器の道具の値を読めない
	if query.ChargeTriggerOf(world, weapon) == query.ChargeTriggerOnUse {
		defer gameaction.ConsumeCharge(world, weapon, attackChargeCost(world, weapon, target))
	}

	hit, criticalHit := rollHitCheckWithModifier(actor, target, world, attack, hitRateModifier)
	if !hit {
		logAttackResult(actor, target, world, false, false, 0, attackMethodName)
		lifecycle.SpawnVisualEffect(target, gc.NewMissEffect(), world)
		return nil
	}
	_, err := applyHit(actor, target, world, attack, attackMethodName, hitOptions{
		critical:  criticalHit,
		injury:    true,
		growSkill: true,
		weapon:    weapon,
		damageMod: damageModifier,
	})
	return err
}

// hitOptions は命中したあとの処理のうち、呼ぶ側が決める部分
type hitOptions struct {
	critical  bool       // 会心
	injury    bool       // 怪我を抽選する
	growSkill bool       // 傷つけたら武器スキルを伸ばす
	weapon    ecs.Entity // 攻撃に使った物。置物への道具の倍率・廃材の抽選・道具スキルの伸びを決める。持たない攻撃は gc.InvalidEntity
	damageMod int        // ダメージに足す値。射撃の装填のボーナスなど
}

// applyHit は命中した攻撃を target に与え、与えたダメージを返す。ログ・見た目の効果・ダメージ・置物の廃材・
// 被弾した者の行動の中断をまとめて行う。命中の判定をしない爆風もこれを使う
func applyHit(actor, target ecs.Entity, world w.World, attack gc.Attacker, attackMethodName i18n.Msgid, opts hitOptions) (int, error) {
	mult := elementMult(target, attack.GetElement(), world)
	// 道具の倍率は最低保証の後に掛け、道具が必須なのに無いときは確実に0にする
	damage := max(calculateDamage(actor, target, world, attack, opts.critical, opts.damageMod, mult), 0) *
		toolDamageMultiplier(world, opts.weapon, target)

	logAttackResult(actor, target, world, true, opts.critical, damage, attackMethodName)
	logElementAffinity(actor, target, world, attack.GetElement(), mult)
	// 削れない相手を殴り続けて安全に稼げないよう、ダメージ0では武器スキルを伸ばさない
	if opts.growSkill && damage > 0 {
		growWeaponSkill(actor, world, attack)
	}
	lifecycle.SpawnVisualEffect(target, gc.NewDamageEffect(damage, attack.GetElement(), mult), world)
	gameaction.ApplyDamage(world, target, damage, actor)
	if world.Components.Prop.Has(target) && world.Components.Dead.Has(target) {
		if err := dropSalvage(world, actor, opts.weapon, target); err != nil {
			return damage, err
		}
	}
	if opts.injury {
		applyInjury(actor, target, world, attack)
	}

	// 被ダメージで中断可能なアクティビティをキャンセルする
	if comp := query.GetActivity(world, target); comp != nil && CanInterrupt(comp) {
		CancelActivity(target, "took an attack", world)
	}

	return damage, nil
}

// attackChargeCost は weapon で target を攻撃したときに減る残量を返す。置物は分類の合う道具で殴れば1、
// ほかは本来の用途以外として倍にする。敵などへの攻撃は、道具を兼ねる物でも1
func attackChargeCost(world w.World, weapon, target ecs.Entity) int {
	if !world.Components.Prop.Has(target) {
		return 1
	}
	if def, ok := raw.FindDisassembly(world.Resources.RawMaster, query.GetEntityID(target, world)); ok {
		if _, fit := toolGrade(world, weapon, def); fit {
			return 1
		}
	}
	return misuseChargeMultiplier
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

	// 基礎命中は反射神経 DEX と対象の回避の対抗。武器習熟の能力値と体調由来の
	// 命中低下は WeaponAccuracy 側が畳むのでここには足さない
	hitRate := formula.BaseHitRate + (attackerAbils.Dexterity.Total-targetAgility)*formula.HitRatePerStatPoint
	hitRate += getWeaponAccuracyFromAttack(attack)
	hitRate = getSkillMult(attacker, attack, world, false).ApplyInt(hitRate)
	hitRate += modifier

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

// calculateDamage は ECS から攻撃側と対象の値を集め、formula.Damage で1回の命中のダメージを求める。
// 属性の倍率 elemMult は、表示にも使うので呼ぶ側が1度だけ求めて渡す
func calculateDamage(attacker, target ecs.Entity, world w.World, attack gc.Attacker, critical bool, damageModifier int, elemMult consts.Percent) int {
	if !world.Components.Abilities.Has(attacker) {
		return 0
	}
	attackerAbils := world.Components.Abilities.Get(attacker)

	ability := attackerAbils.Strength.Total
	if _, ranged := world.Resources.SkillDefs.RangeParams(attack.GetAttackCategory()); ranged {
		ability = attackerAbils.Sensation.Total
	}

	defense := 0
	if world.Components.Abilities.Has(target) {
		defense = world.Components.Abilities.Get(target).Defense.Total
	}

	return formula.Damage(formula.DamageInput{
		Ability:      ability,
		Die:          world.Resources.Config.RNG.IntN(formula.DamageRandomRange) + 1,
		WeaponDamage: attack.GetDamage(),
		Modifier:     damageModifier,
		SkillMult:    int(getSkillMult(attacker, attack, world, true)),
		Critical:     critical,
		ElementMult:  int(elemMult),
		Defense:      defense,
	})
}

// growWeaponSkill は攻撃成功時に武器スキルの経験値を加算する
func growWeaponSkill(actor ecs.Entity, world w.World, attack gc.Attacker) {
	// スキル成長は味方側の進行要素。敵も Skills を持つが戦闘中に強くならないよう除外する
	if query.IsEnemy(world, actor) {
		return
	}
	if !world.Components.Skills.Has(actor) {
		return
	}
	skills := world.Components.Skills.Get(actor)

	weapon, ok := world.Resources.SkillDefs.Weapon(attack.GetAttackCategory())
	if !ok {
		return
	}
	skillID := weapon.Skill
	s := skills.Get(skillID)

	if !world.Components.Abilities.Has(actor) {
		return
	}
	abils := world.Components.Abilities.Get(actor)
	ablID := world.Resources.SkillDefs.Ability(skillID)

	if skill.GainExp(s, abils.ValueOf(ablID)) {
		actorName := query.GetEntityName(actor, world)
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s's skill rose! (%s Lv%d)", actorName, skillID, s.Value)).
			Log()
	}
}

// logElementAffinity は属性の倍率で攻撃が減ったか増えたかをログに出す。味方が関わる攻撃だけを出す
func logElementAffinity(attacker, target ecs.Entity, world w.World, element rawschema.Element, mult consts.Percent) {
	if mult == consts.PercentBase || (!query.IsAlly(world, attacker) && !query.IsAlly(world, target)) {
		return
	}
	targetMarkup := query.NameMarkup(target, query.GetEntityName(target, world), world)
	elementName := query.T(world, gc.ElementName(element))
	text := query.T(world, "%s resisted the %s.", targetMarkup, elementName)
	if mult > consts.PercentBase {
		text = query.T(world, "%s is weak to %s.", targetMarkup, elementName)
	}
	gamelog.New(query.GetGameLog(world)).Markup(text).Log()
}

// logAttackResult は攻撃結果をログに出力する
func logAttackResult(attacker, target ecs.Entity, world w.World, hit bool, critical bool, damage int, attackMethodName i18n.Msgid) {
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
	var methodName string
	if withMethod {
		methodName = query.T(world, attackMethodName)
	}
	switch {
	case hit && attacker == target && withMethod:
		logger.Markup(query.T(world, "%s was caught in %s and took %s damage.", targetMarkup, methodName, damageStr))
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
