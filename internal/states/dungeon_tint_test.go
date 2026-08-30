package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestTimeOfDayTintAnchor は時間帯の中心で世界へ掛ける乗算色を固定する。
// 昼は素通し、朝夕は暖色、夜は寒色という空気感の方向を検証する。
func TestTimeOfDayTintAnchor(t *testing.T) {
	t.Parallel()

	// 昼は白で素通し
	r, g, b := timeOfDayTintAnchor(gc.TimeDay)
	assert.Equal(t, [3]float32{1, 1, 1}, [3]float32{r, g, b}, "昼は素通し")

	// 朝夕は暖色。赤が青より強い
	for _, tod := range []gc.TimeOfDay{gc.TimeDawn, gc.TimeEvening} {
		r, _, b := timeOfDayTintAnchor(tod)
		assert.Greater(t, r, b, "朝夕は暖色で赤が強い")
	}

	// 夜・深夜は寒色。青が赤より強い
	for _, tod := range []gc.TimeOfDay{gc.TimeNight, gc.TimeMidnight} {
		r, _, b := timeOfDayTintAnchor(tod)
		assert.Greater(t, b, r, "夜は寒色で青が強い")
	}

	// 深夜は夜より暗い。乗算色の総和が小さいほど暗い
	nr, ng, nb := timeOfDayTintAnchor(gc.TimeNight)
	mr, mg, mb := timeOfDayTintAnchor(gc.TimeMidnight)
	assert.Less(t, mr+mg+mb, nr+ng+nb, "深夜は夜より暗い")
}

// TestTimeOfDayTint_連続補間 は色味が時間帯の境界で段差にならず、区間内で徐々に変わることを検証する。
// 中心アンカーなので、時間帯の中心ではその代表色、区間の入り口では前後の中間になる。
func TestTimeOfDayTint_連続補間(t *testing.T) {
	t.Parallel()

	// 夕の中心(turn 375)は夕の代表色そのもの
	r, g, b := timeOfDayTint(&gc.GameTime{TotalTurns: 375})
	assert.InDelta(t, 1.0, r, 1e-6, "夕の中心は夕の赤")
	assert.InDelta(t, 0.72, g, 1e-6, "夕の中心は夕の緑")
	assert.InDelta(t, 0.52, b, 1e-6, "夕の中心は夕の青")

	// 夕の入り口(turn 250)は昼(白)と夕の中間。まだ暖色寄りだが夕の代表色ほど濃くない
	sr, sg, sb := timeOfDayTint(&gc.GameTime{TotalTurns: 250})
	assert.InDelta(t, (1.0+1.0)/2, sr, 1e-6, "赤は昼と夕の中間")
	assert.InDelta(t, (1.0+0.72)/2, sg, 1e-6, "緑は昼と夕の中間")
	assert.InDelta(t, (1.0+0.52)/2, sb, 1e-6, "青は昼と夕の中間")
	assert.Greater(t, sb, b, "夕の入り口は夕の中心より色が薄い")
}
