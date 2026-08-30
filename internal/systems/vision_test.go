package systems

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeTileRenderMap(t *testing.T) {
	t.Parallel()

	t.Run("視界内タイルはTileRenderVisibleになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{grid: true}
		query.GetCurrentStageField(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderVisible{}, result[grid])
	})

	t.Run("記憶済みだが見えないタイルはTileRenderRememberedになる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{}
		query.GetCurrentStageField(world).ExploredTiles[grid] = true

		result := computeTileRenderMap(world, nil)

		assert.Contains(t, result, grid)
		assert.IsType(t, TileRenderRemembered{}, result[grid])
	})

	t.Run("未探索かつ不可視のタイルはマップに含まれない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{}

		result := computeTileRenderMap(world, nil)

		assert.NotContains(t, result, grid)
	})

	t.Run("光源があるタイルは光源色が設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

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
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

		result := computeTileRenderMap(world, nil)

		v, ok := result[grid].(TileRenderVisible)
		require.True(t, ok, "型が TileRenderVisible であるべき")
		assert.Equal(t, DarknessVisible, v.Darkness)
	})

	t.Run("記憶済みタイルにはDarknessRememberedが設定される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{}
		query.GetCurrentStageField(world).ExploredTiles[grid] = true

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
	grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
	query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{grid: true}
	query.GetCurrentStageField(world).ExploredTiles[grid] = true

	result := computeTileRenderMap(world, nil)

	assert.IsType(t, TileRenderVisible{}, result[grid],
		"可視+記憶済みのタイルはTileRenderVisibleになる")
}

func TestComputeTileRenderMap_LightSourceBoundary(t *testing.T) {
	t.Parallel()

	t.Run("光源Darkness=1.0では光源色が設定されない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

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
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 7, Y: 7}}
		query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{grid: true}

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
	query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{}

	result := computeTileRenderMap(world, nil)

	assert.Empty(t, result)
}

func TestComputeTileRenderMap_MixedTileStates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	visible := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}}
	remembered := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 2, Y: 2}}
	unknown := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}}

	query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{visible: true}
	query.GetCurrentStageField(world).ExploredTiles[visible] = true
	query.GetCurrentStageField(world).ExploredTiles[remembered] = true

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
	insideGrid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}}
	outsideGrid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 99, Y: 99}}

	query.GetVisionState(world).VisibleTiles = map[gc.GridElement]bool{
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

	assert.True(t, isInMapBounds(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}}, level))
	assert.True(t, isInMapBounds(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 9, Y: 4}}, level))
	assert.False(t, isInMapBounds(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 0}}, level))
	assert.False(t, isInMapBounds(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 5}}, level))
	assert.False(t, isInMapBounds(gc.GridElement{Coord: consts.Coord[consts.Tile]{X: -1, Y: 0}}, level))
}

func TestBuildBlockViewIndex(t *testing.T) {
	t.Parallel()

	t.Run("BlockViewを持つエンティティがインデックスに含まれる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// BlockView付きの壁タイルを生成する
		wallGrid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 4}}
		wallEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(wallEntity, &wallGrid)
		world.Components.BlockView.Add(wallEntity, &gc.BlockView{})

		// BlockViewなしの床タイルを生成する
		floorGrid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 6}}
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

func TestCalculateTileVisibility_チャンク境界をまたいで視線が通り壁で遮られる(t *testing.T) {
	t.Parallel()

	// オーバーワールドは複数チャンクを1つのステージに束ねるため、視界はチャンク境界を透過する。
	// 視界にはチャンク固有の境界ロジックが無く絶対座標で動くことを、境界座標 x=chunkW を跨ぐ
	// 水平視線で固定する。遮蔽が無ければ向こう側が見え、境界上の壁で遮られる。
	const chunkW consts.Tile = 30 // 東西チャンク境界の座標
	playerTile := consts.Coord[consts.Tile]{X: chunkW - 2, Y: 10}
	playerPos := consts.TileCenterToWorld(playerTile)
	radius := consts.WorldPixel(8 * int(consts.TileSize))
	target := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: chunkW + 3, Y: 10}} // 隣チャンク側

	lookup := func(vis []TileVisibility, g gc.GridElement) (visible bool, found bool) {
		for _, v := range vis {
			if v.Col == int(g.X) && v.Row == int(g.Y) {
				return v.Visible, true
			}
		}
		return false, false
	}

	t.Run("遮蔽が無ければ隣チャンク側が見える", func(t *testing.T) {
		t.Parallel()
		vis := calculateTileVisibilityWithDistance(playerPos, radius, map[gc.GridElement]bool{})
		visible, found := lookup(vis, target)
		require.True(t, found, "対象タイルが視界走査範囲に入る")
		assert.True(t, visible, "境界の向こう側のタイルが見える")
	})

	t.Run("境界上の壁で視線が遮られる", func(t *testing.T) {
		t.Parallel()
		blockIndex := map[gc.GridElement]bool{
			{Coord: consts.Coord[consts.Tile]{X: chunkW, Y: 10}}: true, // 境界 x=chunkW 上の壁
		}
		vis := calculateTileVisibilityWithDistance(playerPos, radius, blockIndex)
		visible, found := lookup(vis, target)
		require.True(t, found, "対象タイルが視界走査範囲に入る")
		assert.False(t, visible, "境界上の壁の向こうは見えない")
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
			{Coord: consts.Coord[consts.Tile]{X: 2, Y: 2}}: true,
		}

		assert.False(t, bresenhamLineOfSight(0, 0, 5, 5, blockIndex))
	})

	t.Run("ターゲット位置の壁は遮蔽しない", func(t *testing.T) {
		t.Parallel()
		// ターゲット自体が壁でも到達判定が先なので見える
		blockIndex := map[gc.GridElement]bool{
			{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}}: true,
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

func TestCalculateLightSourceDarkness_壁が光を遮る(t *testing.T) {
	t.Parallel()

	// 光源を (5,5) に置き Radius 10 とする。(7,5) に壁を立て、壁の手前 (6,5) は照らされ、
	// 壁の裏 (9,5) は照らされないことを固定する。光の伝播が視界と同じ遮蔽判定を通す回帰テスト。
	setup := func(t *testing.T) w.World {
		t.Helper()
		world := testutil.InitTestWorld(t)
		lightGrid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}
		lightEntity := world.ECS.NewEntity()
		world.Components.GridElement.Add(lightEntity, &lightGrid)
		world.Components.LightSource.Add(lightEntity, &gc.LightSource{
			Radius:  10,
			Color:   color.RGBA{R: 255, G: 220, B: 150, A: 255},
			Enabled: true,
		})
		return world
	}
	wall := map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 7, Y: 5}}: true,
	}

	t.Run("壁の手前のタイルは照らされる", func(t *testing.T) {
		t.Parallel()
		world := setup(t)
		info := calculateLightSourceDarkness(world, consts.Coord[int]{X: 6, Y: 5}, wall, 0.0, [3]float64{1, 1, 1})
		assert.Less(t, info.Darkness, 1.0, "手前のタイルは光が届き暗闇が解消される")
	})

	t.Run("壁の裏のタイルは照らされない", func(t *testing.T) {
		t.Parallel()
		world := setup(t)
		info := calculateLightSourceDarkness(world, consts.Coord[int]{X: 9, Y: 5}, wall, 0.0, [3]float64{1, 1, 1})
		assert.Equal(t, 1.0, info.Darkness, "壁の裏は光が遮られ完全に暗いまま")
		// 光が寄与しないと RGB は 0 のまま。A は常に 255 が入るので RGB だけを見る。
		// Darkness=1.0 なら描画側が色を無視するため、この色は実効に影響しない
		assert.Equal(t, color.RGBA{A: 255}, info.Color, "光が届かないので色は乗らない。RGBは0")
	})

	t.Run("遮蔽が無ければ同じタイルが照らされる", func(t *testing.T) {
		t.Parallel()
		world := setup(t)
		// 壁を除いた同座標で照らされることを示し、暗いのは距離でなく遮蔽が原因だと確定する
		info := calculateLightSourceDarkness(world, consts.Coord[int]{X: 9, Y: 5}, map[gc.GridElement]bool{}, 0.0, [3]float64{1, 1, 1})
		assert.Less(t, info.Darkness, 1.0, "壁が無ければ壁の裏と同じ位置でも照らされる")
	})
}

// TestCalculateLightSourceDarkness_明るさの合成 は明るさが距離減衰・加算・環境光で
// 決まることを固定する。遮蔽テストは「届くか」しか見ないので、減衰形状と合成を別に押さえる。
func TestCalculateLightSourceDarkness_明るさの合成(t *testing.T) {
	t.Parallel()

	noWall := map[gc.GridElement]bool{}
	addLight := func(world w.World, x, y int) {
		grid := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}}
		e := world.ECS.NewEntity()
		world.Components.GridElement.Add(e, &grid)
		world.Components.LightSource.Add(e, &gc.LightSource{
			Radius:  10,
			Color:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
			Enabled: true,
		})
	}

	t.Run("光源に近いほど明るい", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		addLight(world, 5, 5)
		near := calculateLightSourceDarkness(world, consts.Coord[int]{X: 6, Y: 5}, noWall, 0.0, [3]float64{1, 1, 1})
		far := calculateLightSourceDarkness(world, consts.Coord[int]{X: 13, Y: 5}, noWall, 0.0, [3]float64{1, 1, 1})
		assert.Less(t, near.Darkness, far.Darkness, "近いタイルの方が暗さが小さい")
	})

	t.Run("光源が重なると明るくなる", func(t *testing.T) {
		t.Parallel()
		// プラトー外の距離8で、1灯と2灯を比べる。加算されるので2灯の方が暗さが小さい
		one := testutil.InitTestWorld(t)
		addLight(one, 5, 5)
		darknessOne := calculateLightSourceDarkness(one, consts.Coord[int]{X: 13, Y: 5}, noWall, 0.0, [3]float64{1, 1, 1}).Darkness

		two := testutil.InitTestWorld(t)
		addLight(two, 5, 5)
		addLight(two, 21, 5)
		darknessTwo := calculateLightSourceDarkness(two, consts.Coord[int]{X: 13, Y: 5}, noWall, 0.0, [3]float64{1, 1, 1}).Darkness

		assert.Less(t, darknessTwo, darknessOne, "2灯に照らされると加算で明るくなる")
	})

	t.Run("光の届かないタイルは環境光で決まる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// 光源を置かない。暗さ = 1 - 環境光
		info := calculateLightSourceDarkness(world, consts.Coord[int]{X: 5, Y: 5}, noWall, 0.3, [3]float64{1, 1, 1})
		assert.InDelta(t, 0.7, info.Darkness, 1e-9, "光が無ければ暗さは環境光で決まる")
	})
}
