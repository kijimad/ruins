package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestAbilities_BodyWeight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		vitality int
		strength int
		wantKg   int
	}{
		{"体格ゼロは基準体重", 0, 0, 40},
		{"標準的な体格", 5, 5, 50},
		{"頑健な体格ほど重い", 11, 11, 62},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := Abilities{
				Vitality: Ability{Base: tt.vitality},
				Strength: Ability{Base: tt.strength},
			}
			assert.Equal(t, consts.Milligram(tt.wantKg*consts.MilligramPerKg), a.BodyWeight())
		})
	}
}

func TestAbilityName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   rawschema.AbilityId
		want i18n.Msgid
	}{
		{rawschema.Str, "STR"},
		{rawschema.Sen, "SEN"},
		{rawschema.Dex, "DEX"},
		{rawschema.Agi, "AGI"},
		{rawschema.Vit, "VIT"},
		{rawschema.Def, "DEF"},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, AbilityName(tt.id))
		})
	}
}

func TestAbilityName_InvalidID(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		AbilityName(rawschema.AbilityId("unknown"))
	})
}

func TestAbilities_ValueOf(t *testing.T) {
	t.Parallel()

	a := &Abilities{
		Strength:  Ability{Total: 10},
		Sensation: Ability{Total: 20},
		Dexterity: Ability{Total: 30},
		Agility:   Ability{Total: 40},
		Vitality:  Ability{Total: 50},
		Defense:   Ability{Total: 60},
	}

	assert.Equal(t, 10, a.ValueOf(rawschema.Str))
	assert.Equal(t, 20, a.ValueOf(rawschema.Sen))
	assert.Equal(t, 30, a.ValueOf(rawschema.Dex))
	assert.Equal(t, 40, a.ValueOf(rawschema.Agi))
	assert.Equal(t, 50, a.ValueOf(rawschema.Vit))
	assert.Equal(t, 60, a.ValueOf(rawschema.Def))
	assert.Equal(t, 0, a.ValueOf(rawschema.AbilityId("unknown")), "未定義IDは0を返す")
}
