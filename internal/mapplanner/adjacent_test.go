package mapplanner

import (
	"testing"

	"github.com/stretchr/testify/assert"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

func TestPlanData_AdjacentAnyFloor(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成
	width, height := consts.Tile(5), consts.Tile(5)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]oapi.Tile, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]gc.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 中央(2,2)を床にする
	centerIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 2, Y: 2})
	planData.Tiles[centerIdx] = planData.GetTile("floor")

	// テストケース1: 直交する隣接タイルは床を検出する
	upIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 2})    // 上
	downIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 3, Y: 2})  // 下
	leftIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 2, Y: 1})  // 左
	rightIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 2, Y: 3}) // 右

	assert.True(t, planData.AdjacentAnyFloor(upIdx), "上の隣接タイルで床を検出できていない")
	assert.True(t, planData.AdjacentAnyFloor(downIdx), "下の隣接タイルで床を検出できていない")
	assert.True(t, planData.AdjacentAnyFloor(leftIdx), "左の隣接タイルで床を検出できていない")
	assert.True(t, planData.AdjacentAnyFloor(rightIdx), "右の隣接タイルで床を検出できていない")

	// テストケース2: 斜めの隣接タイルも床を検出する
	diagUpLeftIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})    // 左上
	diagUpRightIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 3})   // 右上
	diagDownLeftIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 3, Y: 1})  // 左下
	diagDownRightIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 3, Y: 3}) // 右下

	assert.True(t, planData.AdjacentAnyFloor(diagUpLeftIdx), "斜め左上の隣接タイルで床を検出できていない")
	assert.True(t, planData.AdjacentAnyFloor(diagUpRightIdx), "斜め右上の隣接タイルで床を検出できていない")
	assert.True(t, planData.AdjacentAnyFloor(diagDownLeftIdx), "斜め左下の隣接タイルで床を検出できていない")
	assert.True(t, planData.AdjacentAnyFloor(diagDownRightIdx), "斜め右下の隣接タイルで床を検出できていない")

	// テストケース3: 離れたタイルは床を検出しない
	farIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 0, Y: 0}) // 離れた位置
	assert.False(t, planData.AdjacentAnyFloor(farIdx), "離れたタイルで床を誤検出している")
}

func TestPlanData_AdjacentAnyFloor_WithWarpTiles(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成
	width, height := consts.Tile(5), consts.Tile(5)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]oapi.Tile, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]gc.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 床タイルを配置
	floorIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 2, Y: 2})
	planData.Tiles[floorIdx] = planData.GetTile("floor")

	// 床タイルに隣接する場所から床の検出をテスト
	adjacentIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 2}) // (2,2)の床タイルの左隣
	assert.True(t, planData.AdjacentAnyFloor(adjacentIdx), "床タイルに隣接する位置で隣接床検出が失敗")

	adjacentEscapeIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 3}) // (2,3)の床タイルの左隣
	assert.True(t, planData.AdjacentAnyFloor(adjacentEscapeIdx), "床タイルに隣接する位置で隣接床検出が失敗")
}
