package balance

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
)

func TestWorldTemperatureAtDay_季節を巡る(t *testing.T) {
	t.Parallel()
	calendar := raw.NewCalendarDefs(loadTestMaster(t))
	// 夏ピークが最も暖かく、冬底が最も寒い。春は中間。
	assert.Greater(t, WorldTemperatureAtDay(calendar, 8), WorldTemperatureAtDay(calendar, 1), "夏は春より暖かい")
	assert.Less(t, WorldTemperatureAtDay(calendar, 24), WorldTemperatureAtDay(calendar, 1), "冬は春より寒い")
	// 1年32日周期。1年後の同じ日は同じ温度に戻る。
	assert.Equal(t, WorldTemperatureAtDay(calendar, 1), WorldTemperatureAtDay(calendar, 1+32), "1年周期で戻る")
}
