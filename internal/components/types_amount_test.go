package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumeralAmount_Calc(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 42, NumeralAmount{Numeral: 42}.Calc())
	assert.Equal(t, 0, NumeralAmount{Numeral: 0}.Calc())
	assert.Equal(t, -5, NumeralAmount{Numeral: -5}.Calc())
}

func TestRatioAmount_Calc(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 50, RatioAmount{Ratio: 0.5}.Calc(100))
	assert.Equal(t, 100, RatioAmount{Ratio: 1.0}.Calc(100))
	assert.Equal(t, 0, RatioAmount{Ratio: 0.5}.Calc(0))
	assert.Equal(t, 150, RatioAmount{Ratio: 1.5}.Calc(100))
}

func TestNewTurnState(t *testing.T) {
	t.Parallel()

	ts := NewTurnState()
	assert.Equal(t, TurnPhasePlayer, ts.Phase)
	assert.Equal(t, 1, ts.TurnNumber)
}
