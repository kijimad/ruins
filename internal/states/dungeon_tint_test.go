package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestTimeOfDayTint は時間帯から世界へ掛ける乗算色への写像を固定する。
// 昼は素通し、朝夕は暖色、夜は寒色という空気感の方向を検証する。
func TestTimeOfDayTint(t *testing.T) {
	t.Parallel()

	// 昼は白で素通し
	r, g, b := timeOfDayTint(gc.TimeDay)
	assert.Equal(t, [3]float32{1, 1, 1}, [3]float32{r, g, b}, "昼は素通し")

	// 朝夕は暖色。赤が青より強い
	for _, tod := range []gc.TimeOfDay{gc.TimeDawn, gc.TimeEvening} {
		r, _, b := timeOfDayTint(tod)
		assert.Greater(t, r, b, "朝夕は暖色で赤が強い")
	}

	// 夜・深夜は寒色。青が赤より強い
	for _, tod := range []gc.TimeOfDay{gc.TimeNight, gc.TimeMidnight} {
		r, _, b := timeOfDayTint(tod)
		assert.Greater(t, b, r, "夜は寒色で青が強い")
	}

	// 深夜は夜より暗い。乗算色の総和が小さいほど暗い
	nr, ng, nb := timeOfDayTint(gc.TimeNight)
	mr, mg, mb := timeOfDayTint(gc.TimeMidnight)
	assert.Less(t, mr+mg+mb, nr+ng+nb, "深夜は夜より暗い")
}
