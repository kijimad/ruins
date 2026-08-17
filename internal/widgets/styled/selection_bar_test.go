package styled

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectionBlinkAlpha_位相0はフルアルファのピーク(t *testing.T) {
	t.Parallel()
	// 位相0をピークに取るので、単フレーム描画は常に不透明になり静止表示と一致する
	assert.Equal(t, float32(1.0), selectionBlinkAlpha(0, true))
}

func TestSelectionBlinkAlpha_animateが偽なら常にピーク(t *testing.T) {
	t.Parallel()
	for _, tick := range []int{0, 5, 15, 31, 60} {
		assert.Equal(t, float32(1.0), selectionBlinkAlpha(tick, false))
	}
}

func TestSelectionBlinkAlpha_谷は下限アルファに達する(t *testing.T) {
	t.Parallel()
	// cos が -1 に最も近づく位相は π。そのフレームで下限へ最も沈む
	tick := int(math.Round(math.Pi / selectionBlinkSpeed))
	assert.InDelta(t, selectionBlinkFloor, float64(selectionBlinkAlpha(tick, true)), 0.01)
}

func TestSelectionBlinkAlpha_値域は下限とピークに収まる(t *testing.T) {
	t.Parallel()
	for tick := range 200 {
		got := float64(selectionBlinkAlpha(tick, true))
		assert.GreaterOrEqual(t, got, selectionBlinkFloor-1e-6)
		assert.LessOrEqual(t, got, 1.0+1e-6)
	}
}
