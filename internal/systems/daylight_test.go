package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// calendarDefs は実データの時間帯と季節の表を返す
func calendarDefs(t *testing.T) gc.CalendarDefs {
	t.Helper()
	return testutil.InitTestWorld(t).Resources.CalendarDefs
}

// TestOverworldDaylightAnchor_昼夜の大小関係 は昼が最も明るく深夜が最も暗いことを保証する。
// 個々の値を変えても、この不変条件が崩れないことを検知する。
func TestOverworldDaylightAnchor_昼夜の大小関係(t *testing.T) {
	t.Parallel()
	tods := calendarDefs(t).TimesOfDay
	overworldDaylightAnchor := func(tod gc.TimeOfDay) float64 { return tods[tod].Daylight }

	assert.Greater(t, overworldDaylightAnchor(gc.TimeDay), overworldDaylightAnchor(gc.TimeNight),
		"昼は夜より明るい")
	assert.Greater(t, overworldDaylightAnchor(gc.TimeNight), overworldDaylightAnchor(gc.TimeMidnight),
		"夜は深夜より明るい")

	// 薄暮の夕・夜明けは昼より暗く夜より明るい。逆転させる誤りを検知する
	assert.Greater(t, overworldDaylightAnchor(gc.TimeDay), overworldDaylightAnchor(gc.TimeEvening),
		"昼は夕より明るい")
	assert.Greater(t, overworldDaylightAnchor(gc.TimeEvening), overworldDaylightAnchor(gc.TimeNight),
		"夕は夜より明るい")
	assert.Greater(t, overworldDaylightAnchor(gc.TimeMorning), overworldDaylightAnchor(gc.TimeDawn),
		"朝は夜明けより明るい")
	assert.Greater(t, overworldDaylightAnchor(gc.TimeDawn), overworldDaylightAnchor(gc.TimeMidnight),
		"夜明けは深夜より明るい")
}

// TestOverworldAmbientColorAnchor は時間帯の中心での環境光の色を固定する。
// 昼は無彩色、朝夕は暖色で赤が強い、夜は寒色で青が強い、という空気感の方向を検証する。
func TestOverworldAmbientColorAnchor(t *testing.T) {
	t.Parallel()
	tods := calendarDefs(t).TimesOfDay
	overworldAmbientColorAnchor := func(tod gc.TimeOfDay) [3]float64 { return tods[tod].AmbientColor }

	assert.Equal(t, [3]float64{1, 1, 1}, overworldAmbientColorAnchor(gc.TimeDay), "昼は無彩色")

	for _, tod := range []gc.TimeOfDay{gc.TimeDawn, gc.TimeEvening} {
		c := overworldAmbientColorAnchor(tod)
		assert.Greater(t, c[0], c[2], "朝夕は暖色で赤が強い")
	}

	for _, tod := range []gc.TimeOfDay{gc.TimeNight, gc.TimeMidnight} {
		c := overworldAmbientColorAnchor(tod)
		assert.Greater(t, c[2], c[0], "夜は寒色で青が強い")
	}
}

// TestOverworldAmbientColor_連続補間 は色が時間帯の境界で段差にならず、区間内で徐々に変わることを検証する。
func TestOverworldAmbientColor_連続補間(t *testing.T) {
	t.Parallel()
	defs := calendarDefs(t)

	// 夕の中心(turn 875)は夕の代表色そのもの
	assert.Equal(t, [3]float64{1.0, 0.72, 0.52}, overworldAmbientColor(defs, &gc.GameTime{TotalTurns: 875}), "夕の中心は夕の色")

	// 夕の入り口(turn 750)は昼(無彩色)と夕の中間。まだ暖色寄りだが夕ほど濃くない
	c := overworldAmbientColor(defs, &gc.GameTime{TotalTurns: 750})
	assert.InDelta(t, (1.0+1.0)/2, c[0], 1e-9, "赤は昼と夕の中間")
	assert.InDelta(t, (1.0+0.72)/2, c[1], 1e-9, "緑は昼と夕の中間")
	assert.Greater(t, c[2], 0.52, "夕の入り口は夕の中心より青が残り色が薄い")
}

// TestOverworldDaylight_連続補間 は明るさが時間帯の境界で段差にならず、区間内で徐々に変わることを検証する。
// 中心アンカーなので、時間帯の中心ではその代表値、区間の入り口では前後の中間になる。
func TestOverworldDaylight_連続補間(t *testing.T) {
	t.Parallel()
	defs := calendarDefs(t)

	// turnsPerTimeOfDay=250。夜明け始まりなので各時間帯の中心は 125 + 250*idx（昼=625, 夕=875）
	eveningCenter := &gc.GameTime{TotalTurns: 875} // 夕の中心
	assert.InDelta(t, 0.55, overworldDaylight(defs, eveningCenter), 1e-9, "夕の中心は夕の代表値")

	// 夕の入り口(turn 750)は昼と夕の中間。フラットな夕より明るく、夜っぽくない
	eveningStart := &gc.GameTime{TotalTurns: 750}
	got := overworldDaylight(defs, eveningStart)
	assert.InDelta(t, (0.95+0.55)/2, got, 1e-9, "夕の入り口は昼と夕の中間")
	assert.Greater(t, got, defs.TimesOfDay[gc.TimeEvening].Daylight, "夕の入り口はフラットな夕より明るい")

	// 夕の入り口から夜の入り口へ向けて単調に暗くなる
	assert.Greater(t, overworldDaylight(defs, &gc.GameTime{TotalTurns: 750}),
		overworldDaylight(defs, &gc.GameTime{TotalTurns: 900}), "夕は進むほど暗くなる")
	assert.Greater(t, overworldDaylight(defs, &gc.GameTime{TotalTurns: 900}),
		overworldDaylight(defs, &gc.GameTime{TotalTurns: 1000}), "夜へ向けてさらに暗くなる")
}

// TestOverworldDaylight_一日を通して単調に増減する は、昼の中心から深夜の中心まで単調に暗くなり、
// 深夜から翌昼まで単調に明るくなることを、細かく刻んで検証する。
func TestOverworldDaylight_一日を通して単調に増減する(t *testing.T) {
	t.Parallel()
	defs := calendarDefs(t)

	daylightAt := func(turn consts.Turn) float64 {
		return overworldDaylight(defs, &gc.GameTime{TotalTurns: turn})
	}

	// 昼の中心 625 から深夜の中心 1375 まで、暗くなり続ける
	for turn := consts.Turn(625); turn < 1375; turn += 25 {
		assert.Greater(t, daylightAt(turn), daylightAt(turn+25),
			"昼から深夜へ向かう %d→%d は暗くなる", turn, turn+25)
	}

	// 深夜の中心 1375 から翌日の昼の中心 2125 まで、明るくなり続ける
	for turn := consts.Turn(1375); turn < 2125; turn += 25 {
		assert.Less(t, daylightAt(turn), daylightAt(turn+25),
			"深夜から翌昼へ向かう %d→%d は明るくなる", turn, turn+25)
	}
}
