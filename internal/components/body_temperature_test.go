package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBodyPart(t *testing.T) {
	t.Parallel()

	t.Run("String returns correct names", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			part     BodyPart
			expected string
		}{
			{BodyPartTorso, "胴体"},
			{BodyPartHead, "頭"},
			{BodyPartArms, "腕"},
			{BodyPartHands, "手"},
			{BodyPartLegs, "脚"},
			{BodyPartFeet, "足"},
		}

		for _, tt := range tests {
			t.Run(tt.expected, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.expected, tt.part.String())
			})
		}
	})

	t.Run("不正な値でpanicする", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			_ = BodyPart(99).String()
		})
	})

	t.Run("BodyPartCount is 6", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, BodyPart(6), BodyPartCount)
	})
}

func TestTempLevel(t *testing.T) {
	t.Parallel()

	t.Run("String returns correct names", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			level    TempLevel
			expected string
		}{
			{TempLevelFreezing, "凍結"},
			{TempLevelVeryCold, "非常に寒い"},
			{TempLevelCold, "寒い"},
			{TempLevelNormal, "正常"},
			{TempLevelHot, "暑い"},
			{TempLevelVeryHot, "非常に暑い"},
			{TempLevelScorching, "灼熱"},
		}

		for _, tt := range tests {
			t.Run(tt.expected, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.expected, tt.level.String())
			})
		}
	})

	t.Run("不正な値でpanicする", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			_ = TempLevel(99).String()
		})
	})
}

func TestNewBodyTemperature(t *testing.T) {
	t.Parallel()

	t.Run("初期値は全部位が正常体温", func(t *testing.T) {
		t.Parallel()
		bt := NewBodyTemperature()

		for i := 0; i < int(BodyPartCount); i++ {
			assert.Equal(t, TempNormal, bt.Parts[i].Temp, "Temp[%d]が正常体温でない", i)
			assert.Equal(t, TempNormal, bt.Parts[i].Convergent, "Convergent[%d]が正常体温でない", i)
			assert.Equal(t, 0, bt.Parts[i].FrostbiteTimer, "FrostbiteTimer[%d]が0でない", i)
			assert.False(t, bt.Parts[i].HasFrostbite, "HasFrostbite[%d]がfalseでない", i)
		}
	})
}

func TestBodyTemperature_GetLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		temp     int
		expected TempLevel
	}{
		{"凍結下限", 0, TempLevelFreezing},
		{"凍結閾値", TempFreezing, TempLevelFreezing},
		{"非常に寒い下限", TempFreezing + 1, TempLevelVeryCold},
		{"非常に寒い閾値", TempVeryCold, TempLevelVeryCold},
		{"寒い下限", TempVeryCold + 1, TempLevelCold},
		{"寒い閾値", TempCold, TempLevelCold},
		{"正常下限", TempCold + 1, TempLevelNormal},
		{"正常中央", TempNormal, TempLevelNormal},
		{"正常上限", TempHot, TempLevelNormal},
		{"暑い下限", TempHot + 1, TempLevelHot},
		{"暑い閾値", TempVeryHot, TempLevelHot},
		{"非常に暑い下限", TempVeryHot + 1, TempLevelVeryHot},
		{"非常に暑い閾値", TempScorching, TempLevelVeryHot},
		{"灼熱", TempScorching + 1, TempLevelScorching},
		{"灼熱上限", 100, TempLevelScorching},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bt := NewBodyTemperature()
			bt.Parts[BodyPartTorso].Temp = tt.temp

			level := bt.GetLevel(BodyPartTorso)
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestBodyTemperature_GetPenalty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		temp            int
		expectedPenalty int
	}{
		{"凍結時は-3", TempFreezing, -3},
		{"非常に寒い時は-2", TempVeryCold, -2},
		{"寒い時は-1", TempCold, -1},
		{"正常時は0", TempNormal, 0},
		{"暑い時は-1", TempHot + 1, -1},
		{"非常に暑い時は-2", TempVeryHot + 1, -2},
		{"灼熱時は-3", TempScorching + 1, -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bt := NewBodyTemperature()
			bt.Parts[BodyPartTorso].Temp = tt.temp

			penalty := bt.GetPenalty(BodyPartTorso)
			assert.Equal(t, tt.expectedPenalty, penalty)
		})
	}
}

func TestIsExtremity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		part     BodyPart
		expected bool
	}{
		{BodyPartTorso, false},
		{BodyPartHead, false},
		{BodyPartArms, false},
		{BodyPartHands, true},
		{BodyPartLegs, false},
		{BodyPartFeet, true},
	}

	for _, tt := range tests {
		t.Run(tt.part.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsExtremity(tt.part))
		})
	}
}

func TestClampTemp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"負の値は0になる", -10, 0},
		{"0はそのまま", 0, 0},
		{"50はそのまま", 50, 50},
		{"100はそのまま", 100, 100},
		{"100を超える値は100になる", 150, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, ClampTemp(tt.input))
		})
	}
}

func TestTemperatureConstants(t *testing.T) {
	t.Parallel()

	t.Run("温度定数が正しい順序", func(t *testing.T) {
		t.Parallel()
		assert.Less(t, TempFreezing, TempVeryCold)
		assert.Less(t, TempVeryCold, TempCold)
		assert.Less(t, TempCold, TempNormal)
		assert.Less(t, TempNormal, TempHot)
		assert.Less(t, TempHot, TempVeryHot)
		assert.Less(t, TempVeryHot, TempScorching)
	})

	t.Run("正常体温は50", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 50, TempNormal)
	})
}
