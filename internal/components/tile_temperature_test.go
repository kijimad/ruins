package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTileTemperature_Total(t *testing.T) {
	t.Parallel()

	t.Run("全要素がゼロの場合", func(t *testing.T) {
		t.Parallel()
		tt := &TileTemperature{}
		assert.Equal(t, 0, tt.Total())
	})

	t.Run("屋内のみ", func(t *testing.T) {
		t.Parallel()
		tt := &TileTemperature{Shelter: ShelterFull}
		assert.Equal(t, 10, tt.Total())
	})

	t.Run("水辺で森", func(t *testing.T) {
		t.Parallel()
		tt := &TileTemperature{Water: WaterNearby, Foliage: FoliageForest}
		assert.Equal(t, -8, tt.Total()) // -5 + (-3) = -8
	})

	t.Run("水中", func(t *testing.T) {
		t.Parallel()
		tt := &TileTemperature{Water: WaterSubmerged}
		assert.Equal(t, -10, tt.Total())
	})

	t.Run("全要素の組み合わせ", func(t *testing.T) {
		t.Parallel()
		tt := &TileTemperature{
			Shelter: ShelterPartial,
			Water:   WaterNearby,
			Foliage: FoliageGrass,
		}
		// 5 + (-5) + (-1) = -1
		assert.Equal(t, -1, tt.Total())
	})
}

func TestTileTemperatureConstants(t *testing.T) {
	t.Parallel()

	t.Run("屋内は屋外より暖かい", func(t *testing.T) {
		t.Parallel()
		assert.Greater(t, int(ShelterFull), int(ShelterNone))
	})

	t.Run("半屋外は屋内と屋外の中間", func(t *testing.T) {
		t.Parallel()
		assert.Greater(t, int(ShelterPartial), int(ShelterNone))
		assert.Less(t, int(ShelterPartial), int(ShelterFull))
	})

	t.Run("水中は水辺より冷たい", func(t *testing.T) {
		t.Parallel()
		assert.Less(t, int(WaterSubmerged), int(WaterNearby))
	})

	t.Run("森は草原より涼しい", func(t *testing.T) {
		t.Parallel()
		assert.Less(t, int(FoliageForest), int(FoliageGrass))
	})
}
