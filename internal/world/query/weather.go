package query

import (
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetWeather はシングルトンから世界全体の天候を取得する。
func GetWeather(world w.World) *gc.Weather {
	return GetSingleton[gc.Weather](world, world.Components.Weather)
}

// placeSeverityCoef は奥地1チャンクあたり厳しい天候の重みをどれだけ伸ばすか。値は暫定。
const placeSeverityCoef = 0.05

// weatherStayWeight は遷移の粘り。同じ天候を引き継ぐ重み。1より大きいほど regime が続く。値は暫定。
const weatherStayWeight = 2.5

// stickiness は前の天候から次への遷移の粘りを返す。同一は重く、離れるほど 1/距離 で細る。段階を踏む
// 移ろいを表す。距離は WeatherKind の並び順に密結合するので、並びを変えると遷移の起きやすさも変わる。
func stickiness(current, next gc.WeatherKind) float64 {
	if current == next {
		return weatherStayWeight
	}
	d := int(current) - int(next)
	if d < 0 {
		d = -d
	}
	return 1.0 / float64(d)
}

// placeFactor は奥地ほど厳しい天候を厚くする係数を返す。severity0 の晴れ・曇り・雨は奥行きに反応せず、
// 雪・吹雪・寒波だけが northDepth に応じて伸びる。北へ進むほど荒天が主役になる。
func placeFactor(defs gc.WeatherDefs, northDepth int, next gc.WeatherKind) float64 {
	if northDepth < 0 {
		northDepth = 0
	}
	return 1.0 + float64(defs.Kinds[next].Severity)*float64(northDepth)*placeSeverityCoef
}

// NextWeather は現在の天候から次の天候を1つ引く。粘り・季節親和・場所寄せの積で重み付けする。
// 季節親和は defs の天候ごとの季節の重み。northDepth はプレイヤーの北への奥行き。奥ほど厳しい天候へ寄る。
func NextWeather(defs gc.WeatherDefs, current gc.WeatherKind, season gc.Season, northDepth int, rng *rand.Rand) gc.WeatherKind {
	var weights [gc.NumWeatherKind]float64
	for next := range gc.NumWeatherKind {
		kind := gc.WeatherKind(next)
		weights[next] = stickiness(current, kind) *
			defs.Kinds[kind].SeasonWeights[season] *
			placeFactor(defs, northDepth, kind)
	}
	i, ok := consts.Pick(rng, weights[:], func(w float64) float64 { return w })
	if !ok {
		return current
	}
	return gc.WeatherKind(i)
}

// harshSpellStretchPerChunk は奥地1チャンクあたり厳しい天候のスペルをどれだけ伸ばすか。値は暫定。
const harshSpellStretchPerChunk = 0.02

// RollSpellTurns は天候 kind のスペルが続くターン数を引く。天候種で基準長が変わり、冬・奥地では
// 厳しい天候が長引く。下限+乱数で裾を作る。
func RollSpellTurns(defs gc.WeatherDefs, kind gc.WeatherKind, season gc.Season, northDepth int, rng *rand.Rand) consts.Turn {
	if northDepth < 0 {
		northDepth = 0
	}
	def := defs.Kinds[kind]
	lo, hi := def.SpellTurnsMin, def.SpellTurnsMax
	turns := lo
	if hi > lo {
		turns += rng.IntN(hi - lo)
	}
	// 厳しい天候は冬と奥地で長引く。severity0 の晴れ・曇り・雨は伸ばさない
	if def.Severity > 0 {
		stretch := 1.0 + float64(northDepth)*harshSpellStretchPerChunk
		if season == gc.SeasonWinter {
			stretch += 0.5
		}
		turns = int(float64(turns) * stretch)
	}
	return consts.Turn(turns)
}
