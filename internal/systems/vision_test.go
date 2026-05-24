package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestComputeTileRenderMap(t *testing.T) {
	t.Parallel()

	t.Run("視界内タイルはTileRenderVisibleになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}
		world.Resources.Dungeon.ExploredTiles[grid] = true

		result := computeTileRenderMap(world)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderVisible{}, result[grid])
	})

	t.Run("記憶済みだが見えないタイルはTileRenderRememberedになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 3, Y: 3}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{}
		world.Resources.Dungeon.ExploredTiles[grid] = true

		result := computeTileRenderMap(world)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderRemembered{}, result[grid])
	})

	t.Run("未探索かつ不可視のタイルはマップに含まれない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 10, Y: 10}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{}

		result := computeTileRenderMap(world)

		assert.NotContains(t, result, grid)
	})

	t.Run("光源があるタイルは光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}

		lightSourceCache[grid] = LightInfo{
			Darkness: 0.5,
			Color:    color.RGBA{R: 255, G: 200, B: 100, A: 255},
		}
		t.Cleanup(func() { delete(lightSourceCache, grid) })

		result := computeTileRenderMap(world)

		v, ok := result[grid].(TileRenderVisible)
		assert.True(t, ok)
		assert.Equal(t, color.RGBA{R: 255, G: 200, B: 100, A: 255}, v.LightColor)
	})

	t.Run("暗闇フロアで光源外のタイルはTileRenderDarkになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon.Dark = true

		litGrid := gc.GridElement{X: 5, Y: 5}
		darkGrid := gc.GridElement{X: 15, Y: 15}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{litGrid: true}
		world.Resources.Dungeon.ExploredTiles[litGrid] = true

		lightSourceCache[darkGrid] = LightInfo{Darkness: 1.0}
		t.Cleanup(func() { delete(lightSourceCache, darkGrid) })

		result := computeTileRenderMap(world)

		assert.IsType(t, TileRenderVisible{}, result[litGrid])
		assert.IsType(t, TileRenderDark{}, result[darkGrid])
	})

	t.Run("明るいフロアではTileRenderDarkが生成されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon.Dark = false

		grid := gc.GridElement{X: 10, Y: 10}
		lightSourceCache[grid] = LightInfo{Darkness: 1.0}
		t.Cleanup(func() { delete(lightSourceCache, grid) })

		result := computeTileRenderMap(world)

		assert.NotContains(t, result, grid)
	})
}

func TestComputeTileRenderMap_DarknessValues(t *testing.T) {
	t.Parallel()

	t.Run("視界内タイルにはDarknessVisibleが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}

		result := computeTileRenderMap(world)

		v := result[grid].(TileRenderVisible)
		assert.Equal(t, DarknessVisible, v.Darkness)
	})

	t.Run("暗闇タイルにはDarknessDarkが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		world.Resources.Dungeon.Dark = true
		grid := gc.GridElement{X: 10, Y: 10}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{}

		lightSourceCache[grid] = LightInfo{Darkness: 1.0}
		t.Cleanup(func() { delete(lightSourceCache, grid) })

		result := computeTileRenderMap(world)

		v := result[grid].(TileRenderDark)
		assert.Equal(t, DarknessDark, v.Darkness)
	})

	t.Run("記憶済みタイルにはDarknessRememberedが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 3, Y: 3}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{}
		world.Resources.Dungeon.ExploredTiles[grid] = true

		result := computeTileRenderMap(world)

		v := result[grid].(TileRenderRemembered)
		assert.Equal(t, DarknessRemembered, v.Darkness)
	})
}

func TestComputeTileRenderMap_VisibleOverridesRemembered(t *testing.T) {
	t.Parallel()

	// 可視タイルが記憶済みタイルより優先されることを保証する
	world := testutil.InitTestWorld(t)
	grid := gc.GridElement{X: 5, Y: 5}
	world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}
	world.Resources.Dungeon.ExploredTiles[grid] = true

	result := computeTileRenderMap(world)

	assert.IsType(t, TileRenderVisible{}, result[grid],
		"可視+記憶済みのタイルはTileRenderVisibleになる")
}

func TestComputeTileRenderMap_LightSourceBoundary(t *testing.T) {
	t.Parallel()

	t.Run("光源Darkness=1.0では光源色が設定されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}

		lightSourceCache[grid] = LightInfo{
			Darkness: 1.0,
			Color:    color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}
		t.Cleanup(func() { delete(lightSourceCache, grid) })

		result := computeTileRenderMap(world)

		v, ok := result[grid].(TileRenderVisible)
		assert.True(t, ok)
		assert.Equal(t, color.RGBA{}, v.LightColor,
			"Darkness=1.0の光源では光源色が設定されない")
	})

	t.Run("光源Darkness=0.99では光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 7, Y: 7}
		world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{grid: true}

		lightSourceCache[grid] = LightInfo{
			Darkness: 0.99,
			Color:    color.RGBA{R: 200, G: 150, B: 100, A: 255},
		}
		t.Cleanup(func() { delete(lightSourceCache, grid) })

		result := computeTileRenderMap(world)

		v, ok := result[grid].(TileRenderVisible)
		assert.True(t, ok)
		assert.Equal(t, color.RGBA{R: 200, G: 150, B: 100, A: 255}, v.LightColor)
	})
}

func TestComputeTileRenderMap_EmptyState(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{}

	result := computeTileRenderMap(world)

	assert.Empty(t, result)
}

func TestComputeTileRenderMap_MixedTileStates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	visible := gc.GridElement{X: 1, Y: 1}
	remembered := gc.GridElement{X: 2, Y: 2}
	unknown := gc.GridElement{X: 3, Y: 3}

	world.Resources.Dungeon.VisibleTiles = map[gc.GridElement]bool{visible: true}
	world.Resources.Dungeon.ExploredTiles[visible] = true
	world.Resources.Dungeon.ExploredTiles[remembered] = true

	result := computeTileRenderMap(world)

	assert.Len(t, result, 2, "可視1+記憶済み1=2タイルがマップに含まれる")
	assert.IsType(t, TileRenderVisible{}, result[visible])
	assert.IsType(t, TileRenderRemembered{}, result[remembered])
	assert.NotContains(t, result, unknown)
}

func TestClearVisionCaches(t *testing.T) {
	t.Parallel()

	playerPositionCache.isInitialized = true
	playerPositionCache.visibilityData = map[string]TileVisibility{
		"0,0": {Row: 0, Col: 0, Visible: true},
	}
	grid := gc.GridElement{X: 99, Y: 99}
	lightSourceCache[grid] = LightInfo{Darkness: 0.5}

	ClearVisionCaches()

	assert.False(t, playerPositionCache.isInitialized)
	assert.Nil(t, playerPositionCache.visibilityData)
	assert.Empty(t, lightSourceCache)
}
