package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
)

// ProficiencyKey は熟練由来の効果倍率の識別キー。技能と能力値から導く行動ごとの倍率を指す。
// 怪我・病気・疲労・空腹による身体機能の低下は BodyFunc 側で別に扱う。熟練は上達、身体機能は劣化で軸が違う
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

// ProficiencyKeys はコードが名前で参照する効果キーの一覧を返す。raw に行があることの検証に使う
func ProficiencyKeys() []ProficiencyKey {
	return []ProficiencyKey{
		ProfFireResist, ProfThunderResist, ProfChillResist, ProfPhotonResist,
		ProfColdProgress, ProfHungerProgress, ProfHealingEffect, ProfMaxWeight,
		ProfExploration, ProfEnemyVision, ProfNightVision, ProfMoveCost,
		ProfCraftCost, ProfSmithQuality, ProfBuyPrice, ProfSellPrice, ProfHeavyArmor,
		ProfSwordDamage, ProfSpearDamage, ProfFistDamage, ProfBowDamage, ProfHandgunDamage, ProfRifleDamage, ProfCannonDamage,
		ProfSwordAccuracy, ProfSpearAccuracy, ProfFistAccuracy, ProfBowAccuracy, ProfHandgunAccuracy, ProfRifleAccuracy, ProfCannonAccuracy,
	}
}

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

// ProficiencySourceKind は内訳1件の由来の種別
type ProficiencySourceKind string

// 内訳の由来種別
const (
	SourceSkill    ProficiencySourceKind = "skill"         // スキルによる補正
	SourceAbility  ProficiencySourceKind = "ability"       // 能力値による補正
	SourceBodyFunc ProficiencySourceKind = "body_function" // 身体機能の畳み込み
	SourceFatigue  ProficiencySourceKind = "fatigue"       // 疲労段階の畳み込み
	SourceHunger   ProficiencySourceKind = "hunger"        // 空腹段階の畳み込み
	SourceSleeping ProficiencySourceKind = "sleeping"      // 睡眠中の畳み込み
)

// ProficiencySource は効果倍率の算出元1件を表す。整形済みの文字列でなく事実を持ち、
// 表示側が現在言語へ訳して整形する。Kind に応じて Skill / Ability / BodyFunc のどれかが有効
type ProficiencySource struct {
	Kind     ProficiencySourceKind
	Skill    SkillID      // Kind が skill のときのスキル
	Ability  AbilityID    // Kind が ability のときの能力値
	BodyFunc BodyFuncKind // Kind が body function のときの身体機能
	Fatigue  FatigueLevel // Kind が fatigue のときの疲労段階
	Hunger   HungerLevel  // Kind が hunger のときの空腹段階
	Amount   int          // 要因の量。スキルLv・能力値・身体機能%
	Value    int          // この要因による変化量。例: +10, -15
}

// weaponAccuracyBodyFunc は武器スキルの命中に効く身体機能の種別と乗数を返す。
// 近接は操作機能、遠隔は視覚機能。対応する攻撃種が無ければ操作機能を既定にする
func weaponAccuracyBodyFunc(bodyFuncs BodyFuncs, id SkillID) (BodyFuncKind, consts.Percent) {
	for _, at := range AllAttackTypes {
		skillID, ok := WeaponSkillID(at)
		if !ok || skillID != id {
			continue
		}
		if at.Range == AttackRangeRanged {
			return BodyFuncSight, bodyFuncs.Sight
		}
		return BodyFuncManipulation, bodyFuncs.Manipulation
	}
	return BodyFuncManipulation, bodyFuncs.Manipulation
}

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
// skills / abils は不在なら nil でよく、その由来のソースは飛ばす。bodyFuncs は実効身体機能で、
// 疲労・空腹の低下を畳んだものを呼び出し側が渡す
func (d SkillDefs) forEachProficiencySource(skills *Skills, abils *Abilities, bodyFuncs BodyFuncs, key ProficiencyKey, fn func(ProficiencySource)) {
	spec, ok := d.proficiencies[key]
	if !ok {
		return
	}
	if skills == nil {
		return
	}
	v := skills.Get(spec.Skill).Value
	bonus := v * spec.PerLevel
	fn(ProficiencySource{Kind: SourceSkill, Skill: spec.Skill, Amount: v, Value: bonus})

	// 対応する能力値による補正。能力値1ポイントにつきスキル係数と同じ方向に±1%
	if abils != nil {
		ablID := d.Ability(spec.Skill)
		ablVal := abils.ValueOf(ablID)
		ablCoeff := 1
		if spec.PerLevel < 0 {
			ablCoeff = -1
		}
		ablBonus := ablVal * ablCoeff
		fn(ProficiencySource{Kind: SourceAbility, Ability: ablID, Amount: ablVal, Value: ablBonus})
		bonus += ablBonus
	}

	// 命中へ効く身体機能を乗算で畳み、内訳には加法差分で載せる。bodyFuncs は実効身体機能なので、
	// 部位の怪我に加え全身性の不調・疲労・空腹が既に畳まれている。ここは該当機能を掛けるだけ
	if id, isAccuracy := accuracySkillByKey[key]; isAccuracy {
		limbKind, limbVal := weaponAccuracyBodyFunc(bodyFuncs, id)
		acc := int(consts.PercentBase) + bonus
		withCap := limbVal.ApplyInt(acc)
		if withCap != acc {
			fn(ProficiencySource{Kind: SourceBodyFunc, BodyFunc: limbKind, Amount: int(limbVal), Value: withCap - acc})
		}
	}
}

// ProficiencyValue は key の効果倍率を導出する。内訳の加法差分を積むだけで
// アロケーションが無い。表示の%も適用もこの関数を読むので両者は一致する
func (d SkillDefs) ProficiencyValue(skills *Skills, abils *Abilities, bodyFuncs BodyFuncs, key ProficiencyKey) consts.Percent {
	total := int(consts.PercentBase)
	d.forEachProficiencySource(skills, abils, bodyFuncs, key, func(s ProficiencySource) {
		total += s.Value
	})
	return consts.Percent(total)
}

// ProficiencySources は key の内訳を返す。詳細モーダルの表示側だけが読む。
// 未定義キーは空を返す
func (d SkillDefs) ProficiencySources(skills *Skills, abils *Abilities, bodyFuncs BodyFuncs, key ProficiencyKey) []ProficiencySource {
	var srcs []ProficiencySource
	d.forEachProficiencySource(skills, abils, bodyFuncs, key, func(s ProficiencySource) {
		srcs = append(srcs, s)
	})
	return srcs
}
