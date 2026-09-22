package query

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

func newTestRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0))
}

// isHarsh は寒さの厳しい天候かを返す。分布の形を測るのに使う。
func isHarsh(k gc.WeatherKind) bool {
	return k.Severity() > 0
}

func TestNextWeather_同じシードと入力なら同じ結果(t *testing.T) {
	t.Parallel()
	a := NextWeather(gc.WeatherClear, gc.SeasonWinter, 5, newTestRNG(42))
	b := NextWeather(gc.WeatherClear, gc.SeasonWinter, 5, newTestRNG(42))
	assert.Equal(t, a, b, "決定論的で再現できる")
}

func TestNextWeather_冬は夏より厳しい天候が出やすい(t *testing.T) {
	t.Parallel()
	const trials = 3000
	countHarsh := func(season gc.Season) int {
		rng := newTestRNG(1)
		n := 0
		for range trials {
			// current を固定して季節だけを比べる。粘りの影響を揃えるため曇りから引く
			if isHarsh(NextWeather(gc.WeatherCloudy, season, 0, rng)) {
				n++
			}
		}
		return n
	}
	winter := countHarsh(gc.SeasonWinter)
	summer := countHarsh(gc.SeasonSummer)
	assert.Greater(t, winter, summer, "冬は夏より雪・吹雪・寒波が出やすい")
}

func TestNextWeather_奥地ほど厳しい天候が出やすい(t *testing.T) {
	t.Parallel()
	const trials = 3000
	countHarsh := func(depth int) int {
		rng := newTestRNG(7)
		n := 0
		for range trials {
			if isHarsh(NextWeather(gc.WeatherCloudy, gc.SeasonAutumn, depth, rng)) {
				n++
			}
		}
		return n
	}
	deep := countHarsh(30)
	shallow := countHarsh(0)
	assert.Greater(t, deep, shallow, "北へ進むほど荒天が主役になる")
}

func TestRollSpellTurns_基準範囲に収まり厳しい天候は冬奥地で長引く(t *testing.T) {
	t.Parallel()
	rng := newTestRNG(3)
	// 晴れは severity0 で伸縮しないので基準範囲 2250〜4500 に収まる
	for range 100 {
		turns := int(RollSpellTurns(gc.WeatherClear, gc.SeasonSpring, 20, rng))
		assert.GreaterOrEqual(t, turns, 2250)
		assert.Less(t, turns, 4500)
	}
	// 吹雪は冬奥地で夏浅部より長引く。平均で比べる
	avg := func(season gc.Season, depth int) float64 {
		rng := newTestRNG(9)
		var sum int
		const n = 500
		for range n {
			sum += int(RollSpellTurns(gc.WeatherBlizzard, season, depth, rng))
		}
		return float64(sum) / n
	}
	assert.Greater(t, avg(gc.SeasonWinter, 30), avg(gc.SeasonSummer, 0), "厳しい天候は冬奥地で長引く")
}

func TestDrawWeighted_総和0は先頭へ縮退する(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, drawWeighted([]float64{0, 0, 0}, newTestRNG(1)), "全ゼロは先頭で安全側に倒す")
}

func TestDrawWeighted_重みに比例して引く(t *testing.T) {
	t.Parallel()
	rng := newTestRNG(2)
	// 添字1だけに重みを与えると必ず1が出る
	for range 50 {
		assert.Equal(t, 1, drawWeighted([]float64{0, 5, 0}, rng))
	}
}
