package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestOverworldDaylightAnchor は時間帯の中心での地上日照の明るさを固定する。
// 昼が最も明るく深夜が最も暗いという順序と各値を検証する。
func TestOverworldDaylightAnchor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tod  gc.TimeOfDay
		want float64
	}{
		{"夜明けは薄暗い", gc.TimeDawn, 0.40},
		{"朝は明るい", gc.TimeMorning, 0.72},
		{"昼は最も明るい", gc.TimeDay, 0.95},
		{"夕は薄暗い", gc.TimeEvening, 0.38},
		{"夜は暗い", gc.TimeNight, 0.14},
		{"深夜は最も暗い", gc.TimeMidnight, 0.06},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, overworldDaylightAnchor(tt.tod), 1e-9)
		})
	}
}

// TestOverworldDaylightAnchor_昼夜の大小関係 は昼が最も明るく深夜が最も暗いことを保証する。
// 個々の値を変えても、この不変条件が崩れないことを検知する。
func TestOverworldDaylightAnchor_昼夜の大小関係(t *testing.T) {
	t.Parallel()

	assert.Greater(t, overworldDaylightAnchor(gc.TimeDay), overworldDaylightAnchor(gc.TimeNight),
		"昼は夜より明るい")
	assert.Greater(t, overworldDaylightAnchor(gc.TimeNight), overworldDaylightAnchor(gc.TimeMidnight),
		"夜は深夜より明るい")
}

// TestOverworldAmbientColorAnchor は時間帯の中心での環境光の色を固定する。
// 昼は無彩色、朝夕は暖色で赤が強い、夜は寒色で青が強い、という空気感の方向を検証する。
func TestOverworldAmbientColorAnchor(t *testing.T) {
	t.Parallel()

	// 昼は無彩色
	assert.Equal(t, [3]float64{1, 1, 1}, overworldAmbientColorAnchor(gc.TimeDay), "昼は無彩色")

	// 朝夕は暖色。赤が青より強い
	for _, tod := range []gc.TimeOfDay{gc.TimeDawn, gc.TimeEvening} {
		c := overworldAmbientColorAnchor(tod)
		assert.Greater(t, c[0], c[2], "朝夕は暖色で赤が強い")
	}

	// 夜・深夜は寒色。青が赤より強い
	for _, tod := range []gc.TimeOfDay{gc.TimeNight, gc.TimeMidnight} {
		c := overworldAmbientColorAnchor(tod)
		assert.Greater(t, c[2], c[0], "夜は寒色で青が強い")
	}
}

// TestOverworldAmbientColor_連続補間 は色が時間帯の境界で段差にならず、区間内で徐々に変わることを検証する。
func TestOverworldAmbientColor_連続補間(t *testing.T) {
	t.Parallel()

	// 夕の中心(turn 375)は夕の代表色そのもの
	assert.Equal(t, [3]float64{1.0, 0.72, 0.52}, overworldAmbientColor(&gc.GameTime{TotalTurns: 375}), "夕の中心は夕の色")

	// 夕の入り口(turn 250)は昼(無彩色)と夕の中間。まだ暖色寄りだが夕ほど濃くない
	c := overworldAmbientColor(&gc.GameTime{TotalTurns: 250})
	assert.InDelta(t, (1.0+0.72)/2, c[1], 1e-9, "緑は昼と夕の中間")
	assert.Greater(t, c[2], 0.52, "夕の入り口は夕の中心より青が残り色が薄い")
}

// TestOverworldDaylight_連続補間 は明るさが時間帯の境界で段差にならず、区間内で徐々に変わることを検証する。
// 中心アンカーなので、時間帯の中心ではその代表値、区間の入り口では前後の中間になる。
func TestOverworldDaylight_連続補間(t *testing.T) {
	t.Parallel()

	// turnsPerTimeOfDay=250。各時間帯の中心は 125 + 250*idx
	eveningCenter := &gc.GameTime{TotalTurns: 375} // 夕の中心
	assert.InDelta(t, 0.38, overworldDaylight(eveningCenter), 1e-9, "夕の中心は夕の代表値")

	// 夕の入り口(turn 250)は昼と夕の中間。フラットな夕(0.38)より明るく、夜っぽくない
	eveningStart := &gc.GameTime{TotalTurns: 250}
	got := overworldDaylight(eveningStart)
	assert.InDelta(t, (0.95+0.38)/2, got, 1e-9, "夕の入り口は昼と夕の中間")
	assert.Greater(t, got, overworldDaylightAnchor(gc.TimeEvening), "夕の入り口はフラットな夕より明るい")

	// 夕の入り口から夜の入り口へ向けて単調に暗くなる
	assert.Greater(t, overworldDaylight(&gc.GameTime{TotalTurns: 250}),
		overworldDaylight(&gc.GameTime{TotalTurns: 400}), "夕は進むほど暗くなる")
	assert.Greater(t, overworldDaylight(&gc.GameTime{TotalTurns: 400}),
		overworldDaylight(&gc.GameTime{TotalTurns: 500}), "夜へ向けてさらに暗くなる")
}
