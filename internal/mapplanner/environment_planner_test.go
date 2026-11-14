package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentPlanner_PlanMeta_囲われはタイルの行の値を使う(t *testing.T) {
	t.Parallel()

	partial := rawschema.Partial
	indoor := rawschema.Indoor
	mp := &MetaPlan{
		Level: newTestLevel(3, 1),
		Tiles: []rawschema.Tile{
			{Id: "dirt", Name: "dirt"},
			{Id: "carport", Name: "carport", Shelter: &partial},
			{Id: "floor", Name: "floor", Shelter: &indoor},
		},
	}
	require.NoError(t, EnvironmentPlanner{}.PlanMeta(mp))

	assert.Equal(t, []rawschema.Shelter{rawschema.Outdoor, rawschema.Partial, rawschema.Indoor},
		[]rawschema.Shelter{mp.Environments[0].Shelter, mp.Environments[1].Shelter, mp.Environments[2].Shelter})
}

func TestEnvironmentPlanner_calcWater(t *testing.T) {
	t.Parallel()

	t.Run("水タイル上は水中", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]rawschema.Tile, 9),
		}
		for i := range 9 {
			mp.Tiles[i] = rawschema.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}
		mp.Tiles[4] = rawschema.Tile{Id: "water", Name: "water", BlockPass: false, Water: new(true)}

		planner := EnvironmentPlanner{}
		result := planner.calcWater(mp, 4)

		assert.Equal(t, rawschema.Submerged, result)
	})

	t.Run("水タイルに隣接すると水辺", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]rawschema.Tile, 9),
		}
		for i := range 9 {
			mp.Tiles[i] = rawschema.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}
		mp.Tiles[4] = rawschema.Tile{Id: "water", Name: "water", BlockPass: false, Water: new(true)}

		planner := EnvironmentPlanner{}

		// 隣接タイル
		assert.Equal(t, rawschema.Nearby, planner.calcWater(mp, 1)) // 上
		assert.Equal(t, rawschema.Nearby, planner.calcWater(mp, 3)) // 左
		assert.Equal(t, rawschema.Nearby, planner.calcWater(mp, 5)) // 右
		assert.Equal(t, rawschema.Nearby, planner.calcWater(mp, 7)) // 下
	})

	t.Run("水タイルから離れているとなし", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]rawschema.Tile, 9),
		}
		for i := range 9 {
			mp.Tiles[i] = rawschema.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}

		planner := EnvironmentPlanner{}
		result := planner.calcWater(mp, 4)

		assert.Equal(t, rawschema.WaterExposure(""), result)
	})
}

func TestEnvironmentPlanner_PlanMeta_植生はタイルの行の値を使う(t *testing.T) {
	t.Parallel()

	grass := rawschema.FoliageGrass
	mp := &MetaPlan{
		Level: newTestLevel(2, 1),
		Tiles: []rawschema.Tile{
			{Id: "grass_floor", Name: "grass_floor", Foliage: &grass},
			{Id: "floor", Name: "floor"},
		},
	}
	require.NoError(t, EnvironmentPlanner{}.PlanMeta(mp))

	assert.Equal(t, []rawschema.Foliage{rawschema.FoliageGrass, ""}, []rawschema.Foliage{mp.Environments[0].Foliage, mp.Environments[1].Foliage})
}

func newTestLevel(width, height int) gc.Level {
	return gc.Level{
		TileWidth:  consts.Tile(width),
		TileHeight: consts.Tile(height),
	}
}
