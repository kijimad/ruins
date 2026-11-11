package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/rawschema"
)

// SkillDef はスキルごとの定義。raw の skills の1行から組み立てる
type SkillDef struct {
	Name     i18n.Msgid          // 表示名
	Summary  i18n.Msgid          // 概要文
	GainedBy i18n.Msgid          // 獲得条件
	Effect   i18n.Msgid          // 効果
	Ability  rawschema.AbilityId // 成長の速さと熟練の効果倍率に効く能力値
}

// SkillCategory はスキルのカテゴリを表す
type SkillCategory struct {
	Name i18n.Msgid          // カテゴリの表示名
	IDs  []rawschema.SkillId // カテゴリに属するスキルID。表示順
}

// ProficiencyDef は熟練の効果倍率1つの定義
type ProficiencyDef struct {
	Skill    rawschema.SkillId // 倍率を左右するスキル
	PerLevel int               // スキルLv1あたりの倍率の変化量。百分率。負ならスキルが高いほど効果が減る
	BodyFunc BodyFuncKind      // 低下で倍率を目減りさせる身体機能。空なら目減りしない
}

// SkillDefs はスキルの定義、カテゴリの表示順、熟練の効果倍率の表。
// 読み込み時に raw から1度だけ組み立て、以後は変えない
type SkillDefs struct {
	Skills        map[rawschema.SkillId]SkillDef
	Categories    []SkillCategory // 表示順
	Proficiencies map[rawschema.ProficiencyId]ProficiencyDef
	Weapons       []WeaponDef                                  // 表示順
	Resists       []ResistDef                                  // 表示順
	ToolSkills    map[rawschema.ToolCategory]rawschema.SkillId // 合う道具で壊したとき伸びるスキル。伸ばさない分類は持たない
}

// ResistDef は攻撃属性と、そのダメージを軽減する熟練の効果倍率。raw の熟練の効果倍率の行の resists から組み立てる
type ResistDef struct {
	Element     rawschema.Element
	Proficiency rawschema.ProficiencyId
}

// Resist は攻撃属性のダメージを軽減する効果倍率を返す。無属性など軽減する倍率が無ければ ok=false
func (d SkillDefs) Resist(e rawschema.Element) (rawschema.ProficiencyId, bool) {
	for _, r := range d.Resists {
		if r.Element == e {
			return r.Proficiency, true
		}
	}
	return "", false
}

// WeaponDef は攻撃の種類が上達に使うスキルと、ダメージと命中を決める熟練の効果倍率。raw の attackTypes の1行から組み立てる
type WeaponDef struct {
	AttackType  rawschema.AttackCategory
	Name        i18n.Msgid // 表示名
	Range       rawschema.AttackRange
	RangeParams RangeParams // 遠距離だけが持つ。近接はゼロ値
	Skill       rawschema.SkillId
	Damage      rawschema.ProficiencyId
	Accuracy    rawschema.ProficiencyId
}

// RangeParams は遠距離の攻撃の種類の射程を返す。近接や表に無い種類は ok=false
func (d SkillDefs) RangeParams(at rawschema.AttackCategory) (RangeParams, bool) {
	w, ok := d.Weapon(at)
	if !ok || w.Range != rawschema.AttackRangeRanged {
		return RangeParams{}, false
	}
	return w.RangeParams, true
}

// Weapon は攻撃の種類の WeaponDef を返す。無ければ ok=false
func (d SkillDefs) Weapon(at rawschema.AttackCategory) (WeaponDef, bool) {
	for _, w := range d.Weapons {
		if w.AttackType == at {
			return w, true
		}
	}
	return WeaponDef{}, false
}

// Name はスキルの表示名 msgid を返す。未登録なら id をそのまま返す
func (d SkillDefs) Name(id rawschema.SkillId) i18n.Msgid {
	if def, ok := d.Skills[id]; ok {
		return def.Name
	}
	return i18n.Msgid(id)
}

// Ability はスキルに対応する能力値を返す。未登録なら panic する
func (d SkillDefs) Ability(id rawschema.SkillId) rawschema.AbilityId {
	def, ok := d.Skills[id]
	if !ok {
		panic(fmt.Sprintf("no ability defined for skill %q", id))
	}
	return def.Ability
}
