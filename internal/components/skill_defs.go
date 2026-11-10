package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/oapi"
)

// SkillDef はスキルごとの定義。raw の skills の1行から組み立てる
type SkillDef struct {
	Name     string         // 表示名 msgid
	Summary  string         // 概要文 msgid
	GainedBy string         // 獲得条件 msgid
	Effect   string         // 効果 msgid
	Ability  oapi.AbilityId // 成長の速さと熟練の効果倍率に効く能力値
}

// SkillCategory はスキルのカテゴリを表す
type SkillCategory struct {
	Name string         // カテゴリの表示名
	IDs  []oapi.SkillId // カテゴリに属するスキルID。表示順
}

// ProficiencyDef は熟練の効果倍率1つの定義
type ProficiencyDef struct {
	Skill    oapi.SkillId // 倍率を左右するスキル
	PerLevel int          // スキルLv1あたりの倍率の変化量。百分率。負ならスキルが高いほど効果が減る
	BodyFunc BodyFuncKind // 低下で倍率を目減りさせる身体機能。空なら目減りしない
}

// SkillDefs はスキルの定義、カテゴリの表示順、熟練の効果倍率の表。
// 読み込み時に raw から1度だけ組み立て、以後は変えない
type SkillDefs struct {
	Skills        map[oapi.SkillId]SkillDef
	Categories    []SkillCategory // 表示順
	Proficiencies map[oapi.ProficiencyId]ProficiencyDef
	Weapons       []WeaponDef // 表示順
	Resists       []ResistDef // 表示順
}

// ResistDef は攻撃属性と、そのダメージを軽減する熟練の効果倍率。raw の熟練の効果倍率の行の resists から組み立てる
type ResistDef struct {
	Element     oapi.Element
	Proficiency oapi.ProficiencyId
}

// Resist は攻撃属性のダメージを軽減する効果倍率を返す。無属性など軽減する倍率が無ければ ok=false
func (d SkillDefs) Resist(e oapi.Element) (oapi.ProficiencyId, bool) {
	for _, r := range d.Resists {
		if r.Element == e {
			return r.Proficiency, true
		}
	}
	return "", false
}

// WeaponDef は攻撃の種類が上達に使うスキルと、ダメージと命中を決める熟練の効果倍率。raw の attackTypes の1行から組み立てる
type WeaponDef struct {
	AttackType AttackType
	Skill      oapi.SkillId
	Damage     oapi.ProficiencyId
	Accuracy   oapi.ProficiencyId
}

// Weapon は攻撃の種類の WeaponDef を返す。無ければ ok=false
func (d SkillDefs) Weapon(at AttackType) (WeaponDef, bool) {
	for _, w := range d.Weapons {
		if w.AttackType.Type == at.Type {
			return w, true
		}
	}
	return WeaponDef{}, false
}

// Name はスキルの表示名 msgid を返す。未登録なら id をそのまま返す
func (d SkillDefs) Name(id oapi.SkillId) string {
	if def, ok := d.Skills[id]; ok {
		return def.Name
	}
	return string(id)
}

// Ability はスキルに対応する能力値を返す。未登録なら panic する
func (d SkillDefs) Ability(id oapi.SkillId) oapi.AbilityId {
	def, ok := d.Skills[id]
	if !ok {
		panic(fmt.Sprintf("no ability defined for skill %q", id))
	}
	return def.Ability
}
