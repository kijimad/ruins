package components

import (
	"fmt"
	"slices"

	"github.com/kijimaD/ruins/internal/consts"
)

// ProficiencyKey は熟練由来の効果倍率の識別キー。技能と能力値から導く行動ごとの倍率を指す。
// 怪我・病気・疲労・空腹による身体機能の低下は BodyFunction 側で別に扱う。熟練は上達、身体機能は劣化で軸が違う
type ProficiencyKey string

// 効果キー定数
const (
	ProfFireResist     ProficiencyKey = "fire_resist"
	ProfThunderResist  ProficiencyKey = "thunder_resist"
	ProfChillResist    ProficiencyKey = "chill_resist"
	ProfPhotonResist   ProficiencyKey = "photon_resist"
	ProfColdProgress   ProficiencyKey = "cold_progress"
	ProfHungerProgress ProficiencyKey = "hunger_progress"
	ProfHealingEffect  ProficiencyKey = "healing_effect"
	ProfMaxWeight      ProficiencyKey = "max_weight"
	ProfExploration    ProficiencyKey = "exploration"
	ProfEnemyVision    ProficiencyKey = "enemy_vision"
	ProfNightVision    ProficiencyKey = "night_vision"
	ProfMoveCost       ProficiencyKey = "move_cost"
	ProfCraftCost      ProficiencyKey = "craft_cost"
	ProfSmithQuality   ProficiencyKey = "smith_quality"
	ProfBuyPrice       ProficiencyKey = "buy_price"
	ProfSellPrice      ProficiencyKey = "sell_price"
	ProfHeavyArmor     ProficiencyKey = "heavy_armor"

	ProfSwordDamage   ProficiencyKey = "sword_damage"
	ProfSpearDamage   ProficiencyKey = "spear_damage"
	ProfFistDamage    ProficiencyKey = "fist_damage"
	ProfBowDamage     ProficiencyKey = "bow_damage"
	ProfHandgunDamage ProficiencyKey = "handgun_damage"
	ProfRifleDamage   ProficiencyKey = "rifle_damage"
	ProfCannonDamage  ProficiencyKey = "cannon_damage"

	ProfSwordAccuracy   ProficiencyKey = "sword_accuracy"
	ProfSpearAccuracy   ProficiencyKey = "spear_accuracy"
	ProfFistAccuracy    ProficiencyKey = "fist_accuracy"
	ProfBowAccuracy     ProficiencyKey = "bow_accuracy"
	ProfHandgunAccuracy ProficiencyKey = "handgun_accuracy"
	ProfRifleAccuracy   ProficiencyKey = "rifle_accuracy"
	ProfCannonAccuracy  ProficiencyKey = "cannon_accuracy"
)

// weaponDamageKeys は武器スキルIDからダメージ効果キーへのマッピング
var weaponDamageKeys = map[SkillID]ProficiencyKey{
	SkillSword:   ProfSwordDamage,
	SkillSpear:   ProfSpearDamage,
	SkillFist:    ProfFistDamage,
	SkillBow:     ProfBowDamage,
	SkillHandgun: ProfHandgunDamage,
	SkillRifle:   ProfRifleDamage,
	SkillCannon:  ProfCannonDamage,
}

// WeaponDamageKey は武器スキルIDに対応するダメージ効果キーを返す。未定義ならpanicする
func WeaponDamageKey(id SkillID) ProficiencyKey {
	key, ok := weaponDamageKeys[id]
	if !ok {
		panic(fmt.Sprintf("undefined weapon skill ID for damage: %q", id))
	}
	return key
}

// weaponAccuracyKeys は武器スキルIDから命中効果キーへのマッピング
var weaponAccuracyKeys = map[SkillID]ProficiencyKey{
	SkillSword:   ProfSwordAccuracy,
	SkillSpear:   ProfSpearAccuracy,
	SkillFist:    ProfFistAccuracy,
	SkillBow:     ProfBowAccuracy,
	SkillHandgun: ProfHandgunAccuracy,
	SkillRifle:   ProfRifleAccuracy,
	SkillCannon:  ProfCannonAccuracy,
}

// WeaponAccuracyKey は武器スキルIDに対応する命中効果キーを返す。未定義ならpanicする
func WeaponAccuracyKey(id SkillID) ProficiencyKey {
	key, ok := weaponAccuracyKeys[id]
	if !ok {
		panic(fmt.Sprintf("undefined weapon skill ID for accuracy: %q", id))
	}
	return key
}

// elementResistKeys は元素タイプから耐性効果キーへのマッピング
var elementResistKeys = map[ElementType]ProficiencyKey{
	ElementTypeFire:    ProfFireResist,
	ElementTypeThunder: ProfThunderResist,
	ElementTypeChill:   ProfChillResist,
	ElementTypePhoton:  ProfPhotonResist,
}

// ElementResistKey は元素タイプに対応する耐性効果キーを返す。未定義ならpanicする
func ElementResistKey(elem ElementType) ProficiencyKey {
	key, ok := LookupElementResistKey(elem)
	if !ok {
		panic(fmt.Sprintf("undefined element type for resistance: %q", elem))
	}
	return key
}

// LookupElementResistKey は元素タイプに対応する耐性効果キーを返す。無属性など未定義は ok=false
func LookupElementResistKey(elem ElementType) (ProficiencyKey, bool) {
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

// ProficiencySourceKind は内訳1件の由来の種別
type ProficiencySourceKind string

// 内訳の由来種別
const (
	SourceSkill        ProficiencySourceKind = "skill"         // スキルによる補正
	SourceAbility      ProficiencySourceKind = "ability"       // 能力値による補正
	SourceBodyFunction ProficiencySourceKind = "body_function" // 身体機能の畳み込み
	SourceFatigue      ProficiencySourceKind = "fatigue"       // 疲労段階の畳み込み
	SourceHunger       ProficiencySourceKind = "hunger"        // 空腹段階の畳み込み
	SourceSleeping     ProficiencySourceKind = "sleeping"      // 睡眠中の畳み込み
)

// ProficiencySource は効果倍率の算出元1件を表す。整形済みの文字列でなく事実を持ち、
// 表示側が現在言語へ訳して整形する。Kind に応じて Skill / Ability / BodyFunction のどれかが有効
type ProficiencySource struct {
	Kind         ProficiencySourceKind
	Skill        SkillID          // Kind が skill のときのスキル
	Ability      AbilityID        // Kind が ability のときの能力値
	BodyFunction BodyFunctionKind // Kind が body function のときの身体機能
	Fatigue      FatigueLevel     // Kind が fatigue のときの疲労段階
	Hunger       HungerLevel      // Kind が hunger のときの空腹段階
	Amount       int              // 要因の量。スキルLv・能力値・身体機能%
	Value        int              // この要因による変化量。例: +10, -15
}

// IsWeaponAccuracyKey は key が武器命中の効果キーかを返す。疲労など武器命中だけに
// 効く補正を query 層で畳むときの判定に使う
func IsWeaponAccuracyKey(key ProficiencyKey) bool {
	_, ok := accuracySkillByKey[key]
	return ok
}

// weaponAccuracyBodyFunction は武器スキルの命中に効く身体機能の種別と乗数を返す。
// 近接は操作機能、遠隔は視覚機能。対応する攻撃種が無ければ操作機能を既定にする
func weaponAccuracyBodyFunction(caps BodyFunctions, id SkillID) (BodyFunctionKind, consts.Percent) {
	for _, at := range AllAttackTypes {
		skillID, ok := WeaponSkillID(at)
		if !ok || skillID != id {
			continue
		}
		if at.Range == AttackRangeRanged {
			return BodyFuncSight, caps.Sight
		}
		return BodyFuncManipulation, caps.Manipulation
	}
	return BodyFuncManipulation, caps.Manipulation
}

// proficiencySpec は倍率1つの定義。キー・元スキル・スキルLv1あたりの係数を束ねる
type proficiencySpec struct {
	Key   ProficiencyKey
	Skill SkillID
	Coeff int
}

// proficiencySpecs は全倍率の定義表。単発の倍率を足すときはここへ1行足す
var proficiencySpecs = buildProficiencySpecs()

func buildProficiencySpecs() []proficiencySpec {
	specs := slices.Grow([]proficiencySpec{
		{ProfFireResist, SkillFireResist, coeffElementResist},
		{ProfThunderResist, SkillThunderResist, coeffElementResist},
		{ProfChillResist, SkillChillResist, coeffElementResist},
		{ProfPhotonResist, SkillPhotonResist, coeffElementResist},
		{ProfColdProgress, SkillColdResist, coeffColdProgress},
		{ProfHungerProgress, SkillHungerResist, coeffHungerProgress},
		{ProfHealingEffect, SkillHealing, coeffHealingEffect},
		{ProfMaxWeight, SkillWeightBearing, coeffMaxWeight},
		{ProfExploration, SkillExploration, coeffExploration},
		{ProfEnemyVision, SkillStealth, coeffEnemyVision},
		{ProfNightVision, SkillNightVision, coeffNightVision},
		{ProfMoveCost, SkillSprinting, coeffMoveCost},
		{ProfCraftCost, SkillCrafting, coeffCraftCost},
		{ProfSmithQuality, SkillSmithing, coeffSmithQuality},
		{ProfBuyPrice, SkillNegotiation, coeffBuyPrice},
		{ProfSellPrice, SkillNegotiation, coeffSellPrice},
		{ProfHeavyArmor, SkillHeavyArmor, coeffHeavyArmor},
	}, 2*len(WeaponSkillIDs))
	// 武器の行はスキルIDの直積なので生成する
	for _, id := range WeaponSkillIDs {
		specs = append(specs,
			proficiencySpec{WeaponDamageKey(id), id, coeffWeaponDamage},
			proficiencySpec{WeaponAccuracyKey(id), id, coeffWeaponAccuracy})
	}
	return specs
}

// specByKey は ProficiencyKey からスペック行を引く索引
var specByKey = func() map[ProficiencyKey]proficiencySpec {
	m := make(map[ProficiencyKey]proficiencySpec, len(proficiencySpecs))
	for _, s := range proficiencySpecs {
		m[s.Key] = s
	}
	return m
}()

// accuracySkillByKey は命中キーから武器スキルIDを引く索引。命中だけ身体機能を畳むため
var accuracySkillByKey = func() map[ProficiencyKey]SkillID {
	m := make(map[ProficiencyKey]SkillID, len(weaponAccuracyKeys))
	for id, key := range weaponAccuracyKeys {
		m[key] = id
	}
	return m
}()

// forEachProficiencySource は key の内訳を計算順に fn へ渡す。値も内訳もこの1関数から導く。
// 最終値 = 基準 + Σ内訳 の不変条件はこの構造そのものが保証する。未定義キーは何も渡さない。
// skills / abils は不在なら nil でよく、その由来のソースは飛ばす。caps は実効身体機能で、
// 疲労・空腹の低下を畳んだものを呼び出し側が渡す
func forEachProficiencySource(skills *Skills, abils *Abilities, caps BodyFunctions, key ProficiencyKey, fn func(ProficiencySource)) {
	spec, ok := specByKey[key]
	if !ok {
		return
	}
	if skills == nil {
		return
	}
	v := skills.Get(spec.Skill).Value
	bonus := v * spec.Coeff
	fn(ProficiencySource{Kind: SourceSkill, Skill: spec.Skill, Amount: v, Value: bonus})

	// 対応する能力値による補正。能力値1ポイントにつきスキル係数と同じ方向に±1%
	if abils != nil {
		ablID := SkillAbilityID(spec.Skill)
		ablVal := abils.ValueOf(ablID)
		ablCoeff := 1
		if spec.Coeff < 0 {
			ablCoeff = -1
		}
		ablBonus := ablVal * ablCoeff
		fn(ProficiencySource{Kind: SourceAbility, Ability: ablID, Amount: ablVal, Value: ablBonus})
		bonus += ablBonus
	}

	// 命中へ効く身体機能を乗算で畳み、内訳には加法差分で載せる。実効身体機能なので
	// 疲労・空腹による意識低下も命中へここで波及する
	if id, isAccuracy := accuracySkillByKey[key]; isAccuracy {
		capKind, capVal := weaponAccuracyBodyFunction(caps, id)
		acc := int(consts.PercentBase) + bonus
		withCap := capVal.ApplyInt(acc)
		fn(ProficiencySource{Kind: SourceBodyFunction, BodyFunction: capKind, Amount: int(capVal), Value: withCap - acc})
	}
}

// CalcProficiencyValue は key の効果倍率を導出する。内訳の加法差分を積むだけで
// アロケーションが無い。表示の%も適用もこの関数を読むので両者は一致する
func CalcProficiencyValue(skills *Skills, abils *Abilities, caps BodyFunctions, key ProficiencyKey) consts.Percent {
	total := int(consts.PercentBase)
	forEachProficiencySource(skills, abils, caps, key, func(s ProficiencySource) {
		total += s.Value
	})
	return consts.Percent(total)
}

// CalcProficiencySources は key の内訳を返す。詳細モーダルの表示側だけが読む。
// 未定義キーは空を返す
func CalcProficiencySources(skills *Skills, abils *Abilities, caps BodyFunctions, key ProficiencyKey) []ProficiencySource {
	var srcs []ProficiencySource
	forEachProficiencySource(skills, abils, caps, key, func(s ProficiencySource) {
		srcs = append(srcs, s)
	})
	return srcs
}
