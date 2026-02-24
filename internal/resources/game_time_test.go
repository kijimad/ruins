package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeOfDay_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		time     TimeOfDay
		expected string
	}{
		{TimeDawn, "夜明け"},
		{TimeMorning, "朝"},
		{TimeDay, "昼"},
		{TimeEvening, "夕"},
		{TimeNight, "夜"},
		{TimeMidnight, "深夜"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.time.String())
		})
	}

	t.Run("不正な値でpanicする", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			_ = TimeOfDay(99).String()
		})
	})
}

func TestGameTime_GetTimeOfDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns int
		expected   TimeOfDay
	}{
		{"ターン0は夜明け", 0, TimeDawn},
		{"ターン249は夜明け", 249, TimeDawn},
		{"ターン250は朝", 250, TimeMorning},
		{"ターン499は朝", 499, TimeMorning},
		{"ターン500は昼", 500, TimeDay},
		{"ターン749は昼", 749, TimeDay},
		{"ターン750は夕", 750, TimeEvening},
		{"ターン999は夕", 999, TimeEvening},
		{"ターン1000は夜", 1000, TimeNight},
		{"ターン1249は夜", 1249, TimeNight},
		{"ターン1250は深夜", 1250, TimeMidnight},
		{"ターン1499は深夜", 1499, TimeMidnight},
		{"ターン1500は夜明け（2日目）", 1500, TimeDawn},
		{"ターン3000は夜明け（3日目）", 3000, TimeDawn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTimeOfDay())
		})
	}
}

func TestGameTime_GetTemperatureModifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns int
		expected   int
	}{
		{"夜明けは+0°C", 0, 0},
		{"朝は+5°C", 250, 5},
		{"昼は+10°C", 500, 10},
		{"夕は+5°C", 750, 5},
		{"夜は-5°C", 1000, -5},
		{"深夜は-10°C", 1250, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetTemperatureModifier())
		})
	}
}

func TestGameTime_Advance(t *testing.T) {
	t.Parallel()

	gt := &GameTime{TotalTurns: 0}
	gt.Advance()
	assert.Equal(t, 1, gt.TotalTurns)

	gt.Advance()
	assert.Equal(t, 2, gt.TotalTurns)
}

func TestGameTime_GetDayNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalTurns int
		expected   int
	}{
		{"ターン0は1日目", 0, 1},
		{"ターン1499は1日目", 1499, 1},
		{"ターン1500は2日目", 1500, 2},
		{"ターン3000は3日目", 3000, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gt := &GameTime{TotalTurns: tt.totalTurns}
			assert.Equal(t, tt.expected, gt.GetDayNumber())
		})
	}
}
