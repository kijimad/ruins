package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/oapi"
)

// SkillDef はスキルごとの定義。raw の skills の1行から組み立てる
type SkillDef struct {
	Name     string    // 表示名 msgid
	Summary  string    // 概要文 msgid
	GainedBy string    // 獲得条件 msgid
	Effect   string    // 効果 msgid
	Ability  AbilityID // 成長の速さと熟練の効果倍率に効く能力値
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
}

// SkillDefs はスキルの定義、カテゴリの表示順、熟練の効果倍率の表。
// 読み込み時に raw から1度だけ組み立て、以後は変えない
type SkillDefs struct {
	Skills        map[oapi.SkillId]SkillDef
	Categories    []SkillCategory // 表示順
	Proficiencies map[oapi.ProficiencyId]ProficiencyDef
}

// Name はスキルの表示名 msgid を返す。未登録なら id をそのまま返す
func (d SkillDefs) Name(id oapi.SkillId) string {
	if def, ok := d.Skills[id]; ok {
		return def.Name
	}
	return string(id)
}

// Ability はスキルに対応する能力値を返す。未登録なら panic する
func (d SkillDefs) Ability(id oapi.SkillId) AbilityID {
	def, ok := d.Skills[id]
	if !ok {
		panic(fmt.Sprintf("no ability defined for skill %q", id))
	}
	return def.Ability
}
