package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
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

// InRaidNight は経過ターン turn が襲撃の夜の中、すなわち襲撃の日の最初の波から翌日の夜明けの前までかを返す
func InRaidNight(turn consts.Turn) bool {
	day, offset := dayAndOffset(turn)
	return isRaidDay(day) && offset >= raidWaveTurns[0]
}

// daysUntilRaid は今日を含めず、次の襲撃の夜まであと何日かを返す。今日が襲撃の日なら0
func daysUntilRaid(turn consts.Turn) int {
	day, _ := dayAndOffset(turn)
	return (raidEveryDays - day%raidEveryDays) % raidEveryDays
}

// RaidTargets は襲撃の敵が狙う的の位置を返す。現ステージのキューブで、展開しているかに関わらない。畳んだまま夜を
// 越せばキューブを守らずに済む、という抜け道を作らない。キューブが無ければプレイヤーの位置にし、どちらも無ければ空を返す
func RaidTargets(world w.World) []consts.Coord[consts.Tile] {
	var out []consts.Coord[consts.Tile]
	q := ActiveFilter2[gc.Cube, gc.GridElement](world).Query()
	for q.Next() {
		_, g := q.Get()
		out = append(out, g.Coord)
	}
	if len(out) > 0 {
		return out
	}
	player, err := GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return nil
	}
	return []consts.Coord[consts.Tile]{world.Components.GridElement.Get(player).Coord}
}

// NextRaidWaveTurn は経過ターン turn から数えて、次の襲撃の最初の波が来る経過ターンを返す。
// 今日が襲撃の日で、turn が最初の波のターンちょうどかそれより前なら、今日の最初の波を返す。波はターンの終わりに
// 湧くので、ちょうどのターンはまだ湧いていない
func NextRaidWaveTurn(turn consts.Turn) consts.Turn {
	day, offset := dayAndOffset(turn)
	raidDay := day + daysUntilRaid(turn)
	if raidDay == day && offset > raidWaveTurns[0] {
		raidDay += raidEveryDays
	}
	return consts.Turn(raidDay-1)*gc.TurnsPerDay + raidWaveTurns[0]
}

// dayAndOffset は経過ターンを、1から数える日付とその日の中のターンに分ける
func dayAndOffset(turn consts.Turn) (int, consts.Turn) {
	return int(turn/gc.TurnsPerDay) + 1, turn % gc.TurnsPerDay
}

func isRaidDay(day int) bool {
	return day%raidEveryDays == 0
}
