package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeat_String_炎アイコンを先頭に付ける(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		heat     Heat
		expected string
	}{
		{"0", 0, IconFire + " 0"},
		{"1桁", 6, IconFire + " 6"},
		{"3桁", 600, IconFire + " 600"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.heat.String())
		})
	}
}

func TestHeat_BurnTurns_効率で割り引いた燃焼ターン数を返す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		heat       Heat
		efficiency int
		expected   int
	}{
		{"効率100%は等倍", 20, 100, 20},
		{"効率50%は半分", 20, 50, 10},
		{"端数は切り捨て", 15, 50, 7},
		{"効率0%は0", 20, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.heat.BurnTurns(tt.efficiency))
		})
	}
}
