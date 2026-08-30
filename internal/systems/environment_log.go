package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// logEnvironmentChange は季節や日の出入りが変わったとき、プレイヤーが気付けるようにゲームログへ出す。
// 気温や視界はこれらに連動して既に変わるが、変化した瞬間を知らせる手立てがなかった。
// GameTime.Advance の直後に1度だけ呼ぶ。変化は現在ターンと直前ターンの導出値の差で判定する。
func logEnvironmentChange(world w.World) {
	gt := query.GetGameTime(world)

	if gt.SeasonJustChanged() {
		name := query.T(world, gt.GetSeason().String())
		gamelog.New(query.GetGameLog(world)).
			Markup(gamelog.Tag("system", query.T(world, "The season changed to %s.", name))).
			Log()
	}

	// 時間帯は6区分あるが、知らせるのは日の出と日の入りだけにする。
	// 夜明けへ入るのが日の出、夕へ入るのが日の入りに当たる。ほかの区分への変化は出さない
	if tod, changed := gt.TimeOfDayJustChanged(); changed {
		switch tod {
		case gc.TimeDawn:
			gamelog.New(query.GetGameLog(world)).
				Markup(gamelog.Tag("system", query.T(world, "The sun rises."))).
				Log()
		case gc.TimeEvening:
			gamelog.New(query.GetGameLog(world)).
				Markup(gamelog.Tag("system", query.T(world, "The sun sets."))).
				Log()
		default:
			// 昼・夜・深夜・朝への変化は日の出入りではないので知らせない
		}
	}
}
