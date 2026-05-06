package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentPlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	t.Run("壁で囲まれた領域は屋内になる", func(t *testing.T) {
		t.Parallel()

		// 5x5 のマップを作成
		// W W W W W
		// W . . . W
		// W . . . W
		// W . . . W
		// W W W W W
		mp := &MetaPlan{
			Level: newTestLevel(5, 5),
			Tiles: make([]oapi.Tile, 25),
		}

		// 壁で囲む
		for i := 0; i < 25; i++ {
			x, y := i%5, i/5
			if x == 0 || x == 4 || y == 0 || y == 4 {
				mp.Tiles[i] = oapi.Tile{Name: "wall", BlockPass: true}
			} else {
				mp.Tiles[i] = oapi.Tile{Name: "floor", BlockPass: false}
			}
		}

		planner := EnvironmentPlanner{}
		err := planner.PlanMeta(mp)
		require.NoError(t, err)

		// 中央の床タイルは屋内
		centerIdx := 2*5 + 2 // (2, 2)
		assert.Equal(t, oapi.ShelterType(gc.ShelterFull), mp.Tiles[centerIdx].Shelter, "中央は屋内であるべき")

		// 壁タイルは屋内（壁は通過できないので屋外判定されない）
		wallIdx := 0 // (0, 0)
		assert.Equal(t, oapi.ShelterType(gc.ShelterFull), mp.Tiles[wallIdx].Shelter, "壁は屋内扱い")
	})

	t.Run("マップ端から到達可能な領域は屋外になる", func(t *testing.T) {
		t.Parallel()

		// 5x5 の全て床のマップ
		mp := &MetaPlan{
			Level: newTestLevel(5, 5),
			Tiles: make([]oapi.Tile, 25),
		}

		for i := 0; i < 25; i++ {
			mp.Tiles[i] = oapi.Tile{Name: "floor", BlockPass: false}
		}

		planner := EnvironmentPlanner{}
		err := planner.PlanMeta(mp)
		require.NoError(t, err)

		// 全てのタイルが屋外
		for i := 0; i < 25; i++ {
			assert.Equal(t, oapi.ShelterType(gc.ShelterNone), mp.Tiles[i].Shelter, "全て屋外であるべき")
		}
	})

	t.Run("部分的に囲まれた領域", func(t *testing.T) {
		t.Parallel()

		// 7x5 のマップ
		// . . . . . . .
		// . W W W W . .
		// . W . . W . .
		// . W W W W . .
		// . . . . . . .
		mp := &MetaPlan{
			Level: newTestLevel(7, 5),
			Tiles: make([]oapi.Tile, 35),
		}

		for i := 0; i < 35; i++ {
			mp.Tiles[i] = oapi.Tile{Name: "floor", BlockPass: false}
		}

		// 壁を配置
		wallPositions := []int{
			1*7 + 1, 1*7 + 2, 1*7 + 3, 1*7 + 4, // 上の壁
			2*7 + 1, 2*7 + 4, // 左右の壁
			3*7 + 1, 3*7 + 2, 3*7 + 3, 3*7 + 4, // 下の壁
		}
		for _, idx := range wallPositions {
			mp.Tiles[idx] = oapi.Tile{Name: "wall", BlockPass: true}
		}

		planner := EnvironmentPlanner{}
		err := planner.PlanMeta(mp)
		require.NoError(t, err)

		// 囲まれた内部は屋内
		insideIdx := 2*7 + 2 // (2, 2)
		assert.Equal(t, oapi.ShelterType(gc.ShelterFull), mp.Tiles[insideIdx].Shelter, "囲まれた内部は屋内")

		// 外部は屋外
		outsideIdx := 0 // (0, 0)
		assert.Equal(t, oapi.ShelterType(gc.ShelterNone), mp.Tiles[outsideIdx].Shelter, "外部は屋外")
	})
}

func TestEnvironmentPlanner_calcWater(t *testing.T) {
	t.Parallel()

	t.Run("水タイル上は水中", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]oapi.Tile, 9),
		}
		for i := 0; i < 9; i++ {
			mp.Tiles[i] = oapi.Tile{Name: "floor", BlockPass: false}
		}
		mp.Tiles[4] = oapi.Tile{Name: "water", BlockPass: false}

		planner := EnvironmentPlanner{}
		result := planner.calcWater(mp, 4)

		assert.Equal(t, gc.WaterSubmerged, result)
	})

	t.Run("水タイルに隣接すると水辺", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]oapi.Tile, 9),
		}
		for i := 0; i < 9; i++ {
			mp.Tiles[i] = oapi.Tile{Name: "floor", BlockPass: false}
		}
		mp.Tiles[4] = oapi.Tile{Name: "water", BlockPass: false}

		planner := EnvironmentPlanner{}

		// 隣接タイル
		assert.Equal(t, gc.WaterNearby, planner.calcWater(mp, 1)) // 上
		assert.Equal(t, gc.WaterNearby, planner.calcWater(mp, 3)) // 左
		assert.Equal(t, gc.WaterNearby, planner.calcWater(mp, 5)) // 右
		assert.Equal(t, gc.WaterNearby, planner.calcWater(mp, 7)) // 下
	})

	t.Run("水タイルから離れているとなし", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]oapi.Tile, 9),
		}
		for i := 0; i < 9; i++ {
			mp.Tiles[i] = oapi.Tile{Name: "floor", BlockPass: false}
		}

		planner := EnvironmentPlanner{}
		result := planner.calcWater(mp, 4)

		assert.Equal(t, gc.WaterNone, result)
	})
}

func TestEnvironmentPlanner_calcFoliage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tileName string
		expected gc.FoliageType
	}{
		{"森タイルは森", "forest", gc.FoliageForest},
		{"木タイルは森", "tree", gc.FoliageForest},
		{"草タイルは草原", "grass", gc.FoliageGrass},
		{"床タイルはなし", "floor", gc.FoliageNone},
		{"壁タイルはなし", "wall", gc.FoliageNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mp := &MetaPlan{
				Level: newTestLevel(1, 1),
				Tiles: []oapi.Tile{{Name: tt.tileName}},
			}

			planner := EnvironmentPlanner{}
			result := planner.calcFoliage(mp, 0)

			assert.Equal(t, tt.expected, result)
		})
	}
}

func newTestLevel(width, height int) resources.Level {
	return resources.Level{
		TileWidth:  consts.Tile(width),
		TileHeight: consts.Tile(height),
	}
}
