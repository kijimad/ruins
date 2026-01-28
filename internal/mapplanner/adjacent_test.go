package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
)

func TestPlanData_AdjacentAnyFloor(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成
	width, height := gc.Tile(5), gc.Tile(5)
	planData := &MetaPlan{
		Level: resources.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]raw.TileRaw, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]resources.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 中央(2,2)を床にする
	centerIdx := planData.Level.XYTileIndex(2, 2)
	planData.Tiles[centerIdx] = planData.GetTile("floor")

	// テストケース1: 直交する隣接タイルは床を検出する
	upIdx := planData.Level.XYTileIndex(1, 2)    // 上
	downIdx := planData.Level.XYTileIndex(3, 2)  // 下
	leftIdx := planData.Level.XYTileIndex(2, 1)  // 左
	rightIdx := planData.Level.XYTileIndex(2, 3) // 右

	if !planData.AdjacentAnyFloor(upIdx) {
		t.Error("上の隣接タイルで床を検出できていない")
	}
	if !planData.AdjacentAnyFloor(downIdx) {
		t.Error("下の隣接タイルで床を検出できていない")
	}
	if !planData.AdjacentAnyFloor(leftIdx) {
		t.Error("左の隣接タイルで床を検出できていない")
	}
	if !planData.AdjacentAnyFloor(rightIdx) {
		t.Error("右の隣接タイルで床を検出できていない")
	}

	// テストケース2: 斜めの隣接タイルも床を検出する
	diagUpLeftIdx := planData.Level.XYTileIndex(1, 1)    // 左上
	diagUpRightIdx := planData.Level.XYTileIndex(1, 3)   // 右上
	diagDownLeftIdx := planData.Level.XYTileIndex(3, 1)  // 左下
	diagDownRightIdx := planData.Level.XYTileIndex(3, 3) // 右下

	if !planData.AdjacentAnyFloor(diagUpLeftIdx) {
		t.Error("斜め左上の隣接タイルで床を検出できていない")
	}
	if !planData.AdjacentAnyFloor(diagUpRightIdx) {
		t.Error("斜め右上の隣接タイルで床を検出できていない")
	}
	if !planData.AdjacentAnyFloor(diagDownLeftIdx) {
		t.Error("斜め左下の隣接タイルで床を検出できていない")
	}
	if !planData.AdjacentAnyFloor(diagDownRightIdx) {
		t.Error("斜め右下の隣接タイルで床を検出できていない")
	}

	// テストケース3: 離れたタイルは床を検出しない
	farIdx := planData.Level.XYTileIndex(0, 0) // 離れた位置
	if planData.AdjacentAnyFloor(farIdx) {
		t.Error("離れたタイルで床を誤検出している")
	}
}

func TestPlanData_AdjacentAnyFloor_WithWarpTiles(t *testing.T) {
	t.Parallel()
	// テスト用のマップを作成
	width, height := gc.Tile(5), gc.Tile(5)
	planData := &MetaPlan{
		Level: resources.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]raw.TileRaw, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]resources.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 床タイルを配置
	floorIdx := planData.Level.XYTileIndex(2, 2)
	planData.Tiles[floorIdx] = planData.GetTile("floor")

	// 床タイルに隣接する場所から床の検出をテスト
	adjacentIdx := planData.Level.XYTileIndex(1, 2) // (2,2)の床タイルの左隣
	if !planData.AdjacentAnyFloor(adjacentIdx) {
		t.Error("床タイルに隣接する位置で隣接床検出が失敗")
	}

	adjacentEscapeIdx := planData.Level.XYTileIndex(1, 3) // (2,3)の床タイルの左隣
	if !planData.AdjacentAnyFloor(adjacentEscapeIdx) {
		t.Error("床タイルに隣接する位置で隣接床検出が失敗")
	}
}
