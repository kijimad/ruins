package components

import (
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

// WeatherKind は天候の種類。降水の重さと表示の順で並べる。効果と寒さの厳しさは WeatherDefs が持つ。
// 並び順は query の stickiness の距離計算に効くので、種を足すときは並びの影響に注意する。
// enum は基本 string の規約から外れ iota の int にする。Season・TimeOfDay と同じ順序数 enum で、
// WeatherDefs の配列添字や NumWeatherKind に整数値を直に使うため。
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
// VisionDelta は視程半径 VisionRadiusTiles=40 に対するタイル単位の増減。
// 負で寒く・狭くなる。雨・雪の水濡れによる寒さ増幅は将来項で、今は温度と視界の2つだけ持つ。
type WeatherEffect struct {
	TempModifier int // ℃。世界温度への加算
	VisionDelta  int // 視程レンジへの加算。負で狭くなる。加算側で下限の床を打つ
}
