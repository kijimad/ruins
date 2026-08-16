package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCurrency_String_カンマ区切りと通貨記号で整形する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   Currency
		expected string
	}{
		{"0", 0, "󱪯 0"},
		{"1桁", 5, "󱪯 5"},
		{"2桁", 99, "󱪯 99"},
		{"3桁", 999, "󱪯 999"},
		{"4桁でカンマ1つ", 1000, "󱪯 1,000"},
		{"5桁", 12345, "󱪯 12,345"},
		{"6桁", 100204, "󱪯 100,204"},
		{"7桁", 1234567, "󱪯 1,234,567"},
		{"8桁", 10000000, "󱪯 10,000,000"},
		{"負の数", -1234, "󱪯 -1,234"},
		{"負の数で大きい", -1234567, "󱪯 -1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.amount.String())
		})
	}
}
