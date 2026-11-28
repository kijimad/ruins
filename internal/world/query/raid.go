package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// raidEveryDays は襲撃の間隔の日数。この倍数の日の夜に襲撃が来る
const raidEveryDays = 7

// raidWaveTurns は襲撃の夜に波が来る、その日の中のターン。夜の始まりから小休止を挟んで来る
var raidWaveTurns = [...]consts.Turn{1000, 1150, 1300}

// RaidWaveCount は1晩の波の数
const RaidWaveCount = len(raidWaveTurns)

// RaidWarnsAt は経過ターン turn がちょうど襲撃の日の夜明けかを返す
func RaidWarnsAt(turn consts.Turn) bool {
	day, offset := dayAndOffset(turn)
	return offset == 0 && isRaidDay(day)
}

// RaidWaveAt は経過ターン turn がちょうど襲撃の波の来るターンなら、0から数えた波の番号と true を返す
func RaidWaveAt(turn consts.Turn) (int, bool) {
	day, offset := dayAndOffset(turn)
	if !isRaidDay(day) {
		return 0, false
	}
	for i, t := range raidWaveTurns {
		if offset == t {
			return i, true
		}
	}
	return 0, false
}

// RaidEndsAt は経過ターン turn がちょうど襲撃の夜が明けるターン、すなわち襲撃の日の翌日の夜明けかを返す
func RaidEndsAt(turn consts.Turn) bool {
	day, offset := dayAndOffset(turn)
	return offset == 0 && day > 1 && isRaidDay(day-1)
}

// DaysUntilRaid は今日を含めず、次の襲撃の夜まであと何日かを返す。今日が襲撃の日なら0
func DaysUntilRaid(turn consts.Turn) int {
	day, _ := dayAndOffset(turn)
	return (raidEveryDays - day%raidEveryDays) % raidEveryDays
}

// dayAndOffset は経過ターンを、1から数える日付とその日の中のターンに分ける
func dayAndOffset(turn consts.Turn) (int, consts.Turn) {
	return int(turn/gc.TurnsPerDay) + 1, turn % gc.TurnsPerDay
}

func isRaidDay(day int) bool {
	return day%raidEveryDays == 0
}
