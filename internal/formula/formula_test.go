package formula

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcHitRate(t *testing.T) {
	t.Parallel()
	// 器用度と敏捷度が同じ場合、武器命中率がそのまま反映される
	assert.Equal(t, 80, CalcHitRate(10, 10, 80))

	// 器用度が高いと命中率が上がる
	assert.Equal(t, 90, CalcHitRate(15, 10, 80))

	// 最大値でクランプされる
	assert.Equal(t, MaxHitRate, CalcHitRate(50, 10, 90))

	// 最小値でクランプされる
	assert.Equal(t, MinHitRate, CalcHitRate(1, 50, 50))
}

func TestClampHitRate(t *testing.T) {
	t.Parallel()
	assert.Equal(t, MaxHitRate, clampHitRate(100))
	assert.Equal(t, MinHitRate, clampHitRate(0))
	assert.Equal(t, 50, clampHitRate(50))
}

func TestApplyCritical(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 15, ApplyCritical(10))
	assert.Equal(t, 0, ApplyCritical(0))
	assert.Equal(t, 1, ApplyCritical(1))
}

func TestCalcHP(t *testing.T) {
	t.Parallel()
	// HP = 30 + vitality*8 + strength + sensation
	assert.Equal(t, 30+10*8+5+5, CalcHP(10, 5, 5))
	assert.Equal(t, 30, CalcHP(0, 0, 0))
}
