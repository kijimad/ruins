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

func TestSkills_Get(t *testing.T) {
	t.Parallel()

	t.Run("まだ無いスキルは値0で作る", func(t *testing.T) {
		t.Parallel()
		skills := NewSkills()
		assert.Equal(t, &Skill{Exp: IntPool{Max: LevelUpExp}}, skills.Get(SkillSword))
		assert.Equal(t, map[SkillID]*Skill{SkillSword: {Exp: IntPool{Max: LevelUpExp}}}, skills.Data)
	})

	t.Run("作ったスキルへの書き込みが残る", func(t *testing.T) {
		t.Parallel()
		skills := NewSkills()
		skills.Get(SkillBow).Value = 7
		assert.Equal(t, 7, skills.Get(SkillBow).Value)
	})

	t.Run("ゼロ値の Skills でも使える", func(t *testing.T) {
		t.Parallel()
		var skills Skills
		assert.Equal(t, 0, skills.Get(SkillFist).Value)
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
