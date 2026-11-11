package components

import (
	"github.com/kijimaD/ruins/internal/consts"
)

// TimeOfDay は時間帯を表す
type TimeOfDay int

// 時間帯定数。暦の1日は夜明けで始まるので、区切り順に夜明けから並べる。
// これで TotalTurns=0 が1日の始まりを無変換で指し、日数も季節もオフセットなしで導ける。
// かっこ内は1周目の経過ターン範囲で、以降は1日ごとに繰り返す
const (
	TimeDawn     TimeOfDay = iota // 夜明け 経過0-249
	TimeMorning                   // 朝 経過250-499
	TimeDay                       // 昼 経過500-749
	TimeEvening                   // 夕 経過750-999
	TimeNight                     // 夜 経過1000-1249
	TimeMidnight                  // 深夜 経過1250-1499
)

// TurnsPerDay は1日のターン数。バランス導出が生存日数を日単位で出せるよう公開する。
const TurnsPerDay consts.Turn = 1500

// turnsPerDay は当パッケージ内で使う別名。
const turnsPerDay = TurnsPerDay

// 時間帯ごとのターン数
const turnsPerTimeOfDay consts.Turn = turnsPerDay / 6 // 250ターン

// GameTime はゲーム内時間を管理する
type GameTime struct {
	TotalTurns consts.Turn // run 開始からの経過ターン数
}

// GetTimeOfDay は現在の時間帯を返す
func (gt *GameTime) GetTimeOfDay() TimeOfDay {
	return TimeOfDay(gt.TotalTurns % turnsPerDay / turnsPerTimeOfDay)
}

// NumTimeOfDay は時間帯の区分数。時間帯ごとの値の配列の長さに使う
const NumTimeOfDay = int(turnsPerDay / turnsPerTimeOfDay)

// GetDaylightLerp は日照や色味を連続に補間するための、前後のアンカー時間帯とその間の位置 0..1 を返す。
// 各時間帯を区間の中心にアンカーする。境界で段差にせず中心の代表値へなだらかに寄せるため。
// 値表は CalendarDefs が持ち、時間の割り出しだけをここに置く。時間帯の区切りに要る turnsPerDay と
// turnsPerTimeOfDay が components にあるため。補間そのものは呼び出し側が行う。
func (gt *GameTime) GetDaylightLerp() (from, to TimeOfDay, t float64) {
	// 中心へアンカーするため半区間ずらす。turnsPerTimeOfDay=250 は偶数で端数は出ない
	half := turnsPerTimeOfDay / 2
	// 中心より前は1つ前の区間へ回り込むので turnsPerDay を足して正へ寄せる
	rel := (gt.TotalTurns%turnsPerDay - half + turnsPerDay) % turnsPerDay
	idx := int(rel / turnsPerTimeOfDay)
	t = float64(rel%turnsPerTimeOfDay) / float64(turnsPerTimeOfDay)
	return TimeOfDay(idx), TimeOfDay((idx + 1) % NumTimeOfDay), t
}

// GetDayNumber は経過日数を返す（1日目から始まる）。turn 0 が1日目の夜明けなのでオフセットは要らない
func (gt *GameTime) GetDayNumber() int {
	return int(gt.TotalTurns/turnsPerDay) + 1
}

// daysPerYear は季節1周の日数
const daysPerYear = 32

// Season は季節を表す。1年を4等分し、春開始で巡る。
type Season int

// 季節定数。CalendarDefs.SeasonalTemperatureForDay の正弦波と位相を合わせる。
const (
	SeasonSpring Season = iota // 春
	SeasonSummer               // 夏
	SeasonAutumn               // 秋
	SeasonWinter               // 冬
	// NumSeason は季節の数。季節ごとの重みの配列の長さに使う
	NumSeason = iota
)

// GetSeason は経過日数から現在の季節を返す。
func (gt *GameTime) GetSeason() Season {
	const daysPerSeason = daysPerYear / 4
	dayOfYear := (gt.GetDayNumber() - 1) % daysPerYear
	return Season(dayOfYear / daysPerSeason)
}

// SeasonJustChanged は直前のターンから季節が変わったかを返す。Advance の直後に呼ぶ想定。
func (gt *GameTime) SeasonJustChanged() bool {
	if gt.TotalTurns == 0 {
		return false
	}
	prev := GameTime{TotalTurns: gt.TotalTurns - 1}
	return gt.GetSeason() != prev.GetSeason()
}

// DayJustChanged は現在の日数と、直前のターンから日付が変わったかを返す。Advance の直後に呼ぶ想定。
// 呼び出し側が変わった先の日数をそのまま通知に使えるよう日数も返す。
func (gt *GameTime) DayJustChanged() (int, bool) {
	cur := gt.GetDayNumber()
	if gt.TotalTurns == 0 {
		return cur, false
	}
	prev := GameTime{TotalTurns: gt.TotalTurns - 1}
	return cur, prev.GetDayNumber() != cur
}

// TimeOfDayJustChanged は現在の時間帯と、直前のターンから変わったかを返す。
// 呼び出し側が入った先で日の出入りを見分けられるよう時間帯も返す。
func (gt *GameTime) TimeOfDayJustChanged() (TimeOfDay, bool) {
	cur := gt.GetTimeOfDay()
	if gt.TotalTurns == 0 {
		return cur, false
	}
	prev := GameTime{TotalTurns: gt.TotalTurns - 1}
	return cur, prev.GetTimeOfDay() != cur
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
// 冬からは翌年の春へ折り返す。turn 0 が夜明け始まりなので、季節境界は turnsPerSeason の倍数に一致する
func (gt *GameTime) AdvanceToNextSeason() {
	gt.TotalTurns = (gt.TotalTurns/turnsPerSeason + 1) * turnsPerSeason
}
