package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// weaponDamageKeys は武器スキルIDからダメージ効果キーへのマッピング
var weaponDamageKeys = map[oapi.SkillId]oapi.ProficiencyId{
	oapi.SkillIdSword:   oapi.ProficiencyIdSwordDamage,
	oapi.SkillIdSpear:   oapi.ProficiencyIdSpearDamage,
	oapi.SkillIdFist:    oapi.ProficiencyIdFistDamage,
	oapi.SkillIdBow:     oapi.ProficiencyIdBowDamage,
	oapi.SkillIdHandgun: oapi.ProficiencyIdHandgunDamage,
	oapi.SkillIdRifle:   oapi.ProficiencyIdRifleDamage,
	oapi.SkillIdCannon:  oapi.ProficiencyIdCannonDamage,
}

// WeaponDamageKey は武器スキルIDに対応するダメージ効果キーを返す。未定義ならpanicする
func WeaponDamageKey(id oapi.SkillId) oapi.ProficiencyId {
	key, ok := weaponDamageKeys[id]
	if !ok {
		panic(fmt.Sprintf("undefined weapon skill ID for damage: %q", id))
	}
	return key
}

// weaponAccuracyKeys は武器スキルIDから命中効果キーへのマッピング
var weaponAccuracyKeys = map[oapi.SkillId]oapi.ProficiencyId{
	oapi.SkillIdSword:   oapi.ProficiencyIdSwordAccuracy,
	oapi.SkillIdSpear:   oapi.ProficiencyIdSpearAccuracy,
	oapi.SkillIdFist:    oapi.ProficiencyIdFistAccuracy,
	oapi.SkillIdBow:     oapi.ProficiencyIdBowAccuracy,
	oapi.SkillIdHandgun: oapi.ProficiencyIdHandgunAccuracy,
	oapi.SkillIdRifle:   oapi.ProficiencyIdRifleAccuracy,
	oapi.SkillIdCannon:  oapi.ProficiencyIdCannonAccuracy,
}

// WeaponAccuracyKey は武器スキルIDに対応する命中効果キーを返す。未定義ならpanicする
func WeaponAccuracyKey(id oapi.SkillId) oapi.ProficiencyId {
	key, ok := weaponAccuracyKeys[id]
	if !ok {
		panic(fmt.Sprintf("undefined weapon skill ID for accuracy: %q", id))
	}
	return key
}

// elementResistKeys は元素タイプから耐性効果キーへのマッピング
var elementResistKeys = map[ElementType]oapi.ProficiencyId{
	ElementTypeFire:    oapi.ProficiencyIdFireResist,
	ElementTypeThunder: oapi.ProficiencyIdThunderResist,
	ElementTypeChill:   oapi.ProficiencyIdChillResist,
	ElementTypePhoton:  oapi.ProficiencyIdPhotonResist,
}

// ElementResistKey は元素タイプに対応する耐性効果キーを返す。未定義ならpanicする
func ElementResistKey(elem ElementType) oapi.ProficiencyId {
	key, ok := LookupElementResistKey(elem)
	if !ok {
		panic(fmt.Sprintf("undefined element type for resistance: %q", elem))
	}
	return key
}

// LookupElementResistKey は元素タイプに対応する耐性効果キーを返す。無属性など未定義は ok=false
func LookupElementResistKey(elem ElementType) (oapi.ProficiencyId, bool) {
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
	Skill    oapi.SkillId // Kind が skill のときのスキル
	Ability  AbilityID    // Kind が ability のときの能力値
	BodyFunc BodyFuncKind // Kind が body function のときの身体機能
	Fatigue  FatigueLevel // Kind が fatigue のときの疲労段階
	Hunger   HungerLevel  // Kind が hunger のときの空腹段階
	Amount   int          // 要因の量。スキルLv・能力値・身体機能%
	Value    int          // この要因による変化量。例: +10, -15
}

// weaponAccuracyBodyFunc は武器スキルの命中に効く身体機能の種別と乗数を返す。
// 近接は操作機能、遠隔は視覚機能。対応する攻撃種が無ければ操作機能を既定にする
func weaponAccuracyBodyFunc(bodyFuncs BodyFuncs, id oapi.SkillId) (BodyFuncKind, consts.Percent) {
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
var accuracySkillByKey = func() map[oapi.ProficiencyId]oapi.SkillId {
	m := make(map[oapi.ProficiencyId]oapi.SkillId, len(weaponAccuracyKeys))
	for id, key := range weaponAccuracyKeys {
		m[key] = id
	}
	return m
}()

// forEachProficiencySource は key の内訳を計算順に fn へ渡す。値も内訳もこの1関数から導く。
// 最終値 = 基準 + Σ内訳 の不変条件はこの構造そのものが保証する。未定義キーは何も渡さない。
// skills / abils は不在なら nil でよく、その由来のソースは飛ばす。bodyFuncs は実効身体機能で、
// 疲労・空腹の低下を畳んだものを呼び出し側が渡す
func (d SkillDefs) forEachProficiencySource(skills *Skills, abils *Abilities, bodyFuncs BodyFuncs, key oapi.ProficiencyId, fn func(ProficiencySource)) {
	spec, ok := d.Proficiencies[key]
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
func (d SkillDefs) ProficiencyValue(skills *Skills, abils *Abilities, bodyFuncs BodyFuncs, key oapi.ProficiencyId) consts.Percent {
	total := int(consts.PercentBase)
	d.forEachProficiencySource(skills, abils, bodyFuncs, key, func(s ProficiencySource) {
		total += s.Value
	})
	return consts.Percent(total)
}

// ProficiencySources は key の内訳を返す。詳細モーダルの表示側だけが読む。
// 未定義キーは空を返す
func (d SkillDefs) ProficiencySources(skills *Skills, abils *Abilities, bodyFuncs BodyFuncs, key oapi.ProficiencyId) []ProficiencySource {
	var srcs []ProficiencySource
	d.forEachProficiencySource(skills, abils, bodyFuncs, key, func(s ProficiencySource) {
		srcs = append(srcs, s)
	})
	return srcs
}
