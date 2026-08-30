package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// logEnvironmentChange は季節や日の出入りが変わったらゲームログへ出す。
// 変化は現在ターンと直前ターンの導出値の差で判定するので、GameTime.Advance の直後に呼ぶ。
func logEnvironmentChange(world w.World) {
	gt := query.GetGameTime(world)

	if gt.SeasonJustChanged() {
		name := query.T(world, gt.GetSeason().String())
		gamelog.New(query.GetGameLog(world)).
			Markup(gamelog.Tag("system", query.T(world, "The season changed to %s.", name))).
			Log()
	}

	// 知らせるのは日の出と日の入りだけにする。夜明け入りが日の出、夕入りが日の入りに当たる。
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
			// ほかの区分は日の出入りではないので出さない
		}
	}
}
