package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTurnPhase_String_InvalidPanic(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		_ = TurnPhase(99).String()
	}, "不正なTurnPhase値でpanicする")
}
