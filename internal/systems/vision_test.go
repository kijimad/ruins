package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestComputeTileRenderMap(t *testing.T) {
	t.Parallel()

	t.Run("可視タイルはDrawFloorとDrawObjectsが両方true", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}
		world.Resources.Dungeon.ExploredTiles[grid] = true

		result := computeTileRenderMap(world)

		assert.Contains(t, result, grid)
		assert.True(t, result[grid].DrawFloor)
		assert.True(t, result[grid].DrawObjects)
		assert.Equal(t, TileDarknessVisible.DarknessValue(), result[grid].Darkness)
	})

	t.Run("探索済みだが見えないタイルはDrawFloorのみtrue", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 3, Y: 3}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{}
		world.Resources.Dungeon.ExploredTiles[grid] = true

		result := computeTileRenderMap(world)

		assert.Contains(t, result, grid)
		assert.True(t, result[grid].DrawFloor)
		assert.False(t, result[grid].DrawObjects)
		assert.Equal(t, TileDarknessExplored.DarknessValue(), result[grid].Darkness)
	})

	t.Run("未探索かつ不可視のタイルはマップに含まれない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 10, Y: 10}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{}

		result := computeTileRenderMap(world)

		assert.NotContains(t, result, grid)
	})

	t.Run("光源があるタイルはLit暗さと光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}

		// 光源キャッシュを設定する
		lightSourceCache[grid] = LightInfo{
			Darkness: 0.5,
			Color:    color.RGBA{R: 255, G: 200, B: 100, A: 255},
		}
		t.Cleanup(func() { delete(lightSourceCache, grid) })

		result := computeTileRenderMap(world)

		assert.Equal(t, TileDarknessLit.DarknessValue(), result[grid].Darkness)
		assert.Equal(t, color.RGBA{R: 255, G: 200, B: 100, A: 255}, result[grid].LightColor)
	})

	t.Run("暗闇フロアで光源外のタイルはVisibleTilesに含まれない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 暗闇フロアの設定。VisionSystemがVisibleTilesを制御するため、
		// 光源外のタイルはVisibleTilesに入らない
		world.Resources.Dungeon.Dark = true
		litGrid := gc.GridElement{X: 5, Y: 5}
		darkGrid := gc.GridElement{X: 15, Y: 15}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{litGrid: true}
		world.Resources.Dungeon.ExploredTiles[litGrid] = true
		world.Resources.Dungeon.ExploredTiles[darkGrid] = true

		result := computeTileRenderMap(world)

		assert.True(t, result[litGrid].DrawObjects, "光源内タイルはオブジェクトを描画する")
		assert.False(t, result[darkGrid].DrawObjects, "光源外の探索済みタイルはオブジェクトを描画しない")
		assert.True(t, result[darkGrid].DrawFloor, "光源外の探索済みタイルは床のみ描画する")
	})
}

func TestComputeTileRenderMap_PositionIndependence(t *testing.T) {
	t.Parallel()

	// タイル座標の位置に関わらず、同じ条件なら同じ結果を返すことを保証する
	world := testutil.InitTestWorld(t)
	positions := []gc.GridElement{
		{X: 0, Y: 0},
		{X: consts.Tile(49), Y: consts.Tile(49)},
		{X: 25, Y: 25},
	}
	world.Resources.Dungeon.VisibleTiles = make(map[gc.GridElement]bool)
	for _, pos := range positions {
		world.Resources.Dungeon.VisibleTiles[pos] = true
	}

	result := computeTileRenderMap(world)

	for i := 1; i < len(positions); i++ {
		assert.Equal(t, result[positions[0]].DrawFloor, result[positions[i]].DrawFloor)
		assert.Equal(t, result[positions[0]].DrawObjects, result[positions[i]].DrawObjects)
		assert.Equal(t, result[positions[0]].Darkness, result[positions[i]].Darkness)
	}
}

func TestTileDarknessLevelOrdering(t *testing.T) {
	t.Parallel()

	// 暗さの段階が明るい順に並んでいることを保証する
	assert.LessOrEqual(t, TileDarknessLit.DarknessValue(), TileDarknessVisible.DarknessValue())
	assert.Less(t, TileDarknessVisible.DarknessValue(), TileDarknessExplored.DarknessValue())
	assert.Less(t, TileDarknessExplored.DarknessValue(), TileDarknessFull.DarknessValue())
}

func TestTileDarknessExploredNotFullyBlack(t *testing.T) {
	t.Parallel()

	// 探索済みタイルの暗さが完全な黒と区別できることを保証する
	assert.Less(t, TileDarknessExplored.DarknessValue(), TileDarknessFull.DarknessValue(),
		"探索済みタイルは完全な黒より明るくなければならない")

	// ceil量子化でも完全な黒（level DarknessLevels）にならないことを保証する
	darknessLevel := int(TileDarknessExplored.DarknessValue() * float64(DarknessLevels))
	assert.Less(t, darknessLevel, DarknessLevels,
		"探索済みタイルの暗さが量子化で完全な黒(level %d)になってはいけない", DarknessLevels)
}
