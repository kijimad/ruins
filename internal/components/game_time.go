package components

import (
	"fmt"
	"math"

	"github.com/kijimaD/ruins/internal/consts"
)

// TimeOfDay は時間帯を表す
type TimeOfDay int

// 時間帯定数。新規ゲームは昼から始まるので、経過ターンの区切り順に昼から並べる。
// これで TotalTurns=0 が昼を無変換で指す。各時間帯は turnsPerTimeOfDay ずつ続き、
// 深夜の次に翌日の夜明けへ折り返す。かっこ内は1周目の経過ターン範囲で、以降は1日ごとに繰り返す
const (
	TimeDay      TimeOfDay = iota // 昼 経過0-249
	TimeEvening                   // 夕 経過250-499
	TimeNight                     // 夜 経過500-749
	TimeMidnight                  // 深夜 経過750-999
	TimeDawn                      // 夜明け 経過1000-1249
	TimeMorning                   // 朝 経過1250-1499
)

// String は時間帯名を返す
func (t TimeOfDay) String() string {
	switch t {
	case TimeDawn:
		return "Dawn"
	case TimeMorning:
		return "Morning"
	case TimeDay:
		return "Noon"
	case TimeEvening:
		return "Evening"
	case TimeNight:
		return "Night"
	case TimeMidnight:
		return "Midnight"
	default:
		panic("invalid TimeOfDay value")
	}
}

// 1日のターン数
const turnsPerDay consts.Turn = 1500

// 時間帯ごとのターン数
const turnsPerTimeOfDay consts.Turn = turnsPerDay / 6 // 250ターン

// noonOffsetFromDawn は暦の1日、夜明け始まり、における昼の位置。夜明けから朝を経て昼までの2区切り。
// 新規ゲームは昼から始まるので、暦では1日目のこのぶんが既に過ぎている。
// 時間帯は昼始まりの定数順で無変換に導けるが、日数は暦の夜明けで繰り上がるため、ここだけオフセットが要る。
const noonOffsetFromDawn consts.Turn = 2 * turnsPerTimeOfDay // 500ターン

// GameTime はゲーム内時間を管理する
type GameTime struct {
	TotalTurns consts.Turn // run 開始からの経過ターン数
}

// GetTimeOfDay は現在の時間帯を返す
func (gt *GameTime) GetTimeOfDay() TimeOfDay {
	return TimeOfDay(gt.TotalTurns % turnsPerDay / turnsPerTimeOfDay)
}

// GetDayNumber は経過日数を返す（1日目から始まる）。日付は暦の夜明けで繰り上がる
func (gt *GameTime) GetDayNumber() int {
	return int((gt.TotalTurns+noonOffsetFromDawn)/turnsPerDay) + 1
}

// 季節による世界温度のパラメータ。夏ピークと冬底を持つ1年周期で、値は実プレイで調整する。
const (
	daysPerYear      = 32  // 季節1周の日数
	summerPeakTemp   = 22  // 夏ピークの世界温度
	winterTroughTemp = -30 // 冬底の世界温度。準備なしでは生存できない寒さ
)

// GetSeasonalTemperature は経過日数から季節による世界温度のベース値を返す。
// 春開始の正弦波で、春秋が中点、夏がピーク、冬が底になる。季節は保存せず日数から導く。
func (gt *GameTime) GetSeasonalTemperature() int {
	mid := float64(summerPeakTemp+winterTroughTemp) / 2
	amp := float64(summerPeakTemp-winterTroughTemp) / 2
	// day 1 を春の中点かつ上昇位相の起点にする
	phase := 2 * math.Pi * float64(gt.GetDayNumber()-1) / daysPerYear
	return int(math.Round(mid + amp*math.Sin(phase)))
}

// Season は季節を表す。1年を4等分し、春開始で巡る。
type Season int

// 季節定数。GetSeasonalTemperature の正弦波と位相を合わせる。
const (
	SeasonSpring Season = iota // 春
	SeasonSummer               // 夏
	SeasonAutumn               // 秋
	SeasonWinter               // 冬
)

// String は季節名を返す。表示側が i18n の訳を引く msgid になる。
func (s Season) String() string {
	switch s {
	case SeasonSpring:
		return "Spring"
	case SeasonSummer:
		return "Summer"
	case SeasonAutumn:
		return "Autumn"
	case SeasonWinter:
		return "Winter"
	}
	panic(fmt.Sprintf("unknown Season: %d", s))
}

// GetSeason は経過日数から現在の季節を返す。
func (gt *GameTime) GetSeason() Season {
	const daysPerSeason = daysPerYear / 4
	dayOfYear := (gt.GetDayNumber() - 1) % daysPerYear
	return Season(dayOfYear / daysPerSeason)
}

// SeasonJustChanged は直前のターンから季節が変わったかを返す。Advance の直後に呼ぶ想定。
// 季節は TotalTurns から導くので、現在と1つ前の導出値を比べれば判定でき、前回値を保持する状態はいらない。
func (gt *GameTime) SeasonJustChanged() bool {
	if gt.TotalTurns == 0 {
		return false
	}
	prev := GameTime{TotalTurns: gt.TotalTurns - 1}
	return gt.GetSeason() != prev.GetSeason()
}

// TimeOfDayJustChanged は現在の時間帯と、直前のターンから時間帯が変わったかを返す。
// どの時間帯へ入ったかで日の出入りを見分けられるよう、変わった先の時間帯も返す。
func (gt *GameTime) TimeOfDayJustChanged() (TimeOfDay, bool) {
	cur := gt.GetTimeOfDay()
	if gt.TotalTurns == 0 {
		return cur, false
	}
	prev := GameTime{TotalTurns: gt.TotalTurns - 1}
	return cur, prev.GetTimeOfDay() != cur
}

// GetTemperatureModifier は時間帯による気温修正値を返す。
// default を置かず全 case を列挙する。時間帯を足したら exhaustive linter がここの漏れを検知する。
func (gt *GameTime) GetTemperatureModifier() int {
	tod := gt.GetTimeOfDay()
	switch tod {
	case TimeDawn:
		return 0 // 夜明け: +0°C
	case TimeMorning:
		return 5 // 朝: +5°C
	case TimeDay:
		return 10 // 昼: +10°C
	case TimeEvening:
		return 5 // 夕: +5°C
	case TimeNight:
		return -5 // 夜: -5°C
	case TimeMidnight:
		return -10 // 深夜: -10°C
	}
	panic(fmt.Sprintf("unknown TimeOfDay: %d", tod))
}

// Advance はターンを進める
func (gt *GameTime) Advance() {
	gt.TotalTurns++
}

// AdvanceToNextTimeOfDay は次の時間帯の開始ターンまで進める。
// 現在のフェーズの残りターンを飛ばし、常にちょうど1つ次の時間帯にする。
// 深夜からは翌日の夜明けへ折り返す。
func (gt *GameTime) AdvanceToNextTimeOfDay() {
	gt.TotalTurns = (gt.TotalTurns/turnsPerTimeOfDay + 1) * turnsPerTimeOfDay
}

// turnsPerSeason は1季節ぶんのターン数
const turnsPerSeason consts.Turn = consts.Turn(daysPerYear/4) * turnsPerDay

// AdvanceToNextSeason は次の季節の開始まで進める。季節による世界温度を切り替える。
// 冬からは翌年の春へ折り返す
func (gt *GameTime) AdvanceToNextSeason() {
	gt.TotalTurns = (gt.TotalTurns/turnsPerSeason + 1) * turnsPerSeason
}
