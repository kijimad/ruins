package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
)

// TimeOfDay は時間帯を表す
type TimeOfDay int

// 時間帯定数
const (
	TimeDawn     TimeOfDay = iota // 夜明け (0-249ターン)
	TimeMorning                   // 朝 (250-499ターン)
	TimeDay                       // 昼 (500-749ターン)
	TimeEvening                   // 夕 (750-999ターン)
	TimeNight                     // 夜 (1000-1249ターン)
	TimeMidnight                  // 深夜 (1250-1499ターン)
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

// GameTime はゲーム内時間を管理する
type GameTime struct {
	TotalTurns consts.Turn // 経過した総ターン数
}

// newGameStartTimeOfDay は新規ゲームの開始時間帯。昼から始める。
// TotalTurns は run 開始からの経過ターンで 0 始まり。暦上の時刻はこのぶん進んだ位置から始まる。
// 「昼開始」の規約はこの1点だけが持ち、時刻と日数の導出はここを介して行う。
// おかげで TotalTurns は純粋な経過ターンになり、表示はオフセットの無い値をそのまま使える。
const newGameStartTimeOfDay = TimeDay

// StartOfTimeOfDayTurns は暦上で指定した時間帯が始まるターンを返す純関数。0 は夜明けの開始。
// 新規ゲームの開始オフセットの算出にも使う。
func StartOfTimeOfDayTurns(t TimeOfDay) consts.Turn {
	return turnsPerTimeOfDay * consts.Turn(t)
}

// calendarTurn は時刻導出に使う暦上のターン。経過ターンに新規ゲームの開始時間帯ぶんを足す。
// これで TotalTurns=0 が昼を指し、時間帯と日数は従来と同じ暦位置から導ける。
func (gt *GameTime) calendarTurn() consts.Turn {
	return gt.TotalTurns + StartOfTimeOfDayTurns(newGameStartTimeOfDay)
}

// timeOfDayAt は暦上のターンから時間帯を導く純関数。0 は夜明け。
func timeOfDayAt(calendarTurn consts.Turn) TimeOfDay {
	return TimeOfDay((calendarTurn % turnsPerDay) / turnsPerTimeOfDay)
}

// dayNumberAt は暦上のターンから経過日数を導く純関数。1日目から始まる。
func dayNumberAt(calendarTurn consts.Turn) int {
	return int(calendarTurn/turnsPerDay) + 1
}

// GetTimeOfDay は現在の時間帯を返す
func (gt *GameTime) GetTimeOfDay() TimeOfDay {
	return timeOfDayAt(gt.calendarTurn())
}

// GetTemperatureModifier は時間帯による気温修正値を返す
func (gt *GameTime) GetTemperatureModifier() int {
	return temperatureModifier(gt.GetTimeOfDay())
}

// temperatureModifier は時間帯ごとの気温修正値を返す純関数。
// default を置かず全 case を列挙する。時間帯を足したら exhaustive linter がここの漏れを検知する。
func temperatureModifier(t TimeOfDay) int {
	switch t {
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
	panic(fmt.Sprintf("unknown TimeOfDay: %d", t))
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

// GetDayNumber は経過日数を返す（1日目から始まる）
func (gt *GameTime) GetDayNumber() int {
	return dayNumberAt(gt.calendarTurn())
}
