package theme

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChargeColor_満ちた段の割合で色を選ぶ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		filled int
		want   color.RGBA
	}{
		{filled: 5, want: ChargeHigh},
		{filled: 4, want: ChargeHigh},
		{filled: 3, want: ChargeMid},
		{filled: 2, want: ChargeMid},
		{filled: 1, want: ChargeLow},
		{filled: 0, want: ChargeLow},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ChargeColor(tt.filled, 5), tt.filled)
	}
}
