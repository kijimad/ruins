package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
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
		for i := range 25 {
			x, y := i%5, i/5
			if x == 0 || x == 4 || y == 0 || y == 4 {
				mp.Tiles[i] = oapi.Tile{Id: "wall", Name: "wall", BlockPass: true}
			} else {
				mp.Tiles[i] = oapi.Tile{Id: "floor", Name: "floor", BlockPass: false}
			}
		}

		planner := EnvironmentPlanner{}
		err := planner.PlanMeta(mp)
		require.NoError(t, err)

		// 中央の床タイルは屋内
		centerIdx := 2*5 + 2 // (2, 2)
		assert.Equal(t, oapi.Indoor, mp.Environments[centerIdx].Shelter, "中央は屋内であるべき")

		// 壁タイルは屋内（壁は通過できないので屋外判定されない）
		wallIdx := 0 // (0, 0)
		assert.Equal(t, oapi.Indoor, mp.Environments[wallIdx].Shelter, "壁は屋内扱い")
	})

	t.Run("マップ端から到達可能な領域は屋外になる", func(t *testing.T) {
		t.Parallel()

		// 5x5 の全て床のマップ
		mp := &MetaPlan{
			Level: newTestLevel(5, 5),
			Tiles: make([]oapi.Tile, 25),
		}

		for i := range 25 {
			mp.Tiles[i] = oapi.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}

		planner := EnvironmentPlanner{}
		err := planner.PlanMeta(mp)
		require.NoError(t, err)

		// 全てのタイルが屋外
		for i := range 25 {
			assert.Equal(t, oapi.Outdoor, mp.Environments[i].Shelter, "全て屋外であるべき")
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

		for i := range 35 {
			mp.Tiles[i] = oapi.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}

		// 壁を配置
		wallPositions := []int{
			1*7 + 1, 1*7 + 2, 1*7 + 3, 1*7 + 4, // 上の壁
			2*7 + 1, 2*7 + 4, // 左右の壁
			3*7 + 1, 3*7 + 2, 3*7 + 3, 3*7 + 4, // 下の壁
		}
		for _, idx := range wallPositions {
			mp.Tiles[idx] = oapi.Tile{Id: "wall", Name: "wall", BlockPass: true}
		}

		planner := EnvironmentPlanner{}
		err := planner.PlanMeta(mp)
		require.NoError(t, err)

		// 囲まれた内部は屋内
		insideIdx := 2*7 + 2 // (2, 2)
		assert.Equal(t, oapi.Indoor, mp.Environments[insideIdx].Shelter, "囲まれた内部は屋内")

		// 外部は屋外
		outsideIdx := 0 // (0, 0)
		assert.Equal(t, oapi.Outdoor, mp.Environments[outsideIdx].Shelter, "外部は屋外")
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
		for i := range 9 {
			mp.Tiles[i] = oapi.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}
		mp.Tiles[4] = oapi.Tile{Id: "water", Name: "water", BlockPass: false, Water: new(true)}

		planner := EnvironmentPlanner{}
		result := planner.calcWater(mp, 4)

		assert.Equal(t, oapi.Submerged, result)
	})

	t.Run("水タイルに隣接すると水辺", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]oapi.Tile, 9),
		}
		for i := range 9 {
			mp.Tiles[i] = oapi.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}
		mp.Tiles[4] = oapi.Tile{Id: "water", Name: "water", BlockPass: false, Water: new(true)}

		planner := EnvironmentPlanner{}

		// 隣接タイル
		assert.Equal(t, oapi.Nearby, planner.calcWater(mp, 1)) // 上
		assert.Equal(t, oapi.Nearby, planner.calcWater(mp, 3)) // 左
		assert.Equal(t, oapi.Nearby, planner.calcWater(mp, 5)) // 右
		assert.Equal(t, oapi.Nearby, planner.calcWater(mp, 7)) // 下
	})

	t.Run("水タイルから離れているとなし", func(t *testing.T) {
		t.Parallel()

		mp := &MetaPlan{
			Level: newTestLevel(3, 3),
			Tiles: make([]oapi.Tile, 9),
		}
		for i := range 9 {
			mp.Tiles[i] = oapi.Tile{Id: "floor", Name: "floor", BlockPass: false}
		}

		planner := EnvironmentPlanner{}
		result := planner.calcWater(mp, 4)

		assert.Equal(t, oapi.WaterExposure(""), result)
	})
}

func TestEnvironmentPlanner_PlanMeta_植生はタイルの行の値を使う(t *testing.T) {
	t.Parallel()

	grass := oapi.Grass
	mp := &MetaPlan{
		Level: newTestLevel(2, 1),
		Tiles: []oapi.Tile{
			{Id: "grass_floor", Name: "grass_floor", Foliage: &grass},
			{Id: "floor", Name: "floor"},
		},
	}
	require.NoError(t, EnvironmentPlanner{}.PlanMeta(mp))

	assert.Equal(t, []oapi.Foliage{oapi.Grass, ""}, []oapi.Foliage{mp.Environments[0].Foliage, mp.Environments[1].Foliage})
}

func newTestLevel(width, height int) gc.Level {
	return gc.Level{
		TileWidth:  consts.Tile(width),
		TileHeight: consts.Tile(height),
	}
}
