package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
)

// TestFrostVisible は霜を描くべきかの判定を検証する。霜は帯を持つオーバーワールド固有の演出。
// 共存方式では遺跡へ入っても SeamlessBand の前線は退避されたまま残るため、現ステージが
// オーバーワールドかで判定する。遺跡内では描かないことを固定する。
func TestFrostVisible(t *testing.T) {
	t.Parallel()

	overworld := gc.NewOverworldStage()
	ruin := gc.NewNamedDungeonStage("テスト遺跡", 1)

	assert.True(t, frostVisible(overworld), "オーバーワールドでは霜を描く")
	assert.False(t, frostVisible(ruin), "遺跡内では霜を描かない")
}

func TestFrostAlpha(t *testing.T) {
	t.Parallel()

	const frontEast, coldZoneWest = 30, 10 // ゾーンは (10, 30]、幅20

	t.Run("前線より東は塗らない", func(t *testing.T) {
		t.Parallel()
		_, draw := frostAlpha(frontEast, coldZoneWest, 31)
		assert.False(t, draw)
	})

	t.Run("前線東端は薄く塗る", func(t *testing.T) {
		t.Parallel()
		alpha, draw := frostAlpha(frontEast, coldZoneWest, 30)
		assert.True(t, draw)
		assert.InDelta(t, 0.25, alpha, 0.001, "東端は最も薄い")
	})

	t.Run("ゾーン西へ深いほど濃い", func(t *testing.T) {
		t.Parallel()
		east, _ := frostAlpha(frontEast, coldZoneWest, 25)
		west, _ := frostAlpha(frontEast, coldZoneWest, 15)
		assert.Greater(t, west, east, "西へ深いほど濃くなる")
	})

	t.Run("進入不可ライン以西は凍結壁として最も濃い", func(t *testing.T) {
		t.Parallel()
		wall, draw := frostAlpha(frontEast, coldZoneWest, 10)
		assert.True(t, draw)
		assert.InDelta(t, 0.9, wall, 0.001)

		farWest, draw := frostAlpha(frontEast, coldZoneWest, -100)
		assert.True(t, draw)
		assert.InDelta(t, 0.9, farWest, 0.001, "はるか西も凍結壁")
	})
}
