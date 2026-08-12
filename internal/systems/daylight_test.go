package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestOverworldDaylight は時間帯から地上の日照の明るさへの写像を固定する。
// 昼が最も明るく深夜が最も暗いという順序と各値を検証する。
func TestOverworldDaylight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tod  gc.TimeOfDay
		want float64
	}{
		{"夜明けは薄暗い", gc.TimeDawn, 0.40},
		{"朝は明るい", gc.TimeMorning, 0.72},
		{"昼は最も明るい", gc.TimeDay, 0.95},
		{"夕は薄暗い", gc.TimeEvening, 0.38},
		{"夜は暗い", gc.TimeNight, 0.14},
		{"深夜は最も暗い", gc.TimeMidnight, 0.06},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, overworldDaylight(tt.tod), 1e-9)
		})
	}
}

// TestOverworldDaylight_昼夜の大小関係 は昼が最も明るく深夜が最も暗いことを保証する。
// 個々の値を変えても、この不変条件が崩れないことを検知する。
func TestOverworldDaylight_昼夜の大小関係(t *testing.T) {
	t.Parallel()

	assert.Greater(t, overworldDaylight(gc.TimeDay), overworldDaylight(gc.TimeNight),
		"昼は夜より明るい")
	assert.Greater(t, overworldDaylight(gc.TimeNight), overworldDaylight(gc.TimeMidnight),
		"夜は深夜より明るい")
}
