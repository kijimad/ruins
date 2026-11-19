package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDaysUntilHungerEmpty_手計算と一致(t *testing.T) {
	t.Parallel()
	// 満腹500、減耗1/3/ターン、1500ターン/日 → 500 / (1/3) / 1500 = 1.0 日。
	assert.InDelta(t, 1.0, DaysUntilHungerEmpty(DefaultParams(loadTestMaster(t))), 1e-9)
}

func TestDaysUntilStarving_手計算と一致(t *testing.T) {
	t.Parallel()
	// 33%を割るまでに失う量は 500×0.67=335。335 / (1/3) / 1500 = 0.67 日。
	assert.InDelta(t, 0.67, DaysUntilStarving(DefaultParams(loadTestMaster(t))), 0.01)
}

func TestTurnsToHypothermia_温度帯ごとに一致(t *testing.T) {
	t.Parallel()
	// 低体温帯2.0 / |冷却率|。冷却率は適温10℃を下回った1℃ごとに0.002。
	// -50℃は0.12で約16.7ターン、0℃は0.02で100ターン、5℃は0.01で200ターン、適温は冷えないので0。
	assert.InDelta(t, 2.0/0.12, TurnsToHypothermia(-50), 1e-9)
	assert.InDelta(t, 100, TurnsToHypothermia(0), 1e-9)
	assert.InDelta(t, 200, TurnsToHypothermia(5), 1e-9)
	assert.Equal(t, 0.0, TurnsToHypothermia(10))
}

func TestHPDrainPerTurnAtBlood_崖と段階(t *testing.T) {
	t.Parallel()
	// 危険域40以上は0。40未満で ceil((40-blood)/10) 段階的に増える
	assert.Equal(t, 0, HPDrainPerTurnAtBlood(40))
	assert.Equal(t, 1, HPDrainPerTurnAtBlood(39))
	assert.Equal(t, 1, HPDrainPerTurnAtBlood(30))
	assert.Equal(t, 2, HPDrainPerTurnAtBlood(20))
	assert.Equal(t, 4, HPDrainPerTurnAtBlood(0))
}

func TestDaysUntilTired_手計算と一致(t *testing.T) {
	t.Parallel()
	// 疲労0.5 × 最大2000 / 1per turn / 1500per日 = 0.666...日
	assert.InDelta(t, 0.667, DaysUntilTired(DefaultParams(loadTestMaster(t))), 0.01)
	// 過労 0.8 × 2000 / 1 / 1500 = 1.066...日
	assert.InDelta(t, 1.067, DaysUntilExhausted(DefaultParams(loadTestMaster(t))), 0.01)
}

func TestSleepTurnsToFullRecover_手計算と一致(t *testing.T) {
	t.Parallel()
	// 最大2000 / 回復3per turn = 666.6...ターン。ゲームは整数回復で実際は切り上げ667ターン。
	assert.InDelta(t, 666.667, SleepTurnsToFullRecover(DefaultParams(loadTestMaster(t))), 0.01)
}

func TestSleepTimeFraction_手計算と一致(t *testing.T) {
	t.Parallel()
	// 起床蓄積1、睡眠回復3なので、釣り合いに要する睡眠時間比は 1/(1+3)=0.25。
	assert.InDelta(t, 0.25, SleepTimeFraction(DefaultParams(loadTestMaster(t))), 1e-9)
}
