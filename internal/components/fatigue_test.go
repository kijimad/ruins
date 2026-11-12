package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestFatigue_GetLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		max     int
		want    rawschema.FatigueStageId
	}{
		{"快調: 30%未満", 200, 1000, rawschema.FatigueStageIdRested},
		{"快調: 0", 0, 1000, rawschema.FatigueStageIdRested},
		{"普通: 30%以上", 300, 1000, rawschema.FatigueStageIdNormal},
		{"普通: 50%直前", 499, 1000, rawschema.FatigueStageIdNormal},
		{"疲労: 50%以上", 500, 1000, rawschema.FatigueStageIdTired},
		{"疲労: 80%直前", 799, 1000, rawschema.FatigueStageIdTired},
		{"過労: 80%以上", 800, 1000, rawschema.FatigueStageIdExhausted},
		{"過労: 満タン", 1000, 1000, rawschema.FatigueStageIdExhausted},
		{"Maxが0なら快調", 0, 0, rawschema.FatigueStageIdRested},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &Fatigue{Max: tt.max, Current: tt.current}
			assert.Equal(t, tt.want, f.GetLevel(testNeedDefs))
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
			sev, ok := f.FatigueSeverity(testNeedDefs)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantSev, sev)
		})
	}
}

func TestNeedDefs_FatigueName(t *testing.T) {
	t.Parallel()

	got := map[rawschema.FatigueStageId]string{}
	for _, s := range testNeedDefs.Fatigue {
		got[s.Level] = string(testNeedDefs.FatigueName(s.Level))
	}
	assert.Equal(t, map[rawschema.FatigueStageId]string{
		rawschema.FatigueStageIdRested:    "Rested",
		rawschema.FatigueStageIdNormal:    "Normal",
		rawschema.FatigueStageIdTired:     "Tired",
		rawschema.FatigueStageIdExhausted: "Exhausted",
	}, got)
	assert.Equal(t, i18n.Msgid("unknown"), testNeedDefs.FatigueName("unknown"), "未登録は id をそのまま返す")
}

func TestNewFatigue_初期は疲れていない(t *testing.T) {
	t.Parallel()
	f := NewFatigue()
	assert.Equal(t, DefaultMaxFatigue, f.Max)
	assert.Equal(t, 0, f.Current)
	assert.Equal(t, rawschema.FatigueStageIdRested, f.GetLevel(testNeedDefs))
}
