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
