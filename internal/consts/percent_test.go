package consts_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestPercent_ApplyInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pct  consts.Percent
		base int
		want int
	}{
		{"等倍は変えない", consts.PercentBase, 200, 200},
		{"1.2倍", 120, 100, 120},
		{"半分", 50, 100, 50},
		{"0倍", 0, 100, 0},
		{"端数は切り捨て", 33, 10, 3}, // 10*33/100 = 3.3 -> 3
		{"増幅は100超", 250, 4, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.pct.ApplyInt(tt.base))
		})
	}
}

func TestPercent_ApplyFloat(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 120.0, consts.Percent(120).ApplyFloat(100), 1e-9)
	assert.InDelta(t, 3.3, consts.Percent(33).ApplyFloat(10), 1e-9, "float は切り捨てない")
	assert.InDelta(t, 100.0, consts.PercentBase.ApplyFloat(100), 1e-9)
}
