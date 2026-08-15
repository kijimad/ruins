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

// StartOfTimeOfDayTurns は指定した時間帯が始まる総ターン数を返す。
// 新規ゲームの開始時刻を特定の時間帯へ合わせるのに使う。
func StartOfTimeOfDayTurns(t TimeOfDay) consts.Turn {
	return turnsPerTimeOfDay * consts.Turn(t)
}

// GetTimeOfDay は現在の時間帯を返す
func (gt *GameTime) GetTimeOfDay() TimeOfDay {
	turnInDay := gt.TotalTurns % turnsPerDay
	return TimeOfDay(turnInDay / turnsPerTimeOfDay)
}

// GetTemperatureModifier は時間帯による気温修正値を返す。
// default を置かず全 case を列挙する。時間帯を足したら exhaustive linter がここの漏れを検知する。
func (gt *GameTime) GetTemperatureModifier() int {
	switch gt.GetTimeOfDay() {
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
	panic(fmt.Sprintf("unknown TimeOfDay: %d", gt.GetTimeOfDay()))
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
	return int(gt.TotalTurns/turnsPerDay) + 1
}
