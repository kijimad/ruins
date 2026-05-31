package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
)

func TestComputeTileRenderMap(t *testing.T) {
	t.Parallel()

	t.Run("視界内タイルはTileRenderVisibleになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}
		worldhelper.GetDungeon(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderVisible{}, result[grid])
	})

	t.Run("記憶済みだが見えないタイルはTileRenderRememberedになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 3, Y: 3}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}
		worldhelper.GetDungeon(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderRemembered{}, result[grid])
	})

	t.Run("未探索かつ不可視のタイルはマップに含まれない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 10, Y: 10}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

		result := computeTileRenderMap(world, nil)

		assert.NotContains(t, result, grid)
	})

	t.Run("光源があるタイルは光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		lights := map[gc.GridElement]LightInfo{
			grid: {
				Darkness: 0.5,
				Color:    color.RGBA{R: 255, G: 200, B: 100, A: 255},
			},
		}

		result := computeTileRenderMap(world, lights)

		v, ok := result[grid].(TileRenderVisible)
		assert.True(t, ok)
		assert.Equal(t, color.RGBA{R: 255, G: 200, B: 100, A: 255}, v.LightColor)
	})

	t.Run("暗闇フロアで光源外のタイルはTileRenderDarkになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.GetDungeon(world).Dark = true

		litGrid := gc.GridElement{X: 5, Y: 5}
		darkGrid := gc.GridElement{X: 15, Y: 15}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{litGrid: true}
		worldhelper.GetDungeon(world).ExploredTiles[litGrid] = true

		lights := map[gc.GridElement]LightInfo{
			darkGrid: {Darkness: 1.0},
		}

		result := computeTileRenderMap(world, lights)

		assert.IsType(t, TileRenderVisible{}, result[litGrid])
		assert.IsType(t, TileRenderDark{}, result[darkGrid])
	})

	t.Run("明るいフロアではTileRenderDarkが生成されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.GetDungeon(world).Dark = false

		grid := gc.GridElement{X: 10, Y: 10}
		lights := map[gc.GridElement]LightInfo{
			grid: {Darkness: 1.0},
		}

		result := computeTileRenderMap(world, lights)

		assert.NotContains(t, result, grid)
	})
}

func TestComputeTileRenderMap_DarknessValues(t *testing.T) {
	t.Parallel()

	t.Run("視界内タイルにはDarknessVisibleが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		result := computeTileRenderMap(world, nil)

		v := result[grid].(TileRenderVisible)
		assert.Equal(t, DarknessVisible, v.Darkness)
	})

	t.Run("暗闇タイルにはDarknessDarkが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.GetDungeon(world).Dark = true
		grid := gc.GridElement{X: 10, Y: 10}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

		lights := map[gc.GridElement]LightInfo{
			grid: {Darkness: 1.0},
		}

		result := computeTileRenderMap(world, lights)

		v := result[grid].(TileRenderDark)
		assert.Equal(t, DarknessDark, v.Darkness)
	})

	t.Run("記憶済みタイルにはDarknessRememberedが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 3, Y: 3}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}
		worldhelper.GetDungeon(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		v := result[grid].(TileRenderRemembered)
		assert.Equal(t, DarknessRemembered, v.Darkness)
	})
}

func TestComputeTileRenderMap_VisibleOverridesRemembered(t *testing.T) {
	t.Parallel()

	// 可視タイルが記憶済みタイルより優先されることを保証する
	world := testutil.InitTestWorld(t)
	grid := gc.GridElement{X: 5, Y: 5}
	worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}
	worldhelper.GetDungeon(world).ExploredTiles[grid] = true

	result := computeTileRenderMap(world, nil)

	assert.IsType(t, TileRenderVisible{}, result[grid],
		"可視+記憶済みのタイルはTileRenderVisibleになる")
}

func TestComputeTileRenderMap_LightSourceBoundary(t *testing.T) {
	t.Parallel()

	t.Run("光源Darkness=1.0では光源色が設定されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		lights := map[gc.GridElement]LightInfo{
			grid: {
				Darkness: 1.0,
				Color:    color.RGBA{R: 255, G: 255, B: 255, A: 255},
			},
		}

		result := computeTileRenderMap(world, lights)

		v, ok := result[grid].(TileRenderVisible)
		assert.True(t, ok)
		assert.Equal(t, color.RGBA{}, v.LightColor,
			"Darkness=1.0の光源では光源色が設定されない")
	})

	t.Run("光源Darkness=0.99では光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 7, Y: 7}
		worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		lights := map[gc.GridElement]LightInfo{
			grid: {
				Darkness: 0.99,
				Color:    color.RGBA{R: 200, G: 150, B: 100, A: 255},
			},
		}

		result := computeTileRenderMap(world, lights)

		v, ok := result[grid].(TileRenderVisible)
		assert.True(t, ok)
		assert.Equal(t, color.RGBA{R: 200, G: 150, B: 100, A: 255}, v.LightColor)
	})
}

func TestComputeTileRenderMap_EmptyState(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

	result := computeTileRenderMap(world, nil)

	assert.Empty(t, result)
}

func TestComputeTileRenderMap_MixedTileStates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	visible := gc.GridElement{X: 1, Y: 1}
	remembered := gc.GridElement{X: 2, Y: 2}
	unknown := gc.GridElement{X: 3, Y: 3}

	worldhelper.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{visible: true}
	worldhelper.GetDungeon(world).ExploredTiles[visible] = true
	worldhelper.GetDungeon(world).ExploredTiles[remembered] = true

	result := computeTileRenderMap(world, nil)

	assert.Len(t, result, 2, "可視1+記憶済み1=2タイルがマップに含まれる")
	assert.IsType(t, TileRenderVisible{}, result[visible])
	assert.IsType(t, TileRenderRemembered{}, result[remembered])
	assert.NotContains(t, result, unknown)
}

func TestBuildBlockViewIndex(t *testing.T) {
	t.Parallel()

	t.Run("BlockViewを持つエンティティがインデックスに含まれる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// BlockView付きの壁タイルを生成する
		wallGrid := gc.GridElement{X: 3, Y: 4}
		world.Manager.NewEntity().
			AddComponent(world.Components.GridElement, &wallGrid).
			AddComponent(world.Components.BlockView, &gc.BlockView{})

		// BlockViewなしの床タイルを生成する
		floorGrid := gc.GridElement{X: 5, Y: 6}
		world.Manager.NewEntity().
			AddComponent(world.Components.GridElement, &floorGrid)

		index := buildBlockViewIndex(world)

		assert.True(t, index[wallGrid], "壁タイルがインデックスに含まれる")
		assert.False(t, index[floorGrid], "床タイルはインデックスに含まれない")
	})

	t.Run("BlockViewエンティティがない場合は空マップを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		index := buildBlockViewIndex(world)

		assert.Empty(t, index)
	})
}

func TestBresenhamLineOfSight(t *testing.T) {
	t.Parallel()

	t.Run("遮蔽物がなければ見える", func(t *testing.T) {
		t.Parallel()
		blockIndex := map[gc.GridElement]bool{}

		assert.True(t, bresenhamLineOfSight(0, 0, 5, 5, blockIndex))
	})

	t.Run("途中に壁があれば見えない", func(t *testing.T) {
		t.Parallel()
		blockIndex := map[gc.GridElement]bool{
			{X: 2, Y: 2}: true,
		}

		assert.False(t, bresenhamLineOfSight(0, 0, 5, 5, blockIndex))
	})

	t.Run("ターゲット位置の壁は遮蔽しない", func(t *testing.T) {
		t.Parallel()
		// ターゲット自体が壁でも到達判定が先なので見える
		blockIndex := map[gc.GridElement]bool{
			{X: 3, Y: 3}: true,
		}

		assert.True(t, bresenhamLineOfSight(0, 0, 3, 3, blockIndex))
	})

	t.Run("隣接タイルは常に見える", func(t *testing.T) {
		t.Parallel()
		// 隣接はbresenhamの最初のステップでターゲット到達する
		blockIndex := map[gc.GridElement]bool{}

		assert.True(t, bresenhamLineOfSight(5, 5, 6, 5, blockIndex))
		assert.True(t, bresenhamLineOfSight(5, 5, 5, 6, blockIndex))
	})
}

func TestClearCaches(t *testing.T) {
	t.Parallel()

	vs := NewVisionSystem()
	vs.isInitialized = true
	vs.visibilityData = map[string]TileVisibility{
		"0,0": {Row: 0, Col: 0, Visible: true},
	}
	grid := gc.GridElement{X: 99, Y: 99}
	vs.lightSourceCache[grid] = LightInfo{Darkness: 0.5}

	vs.ClearCaches()

	assert.False(t, vs.isInitialized)
	assert.Nil(t, vs.visibilityData)
	assert.Empty(t, vs.lightSourceCache)
}
