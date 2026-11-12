package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
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
		id       rawschema.AbilityId
		expected int
	}{
		{"STR", rawschema.Str, 10},
		{"SEN", rawschema.Sen, 8},
		{"DEX", rawschema.Dex, 6},
		{"AGI", rawschema.Agi, 12},
		{"VIT", rawschema.Vit, 15},
		{"DEF", rawschema.Def, 5},
		{"未定義のIDは0を返す", rawschema.AbilityId("unknown"), 0},
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
		assert.Equal(t, &Skill{Exp: IntPool{Max: LevelUpExp}}, skills.Get(rawschema.SkillIdSword))
		assert.Equal(t, map[rawschema.SkillId]*Skill{rawschema.SkillIdSword: {Exp: IntPool{Max: LevelUpExp}}}, skills.Data)
	})

	t.Run("作ったスキルへの書き込みが残る", func(t *testing.T) {
		t.Parallel()
		skills := NewSkills()
		skills.Get(rawschema.SkillIdBow).Value = 7
		assert.Equal(t, 7, skills.Get(rawschema.SkillIdBow).Value)
	})

	t.Run("Level は無いスキルを0として読み、Data へ足さない", func(t *testing.T) {
		t.Parallel()
		skills := NewSkills()
		assert.Equal(t, 0, skills.Level(rawschema.SkillIdSword))
		assert.Equal(t, map[rawschema.SkillId]*Skill{}, skills.Data)
	})

	t.Run("ゼロ値の Skills でも使える", func(t *testing.T) {
		t.Parallel()
		var skills Skills
		assert.Equal(t, 0, skills.Get(rawschema.SkillIdFist).Value)
	})
}

func TestAbilityNameMapping(t *testing.T) {
	t.Parallel()

	for _, id := range []rawschema.AbilityId{rawschema.Str, rawschema.Sen, rawschema.Dex, rawschema.Agi, rawschema.Vit, rawschema.Def} {
		assert.NotPanics(t, func() {
			name := AbilityName(id)
			assert.NotEmpty(t, name)
		})
	}

	// 未定義の能力値IDはpanicする
	assert.Panics(t, func() {
		AbilityName(rawschema.AbilityId("unknown"))
	})
}
