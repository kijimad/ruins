package components

import (
	"fmt"
	"math"

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
	TotalTurns consts.Turn // run 開始からの経過ターン数
}

// GetTimeOfDay は現在の時間帯を返す
func (gt *GameTime) GetTimeOfDay() TimeOfDay {
	return TimeOfDay(gt.TotalTurns % turnsPerDay / turnsPerTimeOfDay)
}

// numTimeOfDay は時間帯の区分数
const numTimeOfDay = int(turnsPerDay / turnsPerTimeOfDay)

// GetDaylightLerp は日照や色味を連続に補間するための、前後のアンカー時間帯とその間の位置 0..1 を返す。
// 各時間帯を区間の中心にアンカーする。境界で段差にせず中心の代表値へなだらかに寄せるため。
// 値表は systems 側が持ち、時間の割り出しだけをここに置く。時間帯の区切りに要る turnsPerDay と
// turnsPerTimeOfDay が components にあるため。補間そのものは呼び出し側が行う。
func (gt *GameTime) GetDaylightLerp() (from, to TimeOfDay, t float64) {
	// 中心へアンカーするため半区間ずらす。turnsPerTimeOfDay=250 は偶数で端数は出ない
	half := turnsPerTimeOfDay / 2
	// 中心より前は1つ前の区間へ回り込むので turnsPerDay を足して正へ寄せる
	rel := (gt.TotalTurns%turnsPerDay - half + turnsPerDay) % turnsPerDay
	idx := int(rel / turnsPerTimeOfDay)
	t = float64(rel%turnsPerTimeOfDay) / float64(turnsPerTimeOfDay)
	return TimeOfDay(idx), TimeOfDay((idx + 1) % numTimeOfDay), t
}

// GetDayNumber は経過日数を返す（1日目から始まる）。turn 0 が1日目の夜明けなのでオフセットは要らない
func (gt *GameTime) GetDayNumber() int {
	return int(gt.TotalTurns/turnsPerDay) + 1
}

// 季節による世界温度のパラメータ。寒さの順は 冬 > 秋 > 春 > 夏。値は実プレイで調整する。
const (
	daysPerYear      = 32  // 季節1周の日数
	springTemp       = 5   // 春の中点の世界温度。開始直後は肌寒い程度に留め、冬への準備期間になる
	summerPeakTemp   = 22  // 夏ピークの世界温度
	autumnTemp       = 0   // 秋の中点の世界温度。春より寒く、冬の接近を肌で感じさせる
	winterTroughTemp = -30 // 冬底の世界温度。準備なしでは生存できない寒さ
)

// GetSeasonalTemperature は経過日数から季節による世界温度のベース値を返す。
// 春開始の区分正弦波で、春と秋が肩、夏がピーク、冬が底になる。四半期ごとに肩とピークを
// 結ぶため肩の高さを春と秋で変えられ、振幅も暖側と寒側で非対称になる。季節は保存せず日数から導く。
func (gt *GameTime) GetSeasonalTemperature() int {
	// day 1 を春の中点かつ上昇位相の起点にする。位相の比較のため1年周期へ折り返す
	phase := math.Mod(2*math.Pi*float64(gt.GetDayNumber()-1)/daysPerYear, 2*math.Pi)
	s := math.Sin(phase)

	// 年の前半の肩は春、夏ピークから冬底を挟む後半の肩は秋
	mid := float64(springTemp)
	if phase >= math.Pi/2 && phase < 3*math.Pi/2 {
		mid = float64(autumnTemp)
	}

	if s >= 0 {
		return int(math.Round(mid + (float64(summerPeakTemp)-mid)*s))
	}
	return int(math.Round(mid + (mid-float64(winterTroughTemp))*s))
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
func (gt *GameTime) SeasonJustChanged() bool {
	if gt.TotalTurns == 0 {
		return false
	}
	prev := GameTime{TotalTurns: gt.TotalTurns - 1}
	return gt.GetSeason() != prev.GetSeason()
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
// 冬からは翌年の春へ折り返す。turn 0 が夜明け始まりなので、季節境界は turnsPerSeason の倍数に一致する
func (gt *GameTime) AdvanceToNextSeason() {
	gt.TotalTurns = (gt.TotalTurns/turnsPerSeason + 1) * turnsPerSeason
}
