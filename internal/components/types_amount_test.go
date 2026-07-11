package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvidesHealing_Calc_Numeral(t *testing.T) {
	t.Parallel()

	// 絶対量指定はbaseを無視する
	assert.Equal(t, 42, ProvidesHealing{Kind: HealNumeral, Numeral: 42}.Calc(100))
	assert.Equal(t, 0, ProvidesHealing{Kind: HealNumeral, Numeral: 0}.Calc(100))
	assert.Equal(t, -5, ProvidesHealing{Kind: HealNumeral, Numeral: -5}.Calc(0))
}

func TestProvidesHealing_Calc_Ratio(t *testing.T) {
	t.Parallel()

	// 倍率指定はbase(最大HP)に対する割合
	assert.Equal(t, 50, ProvidesHealing{Kind: HealRatio, Ratio: 0.5}.Calc(100))
	assert.Equal(t, 100, ProvidesHealing{Kind: HealRatio, Ratio: 1.0}.Calc(100))
	assert.Equal(t, 0, ProvidesHealing{Kind: HealRatio, Ratio: 0.5}.Calc(0))
	assert.Equal(t, 150, ProvidesHealing{Kind: HealRatio, Ratio: 1.5}.Calc(100))
}

func TestNewTurnState(t *testing.T) {
	t.Parallel()

	ts := NewTurnState()
	assert.Equal(t, TurnPhasePlayer, ts.Phase)
	assert.Equal(t, 1, ts.TurnNumber)
}
