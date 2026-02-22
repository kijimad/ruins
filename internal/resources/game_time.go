package resources

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
		return "夜明け"
	case TimeMorning:
		return "朝"
	case TimeDay:
		return "昼"
	case TimeEvening:
		return "夕"
	case TimeNight:
		return "夜"
	case TimeMidnight:
		return "深夜"
	default:
		return "不明"
	}
}

// 1日のターン数
const turnsPerDay = 1500

// 時間帯ごとのターン数
const turnsPerTimeOfDay = turnsPerDay / 6 // 250ターン

// GameTime はゲーム内時間を管理する
type GameTime struct {
	TotalTurns int // 経過した総ターン数
}

// GetTimeOfDay は現在の時間帯を返す
func (gt *GameTime) GetTimeOfDay() TimeOfDay {
	turnInDay := gt.TotalTurns % turnsPerDay
	return TimeOfDay(turnInDay / turnsPerTimeOfDay)
}

// GetTemperatureModifier は時間帯による気温修正値を返す
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
	default:
		return 0
	}
}

// Advance はターンを進める
func (gt *GameTime) Advance() {
	gt.TotalTurns++
}

// GetDayNumber は経過日数を返す（1日目から始まる）
func (gt *GameTime) GetDayNumber() int {
	return gt.TotalTurns/turnsPerDay + 1
}
