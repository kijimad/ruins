package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestDarknessForTimeOfDay は時間帯から地上の暗さへの写像を固定する。
// 昼が最も明るく、深夜が最も暗いという順序と各値を検証する。
func TestDarknessForTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tod  gc.TimeOfDay
		want float64
	}{
		{"夜明けは薄暗い", gc.TimeDawn, 0.45},
		{"朝は明るい", gc.TimeMorning, 0.12},
		{"昼は最も明るく素通し", gc.TimeDay, 0.0},
		{"夕は薄暗い", gc.TimeEvening, 0.4},
		{"夜は暗い", gc.TimeNight, 0.75},
		{"深夜は最も暗い", gc.TimeMidnight, 0.9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, darknessForTimeOfDay(tt.tod), 1e-9)
		})
	}
}

// TestDarknessForTimeOfDay_昼夜の大小関係 は昼が最も明るく深夜が最も暗いことを保証する。
// 個々の値を変えても、この不変条件が崩れないことを検知する。
func TestDarknessForTimeOfDay_昼夜の大小関係(t *testing.T) {
	t.Parallel()

	assert.Less(t, darknessForTimeOfDay(gc.TimeDay), darknessForTimeOfDay(gc.TimeNight),
		"昼は夜より明るい")
	assert.Less(t, darknessForTimeOfDay(gc.TimeNight), darknessForTimeOfDay(gc.TimeMidnight),
		"夜は深夜より明るい")
}
