package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueOf(t *testing.T) {
	t.Parallel()

	abils := &Abilities{
		Strength:  Ability{Total: 10},
		Sensation: Ability{Total: 8},
		Dexterity: Ability{Total: 6},
		Agility:   Ability{Total: 12},
		Vitality:  Ability{Total: 15},
		Defense:   Ability{Total: 5},
	}

	tests := []struct {
		name     string
		id       AbilityID
		expected int
	}{
		{"STR", AblSTR, 10},
		{"SEN", AblSEN, 8},
		{"DEX", AblDEX, 6},
		{"AGI", AblAGI, 12},
		{"VIT", AblVIT, 15},
		{"DEF", AblDEF, 5},
		{"未定義のIDは0を返す", AbilityID(99), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, abils.ValueOf(tt.id))
		})
	}
}

func TestNewSkills(t *testing.T) {
	t.Parallel()

	skills := NewSkills()

	assert.Equal(t, len(AllSkillIDs), len(skills.Data), "全スキルが初期化される")
	for _, id := range AllSkillIDs {
		s := skills.Get(id)
		assert.NotNil(t, s, "スキル %s が存在する", id)
		assert.Equal(t, 0, s.Value, "初期スキル値は0")
		assert.Equal(t, 0, s.Exp.Current, "初期経験値は0")
		assert.Equal(t, LevelUpExp, s.Exp.Max, "経験値上限はLevelUpExp")
	}

	// 未定義のスキルIDはpanicする
	assert.Panics(t, func() {
		skills.Get("undefined_skill")
	})
}

func TestWeaponSkillID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		attackType AttackType
		expected   SkillID
		ok         bool
	}{
		{"刀剣", AttackSword, SkillSword, true},
		{"長物", AttackSpear, SkillSpear, true},
		{"格闘", AttackFist, SkillFist, true},
		{"拳銃", AttackHandgun, SkillHandgun, true},
		{"小銃", AttackRifle, SkillRifle, true},
		{"砲撃", AttackCanon, SkillCannon, true},
		{"未定義の武器種", AttackType{Type: "unknown"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, ok := WeaponSkillID(tt.attackType)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.expected, id)
			}
		})
	}
}

func TestAllSkillIDs(t *testing.T) {
	t.Parallel()

	// AllSkillIDsはSkillCategoriesのすべてのスキルを含む
	total := 0
	for _, cat := range SkillCategories {
		total += len(cat.IDs)
	}
	assert.Equal(t, total, len(AllSkillIDs), "AllSkillIDsはカテゴリの合計と一致する")
}

func TestSkillAbilityMapping(t *testing.T) {
	t.Parallel()

	// 全スキルにSkillAbilityIDマッピングがある
	for _, id := range AllSkillIDs {
		assert.NotPanics(t, func() {
			SkillAbilityID(id)
		}, "スキル %s に対応する能力値がマッピングされている", id)
	}

	// 未定義のスキルIDはpanicする
	assert.Panics(t, func() {
		SkillAbilityID("undefined_skill")
	})
}

func TestSkillNameMapping(t *testing.T) {
	t.Parallel()

	// 全スキルにSkillNameがある
	for _, id := range AllSkillIDs {
		assert.NotPanics(t, func() {
			name := SkillName(id)
			assert.NotEmpty(t, name, "スキル %s の表示名が空でない", id)
		})
	}

	// 未定義のスキルIDはpanicする
	assert.Panics(t, func() {
		SkillName("undefined_skill")
	})
}

func TestSkillDescriptionMapping(t *testing.T) {
	t.Parallel()

	// 全スキルにSkillDescriptionがある
	for _, id := range AllSkillIDs {
		assert.NotPanics(t, func() {
			info := SkillDescription(id)
			assert.NotEmpty(t, info.Summary, "スキル %s の概要が空でない", id)
			assert.NotEmpty(t, info.GainedBy, "スキル %s の獲得条件が空でない", id)
			assert.NotEmpty(t, info.Effect, "スキル %s の効果が空でない", id)
		})
	}

	// 未定義のスキルIDはpanicする
	assert.Panics(t, func() {
		SkillDescription("undefined_skill")
	})
}

func TestAbilityNameMapping(t *testing.T) {
	t.Parallel()

	for _, id := range []AbilityID{AblSTR, AblSEN, AblDEX, AblAGI, AblVIT, AblDEF} {
		assert.NotPanics(t, func() {
			name := AbilityName(id)
			assert.NotEmpty(t, name)
		})
	}

	// 未定義の能力値IDはpanicする
	assert.Panics(t, func() {
		AbilityName(AbilityID(99))
	})
}

func TestWeaponDamageKeyMapping(t *testing.T) {
	t.Parallel()

	for _, id := range weaponSkillIDs {
		assert.NotPanics(t, func() {
			key := WeaponDamageKey(id)
			assert.NotEmpty(t, key)
		})
	}

	// 未定義の武器スキルIDはpanicする
	assert.Panics(t, func() {
		WeaponDamageKey("undefined_skill")
	})
}

func TestWeaponAccuracyKeyMapping(t *testing.T) {
	t.Parallel()

	for _, id := range weaponSkillIDs {
		assert.NotPanics(t, func() {
			key := WeaponAccuracyKey(id)
			assert.NotEmpty(t, key)
		})
	}

	// 未定義の武器スキルIDはpanicする
	assert.Panics(t, func() {
		WeaponAccuracyKey("undefined_skill")
	})
}

func TestElementResistKeyMapping(t *testing.T) {
	t.Parallel()

	for _, elem := range []ElementType{ElementTypeFire, ElementTypeThunder, ElementTypeChill, ElementTypePhoton} {
		assert.NotPanics(t, func() {
			key := ElementResistKey(elem)
			assert.NotEmpty(t, key)
		})
	}

	// 未定義の元素タイプはpanicする
	assert.Panics(t, func() {
		ElementResistKey("undefined_element")
	})
}
