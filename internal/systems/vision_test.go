package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeTileRenderMap(t *testing.T) {
	t.Parallel()

	t.Run("視界内タイルはTileRenderVisibleになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}
		query.GetDungeon(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderVisible{}, result[grid])
	})

	t.Run("記憶済みだが見えないタイルはTileRenderRememberedになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 3, Y: 3}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}
		query.GetDungeon(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderRemembered{}, result[grid])
	})

	t.Run("未探索かつ不可視のタイルはマップに含まれない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 10, Y: 10}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

		result := computeTileRenderMap(world, nil)

		assert.NotContains(t, result, grid)
	})

	t.Run("光源があるタイルは光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		lights := map[gc.GridElement]gc.LightInfo{
			grid: {
				Darkness: 0.5,
				Color:    color.RGBA{R: 255, G: 200, B: 100, A: 255},
			},
		}

		result := computeTileRenderMap(world, lights)

		v, ok := result[grid].(TileRenderVisible)
		require.True(t, ok, "型が TileRenderVisible であるべき")
		assert.Equal(t, color.RGBA{R: 255, G: 200, B: 100, A: 255}, v.LightColor)
	})

}

func TestComputeTileRenderMap_DarknessValues(t *testing.T) {
	t.Parallel()

	t.Run("視界内タイルにはDarknessVisibleが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 5, Y: 5}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		result := computeTileRenderMap(world, nil)

		v, ok := result[grid].(TileRenderVisible)
		require.True(t, ok, "型が TileRenderVisible であるべき")
		assert.Equal(t, DarknessVisible, v.Darkness)
	})

	t.Run("記憶済みタイルにはDarknessRememberedが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 3, Y: 3}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}
		query.GetDungeon(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		v, ok := result[grid].(TileRenderRemembered)
		require.True(t, ok, "型が TileRenderRemembered であるべき")
		assert.Equal(t, DarknessRemembered, v.Darkness)
	})
}

func TestComputeTileRenderMap_VisibleOverridesRemembered(t *testing.T) {
	t.Parallel()

	// 可視タイルが記憶済みタイルより優先されることを保証する
	world := testutil.InitTestWorld(t)
	grid := gc.GridElement{X: 5, Y: 5}
	query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}
	query.GetDungeon(world).ExploredTiles[grid] = true

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
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		lights := map[gc.GridElement]gc.LightInfo{
			grid: {
				Darkness: 1.0,
				Color:    color.RGBA{R: 255, G: 255, B: 255, A: 255},
			},
		}

		result := computeTileRenderMap(world, lights)

		v, ok := result[grid].(TileRenderVisible)
		require.True(t, ok, "型が TileRenderVisible であるべき")
		assert.Equal(t, color.RGBA{}, v.LightColor,
			"Darkness=1.0の光源では光源色が設定されない")
	})

	t.Run("光源Darkness=0.99では光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{X: 7, Y: 7}
		query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		lights := map[gc.GridElement]gc.LightInfo{
			grid: {
				Darkness: 0.99,
				Color:    color.RGBA{R: 200, G: 150, B: 100, A: 255},
			},
		}

		result := computeTileRenderMap(world, lights)

		v, ok := result[grid].(TileRenderVisible)
		require.True(t, ok, "型が TileRenderVisible であるべき")
		assert.Equal(t, color.RGBA{R: 200, G: 150, B: 100, A: 255}, v.LightColor)
	})
}

func TestComputeTileRenderMap_EmptyState(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{}

	result := computeTileRenderMap(world, nil)

	assert.Empty(t, result)
}

func TestComputeTileRenderMap_MixedTileStates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	visible := gc.GridElement{X: 1, Y: 1}
	remembered := gc.GridElement{X: 2, Y: 2}
	unknown := gc.GridElement{X: 3, Y: 3}

	query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{visible: true}
	query.GetDungeon(world).ExploredTiles[visible] = true
	query.GetDungeon(world).ExploredTiles[remembered] = true

	result := computeTileRenderMap(world, nil)

	assert.Len(t, result, 2, "可視1+記憶済み1=2タイルがマップに含まれる")
	assert.IsType(t, TileRenderVisible{}, result[visible])
	assert.IsType(t, TileRenderRemembered{}, result[remembered])
	assert.NotContains(t, result, unknown)
}

func TestComputeTileRenderMap_OutOfBoundsIncluded(t *testing.T) {
	t.Parallel()

	// computeTileRenderMapは境界チェックを行わない。
	// マップ外座標の除外はrenderDarkness側で行う
	world := testutil.InitTestWorld(t)
	insideGrid := gc.GridElement{X: 1, Y: 1}
	outsideGrid := gc.GridElement{X: 99, Y: 99}

	query.GetDungeon(world).VisibleTiles = map[gc.GridElement]bool{
		insideGrid:  true,
		outsideGrid: true,
	}

	result := computeTileRenderMap(world, nil)

	assert.Contains(t, result, insideGrid)
	assert.IsType(t, TileRenderVisible{}, result[insideGrid])
	assert.Contains(t, result, outsideGrid)
	assert.IsType(t, TileRenderVisible{}, result[outsideGrid])
}

func TestIsInMapBounds(t *testing.T) {
	t.Parallel()

	level := gc.Level{TileWidth: 10, TileHeight: 5}

	assert.True(t, isInMapBounds(gc.GridElement{X: 0, Y: 0}, level))
	assert.True(t, isInMapBounds(gc.GridElement{X: 9, Y: 4}, level))
	assert.False(t, isInMapBounds(gc.GridElement{X: 10, Y: 0}, level))
	assert.False(t, isInMapBounds(gc.GridElement{X: 0, Y: 5}, level))
	assert.False(t, isInMapBounds(gc.GridElement{X: -1, Y: 0}, level))
}

func TestBuildBlockViewIndex(t *testing.T) {
	t.Parallel()

	t.Run("BlockViewを持つエンティティがインデックスに含まれる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// BlockView付きの壁タイルを生成する
		wallGrid := gc.GridElement{X: 3, Y: 4}
		wallEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(wallEntity, &wallGrid)
		world.Components.BlockView.Add(wallEntity, &gc.BlockView{})

		// BlockViewなしの床タイルを生成する
		floorGrid := gc.GridElement{X: 5, Y: 6}
		floorEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(floorEntity, &floorGrid)

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

func TestInvalidateOnFloorChange(t *testing.T) {
	t.Parallel()

	t.Run("フロアが変わると壁依存の内部キャッシュを破棄する", func(t *testing.T) {
		t.Parallel()
		vs := NewVisionSystem()
		vs.isInitialized = true
		vs.raycastCache[raycastCacheKey{Player: consts.Coord[int]{X: 1}}] = true
		vs.lastDepth = 1
		vs.lastDefinitionName = "old"

		dungeon := gc.NewDungeon()
		dungeon.Depth = 2
		dungeon.DefinitionName = "new"
		dungeon.LightSourceCache[gc.GridElement{X: 99, Y: 99}] = gc.LightInfo{Darkness: 0.5}

		vs.invalidateOnFloorChange(dungeon)

		assert.False(t, vs.isInitialized)
		assert.Empty(t, vs.raycastCache)
		assert.Empty(t, dungeon.LightSourceCache)
		assert.Equal(t, 2, vs.lastDepth)
		assert.Equal(t, "new", vs.lastDefinitionName)
	})

	t.Run("同一フロアではキャッシュを保持する", func(t *testing.T) {
		t.Parallel()
		vs := NewVisionSystem()
		vs.isInitialized = true
		vs.raycastCache[raycastCacheKey{Player: consts.Coord[int]{X: 1}}] = true
		vs.lastDepth = 3
		vs.lastDefinitionName = "same"

		dungeon := gc.NewDungeon()
		dungeon.Depth = 3
		dungeon.DefinitionName = "same"

		vs.invalidateOnFloorChange(dungeon)

		assert.True(t, vs.isInitialized)
		assert.NotEmpty(t, vs.raycastCache)
	})
}
