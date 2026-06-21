package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbilityName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id   AbilityID
		want string
	}{
		{AblSTR, "STR"},
		{AblSEN, "SEN"},
		{AblDEX, "DEX"},
		{AblAGI, "AGI"},
		{AblVIT, "VIT"},
		{AblDEF, "DEF"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, AbilityName(tt.id))
		})
	}
}

func TestAbilityName_InvalidID(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		AbilityName(AbilityID(999))
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

	assert.Equal(t, 10, a.ValueOf(AblSTR))
	assert.Equal(t, 20, a.ValueOf(AblSEN))
	assert.Equal(t, 30, a.ValueOf(AblDEX))
	assert.Equal(t, 40, a.ValueOf(AblAGI))
	assert.Equal(t, 50, a.ValueOf(AblVIT))
	assert.Equal(t, 60, a.ValueOf(AblDEF))
	assert.Equal(t, 0, a.ValueOf(AbilityID(999)), "未定義IDは0を返す")
}
