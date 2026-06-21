package components

import (
	"testing"

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
		want    HungerLevel
	}{
		{"満腹: 95%以上", 950, 1000, HungerSatiated},
		{"満腹: ちょうど95%", 95, 100, HungerSatiated},
		{"普通: 66%以上", 700, 1000, HungerNormal},
		{"普通: ちょうど66%", 66, 100, HungerNormal},
		{"空腹: 33%以上", 400, 1000, HungerHungry},
		{"空腹: ちょうど33%", 33, 100, HungerHungry},
		{"飢餓: 33%未満", 100, 1000, HungerStarving},
		{"飢餓: 0", 0, 1000, HungerStarving},
		{"Max=0は満腹扱い", 0, 0, HungerSatiated},
		{"Max<0は満腹扱い", 0, -1, HungerSatiated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &Hunger{Max: tt.max, Current: tt.current}
			assert.Equal(t, tt.want, h.GetLevel())
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

func TestHunger_GetStatusPenalty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		max     int
		want    int
	}{
		{"飢餓: -20", 0, 100, -20},
		{"空腹: -10", 50, 100, -10},
		{"普通: 0", 70, 100, 0},
		{"満腹: 0", 100, 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &Hunger{Max: tt.max, Current: tt.current}
			assert.Equal(t, tt.want, h.GetStatusPenalty())
		})
	}
}

func TestHungerLevel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level HungerLevel
		want  string
	}{
		{HungerSatiated, "満腹"},
		{HungerNormal, "普通"},
		{HungerHungry, "空腹"},
		{HungerStarving, "飢餓"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestHungerLevel_String_InvalidValue(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		_ = HungerLevel(99).String()
	})
}
