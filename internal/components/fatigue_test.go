package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
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

func TestFatigue_Penalty(t *testing.T) {
	t.Parallel()

	t.Run("快調はペナルティ無し", func(t *testing.T) {
		t.Parallel()
		f := &Fatigue{Max: 1000, Current: 0}
		p := f.Penalty()
		assert.Equal(t, consts.Percent(0), p.RecoveryAdd)
		assert.Equal(t, 0, p.SpeedAdd)
		assert.Equal(t, consts.PercentBase, p.AccuracyMul)
	})

	t.Run("疲労は中程度のペナルティ", func(t *testing.T) {
		t.Parallel()
		f := &Fatigue{Max: 1000, Current: 600}
		p := f.Penalty()
		assert.Equal(t, consts.Percent(-20), p.RecoveryAdd)
		assert.Equal(t, -15, p.SpeedAdd)
		assert.Equal(t, consts.Percent(90), p.AccuracyMul)
	})

	t.Run("過労は重いペナルティ", func(t *testing.T) {
		t.Parallel()
		f := &Fatigue{Max: 1000, Current: 900}
		p := f.Penalty()
		assert.Equal(t, consts.Percent(-40), p.RecoveryAdd)
		assert.Equal(t, -35, p.SpeedAdd)
		assert.Equal(t, consts.Percent(75), p.AccuracyMul)
	})
}

func TestNewFatigue_初期は疲れていない(t *testing.T) {
	t.Parallel()
	f := NewFatigue()
	assert.Equal(t, DefaultMaxFatigue, f.Max)
	assert.Equal(t, 0, f.Current)
	assert.Equal(t, FatigueRested, f.GetLevel())
}
