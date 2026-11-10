package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
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
		id       oapi.AbilityId
		expected int
	}{
		{"STR", oapi.Str, 10},
		{"SEN", oapi.Sen, 8},
		{"DEX", oapi.Dex, 6},
		{"AGI", oapi.Agi, 12},
		{"VIT", oapi.Vit, 15},
		{"DEF", oapi.Def, 5},
		{"未定義のIDは0を返す", oapi.AbilityId("unknown"), 0},
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
		assert.Equal(t, &Skill{Exp: IntPool{Max: LevelUpExp}}, skills.Get(oapi.SkillIdSword))
		assert.Equal(t, map[oapi.SkillId]*Skill{oapi.SkillIdSword: {Exp: IntPool{Max: LevelUpExp}}}, skills.Data)
	})

	t.Run("作ったスキルへの書き込みが残る", func(t *testing.T) {
		t.Parallel()
		skills := NewSkills()
		skills.Get(oapi.SkillIdBow).Value = 7
		assert.Equal(t, 7, skills.Get(oapi.SkillIdBow).Value)
	})

	t.Run("Level は無いスキルを0として読み、Data へ足さない", func(t *testing.T) {
		t.Parallel()
		skills := NewSkills()
		assert.Equal(t, 0, skills.Level(oapi.SkillIdSword))
		assert.Equal(t, map[oapi.SkillId]*Skill{}, skills.Data)
	})

	t.Run("ゼロ値の Skills でも使える", func(t *testing.T) {
		t.Parallel()
		var skills Skills
		assert.Equal(t, 0, skills.Get(oapi.SkillIdFist).Value)
	})
}

func TestAbilityNameMapping(t *testing.T) {
	t.Parallel()

	for _, id := range []oapi.AbilityId{oapi.Str, oapi.Sen, oapi.Dex, oapi.Agi, oapi.Vit, oapi.Def} {
		assert.NotPanics(t, func() {
			name := AbilityName(id)
			assert.NotEmpty(t, name)
		})
	}

	// 未定義の能力値IDはpanicする
	assert.Panics(t, func() {
		AbilityName(oapi.AbilityId("unknown"))
	})
}
