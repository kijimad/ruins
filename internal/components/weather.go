package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
)

// Weather は世界全体の天候を保持するシングルトン。遷移の記憶と持続の残りは日数から導けないので保存する。
type Weather struct {
	Current   WeatherKind
	UntilTurn consts.Turn // このスペルが続く終端の総ターン数。TotalTurns が到達したら次のスペルへ遷移する
}

// InitialWeatherSpellTurns は run 開始の天候スペルの長さ。中立の曇りをこの長さだけ続けてから遷移を
// 始める。turn0 で即ロールすると開始直後に荒天になりうるので、穏やかな立ち上がりにする。
const InitialWeatherSpellTurns consts.Turn = 2500

// WeatherKind は天候の種類。降水の重さと表示の順で並べる。寒さの厳しさは Severity で別に持つ。
// 並び順は query の stickiness の距離計算に効くので、種を足すときは並びの影響に注意する。
// enum は基本 string の規約から外れ iota の int にする。Season・TimeOfDay と同じ順序数 enum で、
// seasonAffinity の行列添字や NumWeatherKind に整数値を直に使うため。
type WeatherKind int

const (
	// WeatherClear は晴れ。わずかに暖かく、視界は削られない
	WeatherClear WeatherKind = iota
	// WeatherCloudy は曇り。中立
	WeatherCloudy
	// WeatherRain は雨。氷点下でない降水。やや冷えて視程が落ちる。将来は水濡れを付ける
	WeatherRain
	// WeatherSnow は雪。やや寒く視程がやや落ちる
	WeatherSnow
	// WeatherBlizzard は吹雪。強い寒さと大幅な視程低下
	WeatherBlizzard
	// WeatherColdSnap は寒波。晴れているが酷寒。視界は保たれる別種の厳しさ
	WeatherColdSnap
	// NumWeatherKind は天候種の数。遷移の重み配列の長さに使う
	NumWeatherKind = iota
)

// WeatherEffect は天候1種の効果。気温補正は世界温度へ加算し、視界増減は視程レンジへ加算する。
// 負で寒く・狭くなる。雨・雪の水濡れによる寒さ増幅は将来項で、今は温度と視界の2つだけ持つ。
type WeatherEffect struct {
	TempModifier int // ℃。世界温度への加算
	VisionDelta  int // 視程レンジへの加算。負で狭くなる。加算側で下限の床を打つ
}

// Effect は天候種の効果を返す。値は暫定。VisionDelta は視程半径 VisionRadiusTiles=40 に対するタイル単位の
// 増減。default を置かず exhaustive に全種を強制し、追加漏れを panic と linter で止める。
func (k WeatherKind) Effect() WeatherEffect {
	switch k {
	case WeatherClear:
		return WeatherEffect{TempModifier: 2, VisionDelta: 0}
	case WeatherCloudy:
		return WeatherEffect{TempModifier: 0, VisionDelta: 0}
	case WeatherRain:
		return WeatherEffect{TempModifier: -2, VisionDelta: -6}
	case WeatherSnow:
		return WeatherEffect{TempModifier: -3, VisionDelta: -10}
	case WeatherBlizzard:
		return WeatherEffect{TempModifier: -8, VisionDelta: -28}
	case WeatherColdSnap:
		return WeatherEffect{TempModifier: -15, VisionDelta: 0}
	}
	panic(fmt.Sprintf("unknown WeatherKind: %d", k))
}

// Severity は寒さの厳しさを返す。placeFactor が奥地ほど厳しい天候を厚くするのに使う。雨は降水だが
// 寒さ軸では軽いので0にする。
func (k WeatherKind) Severity() int {
	switch k {
	case WeatherClear, WeatherCloudy, WeatherRain:
		return 0
	case WeatherSnow:
		return 1
	case WeatherBlizzard, WeatherColdSnap:
		return 2
	}
	panic(fmt.Sprintf("unknown WeatherKind: %d", k))
}

// String は天候名を返す。表示側が i18n の訳を引く msgid になる。Season.String に倣う。
func (k WeatherKind) String() string {
	switch k {
	case WeatherClear:
		return "Clear"
	case WeatherCloudy:
		return "Cloudy"
	case WeatherRain:
		return "Rain"
	case WeatherSnow:
		return "Snow"
	case WeatherBlizzard:
		return "Blizzard"
	case WeatherColdSnap:
		return "Cold snap"
	}
	panic(fmt.Sprintf("unknown WeatherKind: %d", k))
}
