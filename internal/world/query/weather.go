package query

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// GetWeather はシングルトンから世界全体の天候を取得する。
func GetWeather(world w.World) *gc.Weather {
	return GetSingleton[gc.Weather](world, world.Components.Weather)
}

// seasonAffinity は季節ごとの各天候の出やすさ。冬は雪・吹雪・寒波を厚く、夏はほぼ晴れ・曇り・雨。
// 行は gc.Season の並び 春・夏・秋・冬、列は WeatherKind の並び 晴・曇・雨・雪・吹雪・寒波。値は暫定。
var seasonAffinity = [4][gc.NumWeatherKind]float64{
	{1.4, 1.2, 0.9, 0.8, 0.3, 0.1}, // 春
	{1.8, 1.0, 1.1, 0.3, 0.1, 0.0}, // 夏
	{1.1, 1.2, 0.8, 1.0, 0.5, 0.2}, // 秋
	{0.6, 0.9, 0.2, 1.3, 1.2, 0.8}, // 冬
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
func placeFactor(northDepth int, next gc.WeatherKind) float64 {
	if northDepth < 0 {
		northDepth = 0
	}
	return 1.0 + float64(next.Severity())*float64(northDepth)*placeSeverityCoef
}

// NextWeather は現在の天候から次の天候を1つ引く。粘り・季節親和・場所寄せの積で重み付けする。
// northDepth はプレイヤーの北への奥行き。奥ほど厳しい天候へ寄る。
func NextWeather(current gc.WeatherKind, season gc.Season, northDepth int, rng *rand.Rand) gc.WeatherKind {
	var weights [gc.NumWeatherKind]float64
	for next := range gc.NumWeatherKind {
		kind := gc.WeatherKind(next)
		weights[next] = stickiness(current, kind) *
			seasonAffinity[season][next] *
			placeFactor(northDepth, kind)
	}
	return gc.WeatherKind(drawWeighted(weights[:], rng))
}

// drawWeighted は重み配列から1つの添字を確率抽選する。総和が0以下なら0を返す安全側の縮退。
// 丸め誤差で末尾まで抜けた場合は、末尾でなく最大重みの添字へ縮退する。末尾に落とすと微小重みの
// 天候が過剰に選ばれ分布の意図とずれるため。
func drawWeighted(weights []float64, rng *rand.Rand) int {
	var total float64
	// maxIdx は最大重みの添字。丸め誤差での縮退先に使う
	maxIdx := 0
	for i, w := range weights {
		total += w
		if w > weights[maxIdx] {
			maxIdx = i
		}
	}
	if total <= 0 {
		return 0
	}
	r := rng.Float64() * total
	for i, w := range weights {
		r -= w
		if r < 0 {
			return i
		}
	}
	return maxIdx
}

// spellTurnRange は天候種ごとのスペル基準長の下限と上限を返す。晴れは長く吹雪は短い。値は暫定。
// TurnsPerDay=1500 が1日。default を置かず exhaustive に全種を強制し、追加漏れを panic と linter で止める。
func spellTurnRange(kind gc.WeatherKind) (lo, hi int) {
	switch kind {
	case gc.WeatherClear:
		return 2250, 4500 // 1.5〜3日
	case gc.WeatherCloudy:
		return 1500, 3000 // 1〜2日
	case gc.WeatherRain:
		return 750, 2250 // 0.5〜1.5日
	case gc.WeatherSnow:
		return 1200, 2250 // 0.8〜1.5日
	case gc.WeatherBlizzard:
		return 450, 900 // 0.3〜0.6日
	case gc.WeatherColdSnap:
		return 1500, 3000 // 1〜2日
	}
	panic(fmt.Sprintf("unknown WeatherKind: %d", kind))
}

// harshSpellStretchPerChunk は奥地1チャンクあたり厳しい天候のスペルをどれだけ伸ばすか。値は暫定。
const harshSpellStretchPerChunk = 0.02

// RollSpellTurns は天候 kind のスペルが続くターン数を引く。天候種で基準長が変わり、冬・奥地では
// 厳しい天候が長引く。下限+乱数で裾を作る。
func RollSpellTurns(kind gc.WeatherKind, season gc.Season, northDepth int, rng *rand.Rand) consts.Turn {
	if northDepth < 0 {
		northDepth = 0
	}
	lo, hi := spellTurnRange(kind)
	turns := lo
	if hi > lo {
		turns += rng.IntN(hi - lo)
	}
	// 厳しい天候は冬と奥地で長引く。severity0 の晴れ・曇り・雨は伸ばさない
	if kind.Severity() > 0 {
		stretch := 1.0 + float64(northDepth)*harshSpellStretchPerChunk
		if season == gc.SeasonWinter {
			stretch += 0.5
		}
		turns = int(float64(turns) * stretch)
	}
	return consts.Turn(turns)
}
