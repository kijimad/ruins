package states

import (
	"image/color"
	"testing"

	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
)

func TestChargeColor_割合で色を選ぶ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		stage int
		want  color.RGBA
	}{
		{stage: 5, want: theme.ChargeHigh},
		{stage: 4, want: theme.ChargeHigh},
		{stage: 3, want: theme.ChargeMid},
		{stage: 2, want: theme.ChargeMid},
		{stage: 1, want: theme.ChargeLow},
		{stage: 0, want: theme.ChargeLow},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, chargeColor(tt.stage), tt.stage)
	}
}
