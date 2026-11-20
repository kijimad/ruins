package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTurn_String_時計アイコンを先頭に付ける(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		turn     Turn
		expected string
	}{
		{"0", 0, IconClock + " 0"},
		{"1桁", 6, IconClock + " 6"},
		{"3桁", 600, IconClock + " 600"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.turn.String())
		})
	}
}
