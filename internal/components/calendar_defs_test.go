package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCalendarDefs_SeasonalTemperatureForDay は経過日数に対応する季節の世界温度ベースを確認する
func TestCalendarDefs_SeasonalTemperatureForDay(t *testing.T) {
	t.Parallel()

	var defs CalendarDefs
	defs.Seasons[SeasonSpring].Temp = 5
	defs.Seasons[SeasonSummer].Temp = 22
	defs.Seasons[SeasonAutumn].Temp = 0
	defs.Seasons[SeasonWinter].Temp = -30
	tests := []struct {
		name     string
		day      int
		expected int
	}{
		{"1日目は春の中点", 1, 5},
		{"5日目は春の坂の途中", 5, 17},
		{"9日目は夏のピーク", 9, 22},
		{"13日目は秋へ下る坂の途中。春側の同位置より寒い", 13, 16},
		{"17日目は秋の中点。春より寒い", 17, 0},
		{"21日目は冬へ向かう坂の途中", 21, -21},
		{"25日目は冬の底", 25, -30},
		{"29日目は春へ戻る坂の途中", 29, -20},
		{"33日目は翌年の春の中点", 33, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, defs.SeasonalTemperatureForDay(tt.day))
		})
	}
}
