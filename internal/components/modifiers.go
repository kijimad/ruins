package components

import (
	"fmt"
	"slices"

	"github.com/kijimaD/ruins/internal/consts"
)

// ModifierKey は効果倍率の識別キー
type ModifierKey string

// 効果キー定数
const (
	ModFireResist     ModifierKey = "fire_resist"
	ModThunderResist  ModifierKey = "thunder_resist"
	ModChillResist    ModifierKey = "chill_resist"
	ModPhotonResist   ModifierKey = "photon_resist"
	ModColdProgress   ModifierKey = "cold_progress"
	ModHungerProgress ModifierKey = "hunger_progress"
	ModHealingEffect  ModifierKey = "healing_effect"
	ModMaxWeight      ModifierKey = "max_weight"
	ModExploration    ModifierKey = "exploration"
	ModEnemyVision    ModifierKey = "enemy_vision"
	ModNightVision    ModifierKey = "night_vision"
	ModMoveCost       ModifierKey = "move_cost"
	ModCraftCost      ModifierKey = "craft_cost"
	ModSmithQuality   ModifierKey = "smith_quality"
	ModBuyPrice       ModifierKey = "buy_price"
	ModSellPrice      ModifierKey = "sell_price"
	ModHeavyArmor     ModifierKey = "heavy_armor"
	// ModRecovery は自然回復・治癒にかかる速度倍率。基準100。VIT が能力ソース、空腹・疲労・睡眠が
	// 状態ソースとして寄与する。剣攻撃力などと同じく specByKey の単機能倍率で、スキルは持たない
	ModRecovery ModifierKey = "recovery"
	// ModActionSpeed は行動速度。基準100。AGI・DEX が能力ソース、疲労・空腹が状態ソースとして寄与する。
	// 剣攻撃力などと同じく specByKey の単機能倍率で、スキルは持たない。CalculateSpeed が値をそのまま速度に使う
	ModActionSpeed ModifierKey = "action_speed"

	ModSwordDamage   ModifierKey = "sword_damage"
	ModSpearDamage   ModifierKey = "spear_damage"
	ModFistDamage    ModifierKey = "fist_damage"
	ModBowDamage     ModifierKey = "bow_damage"
	ModHandgunDamage ModifierKey = "handgun_damage"
	ModRifleDamage   ModifierKey = "rifle_damage"
	ModCannonDamage  ModifierKey = "cannon_damage"

	ModSwordAccuracy   ModifierKey = "sword_accuracy"
	ModSpearAccuracy   ModifierKey = "spear_accuracy"
	ModFistAccuracy    ModifierKey = "fist_accuracy"
	ModBowAccuracy     ModifierKey = "bow_accuracy"
	ModHandgunAccuracy ModifierKey = "handgun_accuracy"
	ModRifleAccuracy   ModifierKey = "rifle_accuracy"
	ModCannonAccuracy  ModifierKey = "cannon_accuracy"
)

// weaponDamageKeys は武器スキルIDからダメージ効果キーへのマッピング
var weaponDamageKeys = map[SkillID]ModifierKey{
	SkillSword:   ModSwordDamage,
	SkillSpear:   ModSpearDamage,
	SkillFist:    ModFistDamage,
	SkillBow:     ModBowDamage,
	SkillHandgun: ModHandgunDamage,
	SkillRifle:   ModRifleDamage,
	SkillCannon:  ModCannonDamage,
}

// WeaponDamageKey は武器スキルIDに対応するダメージ効果キーを返す。未定義ならpanicする
func WeaponDamageKey(id SkillID) ModifierKey {
	key, ok := weaponDamageKeys[id]
	if !ok {
		panic(fmt.Sprintf("undefined weapon skill ID for damage: %q", id))
	}
	return key
}

// weaponAccuracyKeys は武器スキルIDから命中効果キーへのマッピング
var weaponAccuracyKeys = map[SkillID]ModifierKey{
	SkillSword:   ModSwordAccuracy,
	SkillSpear:   ModSpearAccuracy,
	SkillFist:    ModFistAccuracy,
	SkillBow:     ModBowAccuracy,
	SkillHandgun: ModHandgunAccuracy,
	SkillRifle:   ModRifleAccuracy,
	SkillCannon:  ModCannonAccuracy,
}

// WeaponAccuracyKey は武器スキルIDに対応する命中効果キーを返す。未定義ならpanicする
func WeaponAccuracyKey(id SkillID) ModifierKey {
	key, ok := weaponAccuracyKeys[id]
	if !ok {
		panic(fmt.Sprintf("undefined weapon skill ID for accuracy: %q", id))
	}
	return key
}

// elementResistKeys は元素タイプから耐性効果キーへのマッピング
var elementResistKeys = map[ElementType]ModifierKey{
	ElementTypeFire:    ModFireResist,
	ElementTypeThunder: ModThunderResist,
	ElementTypeChill:   ModChillResist,
	ElementTypePhoton:  ModPhotonResist,
}

// ElementResistKey は元素タイプに対応する耐性効果キーを返す。未定義ならpanicする
func ElementResistKey(elem ElementType) ModifierKey {
	key, ok := LookupElementResistKey(elem)
	if !ok {
		panic(fmt.Sprintf("undefined element type for resistance: %q", elem))
	}
	return key
}

// LookupElementResistKey は元素タイプに対応する耐性効果キーを返す。無属性など未定義は ok=false
func LookupElementResistKey(elem ElementType) (ModifierKey, bool) {
	key, ok := elementResistKeys[elem]
	return key, ok
}

// スキル効果係数の定数。スキル値1あたりの倍率変化量（%）を定義する。
// 正の値はスキルが高いほど効果が増し、負の値は効果が減る。
const (
	coeffWeaponDamage   = 5  // 武器ダメージ: スキルLv1あたり+5%
	coeffWeaponAccuracy = 3  // 武器命中: スキルLv1あたり+3%
	coeffElementResist  = -3 // 元素耐性: スキルLv1あたり-3%（被ダメージ軽減）
	coeffColdProgress   = -3 // 低体温進行: スキルLv1あたり-3%
	coeffHungerProgress = -2 // 空腹進行: スキルLv1あたり-2%
	coeffHealingEffect  = 5  // 回復効果: スキルLv1あたり+5%
	coeffMaxWeight      = 4  // 最大所持重量: スキルLv1あたり+4%
	coeffExploration    = 4  // アイテム発見率: スキルLv1あたり+4%
	coeffEnemyVision    = -3 // 敵視界距離: スキルLv1あたり-3%
	coeffNightVision    = 5  // 暗所視界: スキルLv1あたり+5%
	coeffMoveCost       = -2 // 移動コスト: スキルLv1あたり-2%
	coeffCraftCost      = -3 // 素材消費: スキルLv1あたり-3%
	coeffSmithQuality   = 3  // クラフト品質: スキルLv1あたり+3%
	coeffBuyPrice       = -2 // 買値: スキルLv1あたり-2%
	coeffSellPrice      = 2  // 売値: スキルLv1あたり+2%
	coeffHeavyArmor     = -5 // 重装備ペナルティ: スキルLv1あたり-5%
)

// 能力値ベースの倍率の係数。能力値1ポイントあたりの倍率変化量。スキルを持たない単機能倍率が使う
const (
	coeffActionSpeedAgility   = 2 // 行動速度: AGI1あたり+2%
	coeffActionSpeedDexterity = 1 // 行動速度: DEX1あたり+1%
	coeffRecoveryVitality     = 3 // 自然回復: VIT1あたり+3%
)

// ModifierSourceKind は内訳1件の由来の種別
type ModifierSourceKind string

// 内訳の由来種別
const (
	SourceSkill    ModifierSourceKind = "skill"    // スキルによる補正
	SourceAbility  ModifierSourceKind = "ability"  // 能力値による補正
	SourceCapacity ModifierSourceKind = "capacity" // 身体機能の畳み込み
	SourceFatigue  ModifierSourceKind = "fatigue"  // 疲労段階の畳み込み
	SourceHunger   ModifierSourceKind = "hunger"   // 空腹段階の畳み込み
	SourceSleeping ModifierSourceKind = "sleeping" // 睡眠中の畳み込み
)

// ModifierSource は効果倍率の算出元1件を表す。整形済みの文字列でなく事実を持ち、
// 表示側が現在言語へ訳して整形する。Kind に応じて Skill / Ability / Capacity のどれかが有効
type ModifierSource struct {
	Kind     ModifierSourceKind
	Skill    SkillID      // Kind が skill のときのスキル
	Ability  AbilityID    // Kind が ability のときの能力値
	Capacity CapacityKind // Kind が capacity のときの身体機能
	Fatigue  FatigueLevel // Kind が fatigue のときの疲労段階
	Hunger   HungerLevel  // Kind が hunger のときの空腹段階
	Amount   int          // 要因の量。スキルLv・能力値・身体機能%
	Value    int          // この要因による変化量。例: +10, -15
}

// IsWeaponAccuracyKey は key が武器命中の効果キーかを返す。疲労など武器命中だけに
// 効く補正を query 層で畳むときの判定に使う
func IsWeaponAccuracyKey(key ModifierKey) bool {
	_, ok := accuracySkillByKey[key]
	return ok
}

// weaponAccuracyCapacity は武器スキルの命中に効く身体機能の種別と乗数を返す。
// 近接は操作機能、遠隔は視覚機能。対応する攻撃種が無ければ操作機能を既定にする
func weaponAccuracyCapacity(caps BodyCapacities, id SkillID) (CapacityKind, consts.Percent) {
	for _, at := range AllAttackTypes {
		skillID, ok := WeaponSkillID(at)
		if !ok || skillID != id {
			continue
		}
		if at.Range == AttackRangeRanged {
			return CapacitySight, caps.Sight
		}
		return CapacityManipulation, caps.Manipulation
	}
	return CapacityManipulation, caps.Manipulation
}

// abilityTerm は能力値1ポイントあたりの倍率寄与。単機能倍率で能力値を複数種束ねるために使う
type abilityTerm struct {
	Ability AbilityID
	Coeff   int
}

// modifierSpec は倍率1つの定義。スキル由来と能力値由来の係数を束ねる。
// Skill が空なら能力値ベースの単機能倍率で、行動速度や自然回復がこれにあたる
type modifierSpec struct {
	Key       ModifierKey
	Skill     SkillID       // 元スキル。能力値ベースの倍率では空
	Coeff     int           // スキルLv1あたりの係数
	Abilities []abilityTerm // 能力値1ポイントあたりの係数。スキル基準は元スキルの担当能力を1項導出する
}

// skillSpec はスキル基準の倍率を組む。担当能力の寄与はスキル係数と同じ向きに1ポイント±1%で導出する
func skillSpec(key ModifierKey, skill SkillID, coeff int) modifierSpec {
	ablCoeff := 1
	if coeff < 0 {
		ablCoeff = -1
	}
	return modifierSpec{
		Key:       key,
		Skill:     skill,
		Coeff:     coeff,
		Abilities: []abilityTerm{{SkillAbilityID(skill), ablCoeff}},
	}
}

// modifierSpecs は全倍率の定義表。単発の倍率を足すときはここへ1行足す
var modifierSpecs = buildModifierSpecs()

func buildModifierSpecs() []modifierSpec {
	specs := slices.Grow([]modifierSpec{
		skillSpec(ModFireResist, SkillFireResist, coeffElementResist),
		skillSpec(ModThunderResist, SkillThunderResist, coeffElementResist),
		skillSpec(ModChillResist, SkillChillResist, coeffElementResist),
		skillSpec(ModPhotonResist, SkillPhotonResist, coeffElementResist),
		skillSpec(ModColdProgress, SkillColdResist, coeffColdProgress),
		skillSpec(ModHungerProgress, SkillHungerResist, coeffHungerProgress),
		skillSpec(ModHealingEffect, SkillHealing, coeffHealingEffect),
		skillSpec(ModMaxWeight, SkillWeightBearing, coeffMaxWeight),
		skillSpec(ModExploration, SkillExploration, coeffExploration),
		skillSpec(ModEnemyVision, SkillStealth, coeffEnemyVision),
		skillSpec(ModNightVision, SkillNightVision, coeffNightVision),
		skillSpec(ModMoveCost, SkillSprinting, coeffMoveCost),
		skillSpec(ModCraftCost, SkillCrafting, coeffCraftCost),
		skillSpec(ModSmithQuality, SkillSmithing, coeffSmithQuality),
		skillSpec(ModBuyPrice, SkillNegotiation, coeffBuyPrice),
		skillSpec(ModSellPrice, SkillNegotiation, coeffSellPrice),
		skillSpec(ModHeavyArmor, SkillHeavyArmor, coeffHeavyArmor),
		// 行動速度・自然回復はスキルを持たず能力値だけを土台にする単機能倍率。
		// 疲労・空腹などの状態寄与は query 層の statusSources が同じキーへ足し込む
		{Key: ModActionSpeed, Abilities: []abilityTerm{{AblAGI, coeffActionSpeedAgility}, {AblDEX, coeffActionSpeedDexterity}}},
		{Key: ModRecovery, Abilities: []abilityTerm{{AblVIT, coeffRecoveryVitality}}},
	}, 2*len(WeaponSkillIDs))
	// 武器の行はスキルIDの直積なので生成する
	for _, id := range WeaponSkillIDs {
		specs = append(specs,
			skillSpec(WeaponDamageKey(id), id, coeffWeaponDamage),
			skillSpec(WeaponAccuracyKey(id), id, coeffWeaponAccuracy))
	}
	return specs
}

// specByKey は ModifierKey からスペック行を引く索引
var specByKey = func() map[ModifierKey]modifierSpec {
	m := make(map[ModifierKey]modifierSpec, len(modifierSpecs))
	for _, s := range modifierSpecs {
		m[s.Key] = s
	}
	return m
}()

// accuracySkillByKey は命中キーから武器スキルIDを引く索引。命中だけ身体機能を畳むため
var accuracySkillByKey = func() map[ModifierKey]SkillID {
	m := make(map[ModifierKey]SkillID, len(weaponAccuracyKeys))
	for id, key := range weaponAccuracyKeys {
		m[key] = id
	}
	return m
}()

// KeyRequiresSkill は key の倍率がスキルを土台にするかを返す。行動速度・自然回復のような
// 能力値ベースの単機能倍率はスキルを要らないので false。Skills 非所持でも算出してよいかの判定に使う
func KeyRequiresSkill(key ModifierKey) bool {
	spec, ok := specByKey[key]
	return ok && spec.Skill != ""
}

// forEachModifierSource は key の内訳を計算順に fn へ渡す。値も内訳もこの1関数から導く。
// 最終値 = 基準 + Σ内訳 の不変条件はこの構造そのものが保証する。未定義キーは何も渡さない。
// skills / abils / hs は不在なら nil でよく、その由来のソースは飛ばす
func forEachModifierSource(skills *Skills, abils *Abilities, hs *HealthStatus, key ModifierKey, fn func(ModifierSource)) {
	spec, ok := specByKey[key]
	if !ok {
		return
	}
	bonus := 0

	// スキルによる補正。能力値ベースの倍率は Skill が空なので飛ばす
	if spec.Skill != "" && skills != nil {
		v := skills.Get(spec.Skill).Value
		skillBonus := v * spec.Coeff
		fn(ModifierSource{Kind: SourceSkill, Skill: spec.Skill, Amount: v, Value: skillBonus})
		bonus += skillBonus
	}

	// 能力値による補正。スキル基準は担当能力1項、行動速度などは複数能力を束ねる
	if abils != nil {
		for _, at := range spec.Abilities {
			ablVal := abils.ValueOf(at.Ability)
			ablBonus := ablVal * at.Coeff
			fn(ModifierSource{Kind: SourceAbility, Ability: at.Ability, Amount: ablVal, Value: ablBonus})
			bonus += ablBonus
		}
	}

	// 命中へ効く身体機能を乗算で畳み、内訳には加法差分で載せる
	if id, isAccuracy := accuracySkillByKey[key]; isAccuracy {
		caps := HealthyCapacities()
		if hs != nil {
			caps = hs.Capacities()
		}
		capKind, capVal := weaponAccuracyCapacity(caps, id)
		acc := int(consts.PercentBase) + bonus
		withCap := capVal.ApplyInt(acc)
		fn(ModifierSource{Kind: SourceCapacity, Capacity: capKind, Amount: int(capVal), Value: withCap - acc})
	}
}

// CalcModifierValue は key の効果倍率を導出する。内訳の加法差分を積むだけで
// アロケーションが無い。表示の%も適用もこの関数を読むので両者は一致する
func CalcModifierValue(skills *Skills, abils *Abilities, hs *HealthStatus, key ModifierKey) consts.Percent {
	total := int(consts.PercentBase)
	forEachModifierSource(skills, abils, hs, key, func(s ModifierSource) {
		total += s.Value
	})
	return consts.Percent(total)
}

// CalcModifierSources は key の内訳を返す。詳細モーダルの表示側だけが読む。
// 未定義キーは空を返す
func CalcModifierSources(skills *Skills, abils *Abilities, hs *HealthStatus, key ModifierKey) []ModifierSource {
	var srcs []ModifierSource
	forEachModifierSource(skills, abils, hs, key, func(s ModifierSource) {
		srcs = append(srcs, s)
	})
	return srcs
}
