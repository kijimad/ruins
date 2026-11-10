package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestElementResistKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		elem oapi.Element
		want oapi.ProficiencyId
	}{
		{oapi.ElementFIRE, oapi.ProficiencyIdFireResist},
		{oapi.ElementTHUNDER, oapi.ProficiencyIdThunderResist},
		{oapi.ElementCHILL, oapi.ProficiencyIdChillResist},
		{oapi.ElementPHOTON, oapi.ProficiencyIdPhotonResist},
	}

	for _, tt := range tests {
		t.Run(string(tt.elem), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ElementResistKey(tt.elem))
		})
	}
}

func TestElementResistKey_InvalidType(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		ElementResistKey("invalid")
	})
}
