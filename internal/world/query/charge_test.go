package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChargeStage_割るたびに1段下がる(t *testing.T) {
	t.Parallel()

	tests := []struct {
		current int
		want    int
	}{
		{current: 600, want: 5},
		{current: 481, want: 5},
		{current: 480, want: 4},
		{current: 361, want: 4},
		{current: 360, want: 3},
		{current: 120, want: 1},
		{current: 1, want: 1},
		{current: 0, want: 0},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, chargeStage(tt.current, 600), tt.current)
	}
}
