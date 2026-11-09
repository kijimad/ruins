package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	assert.Len(t, skills.Data, len(AllSkillIDs), "全スキルが初期化される")
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
		{"Swordsmanship", AttackSword, SkillSword, true},
		{"Polearm", AttackSpear, SkillSpear, true},
		{"Unarmed", AttackFist, SkillFist, true},
		{"Pistol skill", AttackHandgun, SkillHandgun, true},
		{"Rifle skill", AttackRifle, SkillRifle, true},
		{"Artillery", AttackCannon, SkillCannon, true},
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

func TestIsSkillID(t *testing.T) {
	t.Parallel()

	for _, id := range AllSkillIDs {
		assert.True(t, IsSkillID(id), "スキル %s は定義済み", id)
	}
	assert.False(t, IsSkillID("undefined_skill"))
}

func TestAllSkillIDs_重複しない(t *testing.T) {
	t.Parallel()

	seen := map[SkillID]bool{}
	for _, id := range AllSkillIDs {
		assert.False(t, seen[id], "スキル %s が重複している", id)
		seen[id] = true
	}
	assert.Len(t, AllSkillIDs, 24)
}

func TestWeaponSkillID_全攻撃種別が同じ名前の武器スキルを持つ(t *testing.T) {
	t.Parallel()

	got := map[AttackType]SkillID{}
	for _, at := range AllAttackTypes {
		id, ok := WeaponSkillID(at)
		require.True(t, ok, at.Type)
		got[at] = id
	}
	assert.Equal(t, map[AttackType]SkillID{
		AttackSword:   SkillSword,
		AttackSpear:   SkillSpear,
		AttackHandgun: SkillHandgun,
		AttackRifle:   SkillRifle,
		AttackFist:    SkillFist,
		AttackCannon:  SkillCannon,
		AttackBow:     SkillBow,
	}, got)
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

	for _, id := range WeaponSkillIDs {
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

	for _, id := range WeaponSkillIDs {
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
