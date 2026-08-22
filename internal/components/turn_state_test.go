package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTurnPhase_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tp   TurnPhase
		want string
	}{
		{TurnPhasePlayer, "PlayerTurn"},
		{TurnPhaseAI, "AITurn"},
		{TurnPhaseEnd, "TurnEnd"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.tp.String())
		})
	}
}

func TestTurnPhase_String_InvalidPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_ = TurnPhase(99).String()
	}, "不正なTurnPhase値でpanicする")
}
