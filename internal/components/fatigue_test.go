package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFatigue_GetLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		max     int
		want    FatigueLevel
	}{
		{"快調: 30%未満", 200, 1000, FatigueRested},
		{"快調: 0", 0, 1000, FatigueRested},
		{"普通: 30%以上", 300, 1000, FatigueNormal},
		{"普通: 50%直前", 499, 1000, FatigueNormal},
		{"疲労: 50%以上", 500, 1000, FatigueTired},
		{"疲労: 80%直前", 799, 1000, FatigueTired},
		{"過労: 80%以上", 800, 1000, FatigueExhausted},
		{"過労: 満タン", 1000, 1000, FatigueExhausted},
		{"Maxが0なら快調", 0, 0, FatigueRested},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &Fatigue{Max: tt.max, Current: tt.current}
			assert.Equal(t, tt.want, f.GetLevel())
		})
	}
}

func TestFatigue_FatigueSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		wantSev Severity
		wantOK  bool
	}{
		{"快調は過労なし", 0, SeverityNone, false},
		{"疲労は軽度の過労", 600, SeverityMinor, true},
		{"過労は中度の過労", 900, SeverityMedium, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &Fatigue{Max: 1000, Current: tt.current}
			sev, ok := f.FatigueSeverity()
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantSev, sev)
		})
	}
}

func TestNewFatigue_初期は疲れていない(t *testing.T) {
	t.Parallel()
	f := NewFatigue()
	assert.Equal(t, DefaultMaxFatigue, f.Max)
	assert.Equal(t, 0, f.Current)
	assert.Equal(t, FatigueRested, f.GetLevel())
}
