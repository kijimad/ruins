package query

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func newTestRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, 0))
}

// isHarsh は寒さの厳しい天候かを返す。分布の形を測るのに使う。
func isHarsh(defs gc.WeatherDefs, k gc.WeatherKind) bool {
	return defs.Kinds[k].Severity > 0
}

func TestNextWeather_同じシードと入力なら同じ結果(t *testing.T) {
	t.Parallel()
	defs := testutil.InitTestWorld(t).Resources.WeatherDefs
	a := NextWeather(defs, gc.WeatherClear, gc.SeasonWinter, 5, newTestRNG(42))
	b := NextWeather(defs, gc.WeatherClear, gc.SeasonWinter, 5, newTestRNG(42))
	assert.Equal(t, a, b, "決定論的で再現できる")
}

func TestNextWeather_冬は夏より厳しい天候が出やすい(t *testing.T) {
	t.Parallel()
	defs := testutil.InitTestWorld(t).Resources.WeatherDefs
	const trials = 3000
	countHarsh := func(season gc.Season) int {
		rng := newTestRNG(1)
		n := 0
		for range trials {
			// current を固定して季節だけを比べる。粘りの影響を揃えるため曇りから引く
			if isHarsh(defs, NextWeather(defs, gc.WeatherCloudy, season, 0, rng)) {
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
	defs := testutil.InitTestWorld(t).Resources.WeatherDefs
	const trials = 3000
	countHarsh := func(depth int) int {
		rng := newTestRNG(7)
		n := 0
		for range trials {
			if isHarsh(defs, NextWeather(defs, gc.WeatherCloudy, gc.SeasonAutumn, depth, rng)) {
				n++
			}
		}
		return n
	}
	deep := countHarsh(30)
	shallow := countHarsh(0)
	assert.Greater(t, deep, shallow, "北へ進むほど荒天が主役になる")
}

// TestNextWeather_全天候の重みが0なら現在の天候を維持する は、全種が季節重み0で選べない
// ときの Pick 失敗フォールバックを固定する。
func TestNextWeather_全天候の重みが0なら現在の天候を維持する(t *testing.T) {
	t.Parallel()
	var defs gc.WeatherDefs // 全種の SeasonWeights が 0 のまま

	got := NextWeather(defs, gc.WeatherCloudy, gc.SeasonWinter, 0, newTestRNG(1))

	assert.Equal(t, gc.WeatherCloudy, got, "選択できなければ現在の天候を維持する")
}

// TestPlaceFactor_負のnorthDepthは0とみなす は、北への奥行きが負のとき0として扱い、
// 奥地補正が負方向に効かないことを固定する。
func TestPlaceFactor_負のnorthDepthは0とみなす(t *testing.T) {
	t.Parallel()
	defs := testutil.InitTestWorld(t).Resources.WeatherDefs

	assert.Equal(t, placeFactor(defs, 0, gc.WeatherBlizzard), placeFactor(defs, -5, gc.WeatherBlizzard),
		"負のnorthDepthは0と同じ結果になる")
}

func TestRollSpellTurns_基準範囲に収まり厳しい天候は冬奥地で長引く(t *testing.T) {
	t.Parallel()
	defs := testutil.InitTestWorld(t).Resources.WeatherDefs
	rng := newTestRNG(3)
	// 晴れは severity0 で伸縮しないので基準範囲 2250〜4500 に収まる
	for range 100 {
		turns := int(RollSpellTurns(defs, gc.WeatherClear, gc.SeasonSpring, 20, rng))
		assert.GreaterOrEqual(t, turns, 2250)
		assert.Less(t, turns, 4500)
	}
	// 吹雪は冬奥地で夏浅部より長引く。平均で比べる
	avg := func(season gc.Season, depth int) float64 {
		rng := newTestRNG(9)
		var sum int
		const n = 500
		for range n {
			sum += int(RollSpellTurns(defs, gc.WeatherBlizzard, season, depth, rng))
		}
		return float64(sum) / n
	}
	assert.Greater(t, avg(gc.SeasonWinter, 30), avg(gc.SeasonSummer, 0), "厳しい天候は冬奥地で長引く")
}

// TestRollSpellTurns_負のnorthDepthは0とみなす は、北への奥行きが負のとき0として扱い、
// 奥地補正が負方向に効かないことを固定する。
func TestRollSpellTurns_負のnorthDepthは0とみなす(t *testing.T) {
	t.Parallel()
	defs := testutil.InitTestWorld(t).Resources.WeatherDefs

	got := RollSpellTurns(defs, gc.WeatherBlizzard, gc.SeasonWinter, -10, newTestRNG(5))
	want := RollSpellTurns(defs, gc.WeatherBlizzard, gc.SeasonWinter, 0, newTestRNG(5))

	assert.Equal(t, want, got, "負のnorthDepthは0と同じ結果になる")
}
