package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestNewHunger(t *testing.T) {
	t.Parallel()

	h := NewHunger()
	assert.Equal(t, DefaultMaxHunger, h.Max)
	assert.Equal(t, DefaultInitialHunger, h.Current)
}

func TestHunger_GetLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		max     int
		want    rawschema.HungerStageId
	}{
		{"満腹: 95%以上", 950, 1000, rawschema.HungerStageIdSatiated},
		{"満腹: ちょうど95%", 95, 100, rawschema.HungerStageIdSatiated},
		{"普通: 66%以上", 700, 1000, rawschema.HungerStageIdNormal},
		{"普通: ちょうど66%", 66, 100, rawschema.HungerStageIdNormal},
		{"空腹: 33%以上", 400, 1000, rawschema.HungerStageIdHungry},
		{"空腹: ちょうど33%", 33, 100, rawschema.HungerStageIdHungry},
		{"飢餓: 33%未満", 100, 1000, rawschema.HungerStageIdStarving},
		{"飢餓: 0", 0, 1000, rawschema.HungerStageIdStarving},
		{"Max=0は満腹扱い", 0, 0, rawschema.HungerStageIdSatiated},
		{"Max<0は満腹扱い", 0, -1, rawschema.HungerStageIdSatiated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &Hunger{Max: tt.max, Current: tt.current}
			assert.Equal(t, tt.want, h.GetLevel(testNeedDefs))
		})
	}
}

func TestHunger_Increase(t *testing.T) {
	t.Parallel()

	t.Run("通常の増加", func(t *testing.T) {
		t.Parallel()
		h := &Hunger{Max: 100, Current: 50}
		h.Increase(30)
		assert.Equal(t, 80, h.Current)
	})

	t.Run("Maxを超えない", func(t *testing.T) {
		t.Parallel()
		h := &Hunger{Max: 100, Current: 80}
		h.Increase(50)
		assert.Equal(t, 100, h.Current)
	})

	t.Run("負の値を加算しても0未満にならない", func(t *testing.T) {
		t.Parallel()
		h := &Hunger{Max: 100, Current: 10}
		h.Increase(-20)
		assert.Equal(t, 0, h.Current)
	})
}

func TestHunger_Decrease(t *testing.T) {
	t.Parallel()

	t.Run("通常の減少", func(t *testing.T) {
		t.Parallel()
		h := &Hunger{Max: 100, Current: 50}
		h.Decrease(30)
		assert.Equal(t, 20, h.Current)
	})

	t.Run("0未満にならない", func(t *testing.T) {
		t.Parallel()
		h := &Hunger{Max: 100, Current: 10}
		h.Decrease(20)
		assert.Equal(t, 0, h.Current)
	})
}

func TestNeedDefs_HungerName(t *testing.T) {
	t.Parallel()

	got := map[rawschema.HungerStageId]string{}
	for _, s := range testNeedDefs.Hunger {
		got[s.Level] = string(testNeedDefs.HungerName(s.Level))
	}
	assert.Equal(t, map[rawschema.HungerStageId]string{
		rawschema.HungerStageIdSatiated: "Full",
		rawschema.HungerStageIdNormal:   "Normal",
		rawschema.HungerStageIdHungry:   "Hungry",
		rawschema.HungerStageIdStarving: "Starving",
	}, got)
	assert.Equal(t, i18n.Msgid("unknown"), testNeedDefs.HungerName("unknown"), "未登録は id をそのまま返す")
}

func TestNeedDefsHungerSeverity_段階ごとに不調の重症度を返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		level   rawschema.HungerStageId
		wantSev Severity
		wantOK  bool
	}{
		{"満腹は不調なし", rawschema.HungerStageIdSatiated, SeverityNone, false},
		{"普通は不調なし", rawschema.HungerStageIdNormal, SeverityNone, false},
		{"空腹は軽度の栄養失調", rawschema.HungerStageIdHungry, SeverityMinor, true},
		{"飢餓は中度の栄養失調", rawschema.HungerStageIdStarving, SeverityMedium, true},
		{"未知の段階は不調なし", rawschema.HungerStageId("unknown"), SeverityNone, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sev, ok := testNeedDefs.HungerSeverity(tt.level)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantSev, sev)
		})
	}
}

func TestNeedDefs_段階の始まりの割合(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.33, testNeedDefs.HungerFrom(rawschema.HungerStageIdHungry), 1e-9)
	assert.InDelta(t, 0.5, testNeedDefs.FatigueFrom(rawschema.FatigueStageIdTired), 1e-9)
	assert.InDelta(t, 0.8, testNeedDefs.FatigueFrom(rawschema.FatigueStageIdExhausted), 1e-9)
}

func TestNeedDefs_表が空なら満腹と快調(t *testing.T) {
	t.Parallel()

	h := &Hunger{Max: 100, Current: 0}
	f := &Fatigue{Max: 100, Current: 100}
	assert.Equal(t, rawschema.HungerStageIdSatiated, h.GetLevel(NeedDefs{}))
	assert.Equal(t, rawschema.FatigueStageIdRested, f.GetLevel(NeedDefs{}))
}
