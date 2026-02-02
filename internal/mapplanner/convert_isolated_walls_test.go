package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/stretchr/testify/assert"
)

func TestConvertIsolatedWallsToFloor(t *testing.T) {
	t.Parallel()

	// テスト用のマップを作成（7x7）
	// 配置パターン:
	// W W W W W W W
	// W F F F F F W
	// W F F F F F W
	// W F F F F F W
	// W F F F F F W
	// W F F F F F W
	// W W W W W W W
	width, height := gc.Tile(7), gc.Tile(7)
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

	// 内側5x5を床にする
	for x := 1; x <= 5; x++ {
		for y := 1; y <= 5; y++ {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))
			planData.Tiles[idx] = planData.GetTile("floor")
		}
	}

	// 中央の床を壁に戻す（床に隣接する壁）
	centerIdx := planData.Level.XYTileIndex(3, 3)
	planData.Tiles[centerIdx] = planData.GetTile("wall")

	// プランナーを実行
	converter := NewConvertIsolatedWallsToFloor()
	err := converter.PlanMeta(planData)
	assert.NoError(t, err)

	// 検証1: 床に隣接する壁はそのまま壁として残る
	// 中央の壁
	assert.Equal(t, "wall", planData.Tiles[centerIdx].Name, "床に隣接する中央の壁はwallのまま")

	// 周囲の壁（床に隣接）
	topWallIdx := planData.Level.XYTileIndex(3, 0)
	bottomWallIdx := planData.Level.XYTileIndex(3, 6)
	leftWallIdx := planData.Level.XYTileIndex(0, 3)
	rightWallIdx := planData.Level.XYTileIndex(6, 3)

	assert.Equal(t, "wall", planData.Tiles[topWallIdx].Name, "床に隣接する上壁はwallのまま")
	assert.Equal(t, "wall", planData.Tiles[bottomWallIdx].Name, "床に隣接する下壁はwallのまま")
	assert.Equal(t, "wall", planData.Tiles[leftWallIdx].Name, "床に隣接する左壁はwallのまま")
	assert.Equal(t, "wall", planData.Tiles[rightWallIdx].Name, "床に隣接する右壁はwallのまま")

	// 検証2: 四隅の壁も斜めで床に隣接しているのでwallのまま
	topLeftIdx := planData.Level.XYTileIndex(0, 0)
	topRightIdx := planData.Level.XYTileIndex(6, 0)
	bottomLeftIdx := planData.Level.XYTileIndex(0, 6)
	bottomRightIdx := planData.Level.XYTileIndex(6, 6)

	assert.Equal(t, "wall", planData.Tiles[topLeftIdx].Name, "四隅の壁も斜めで床に隣接")
	assert.Equal(t, "wall", planData.Tiles[topRightIdx].Name, "四隅の壁も斜めで床に隣接")
	assert.Equal(t, "wall", planData.Tiles[bottomLeftIdx].Name, "四隅の壁も斜めで床に隣接")
	assert.Equal(t, "wall", planData.Tiles[bottomRightIdx].Name, "四隅の壁も斜めで床に隣接")

	// 検証3: 床タイルはそのまま床として残る
	floorIdx := planData.Level.XYTileIndex(1, 1)
	assert.Equal(t, "floor", planData.Tiles[floorIdx].Name, "床タイルはそのまま")
}

func TestConvertIsolatedWallsToFloor_AllWalls(t *testing.T) {
	t.Parallel()

	// 全て壁のマップ（床が無い）
	width, height := gc.Tile(3), gc.Tile(3)
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

	// プランナーを実行
	converter := NewConvertIsolatedWallsToFloor()
	err := converter.PlanMeta(planData)
	assert.NoError(t, err)

	// 全てfloorに変換されるべき
	for i, tile := range planData.Tiles {
		assert.Equal(t, "floor", tile.Name, "タイル%dは床に隣接しないためfloorに変換されるべき", i)
	}
}

func TestConvertIsolatedWallsToFloor_NoWalls(t *testing.T) {
	t.Parallel()

	// 壁が無いマップ（全て床）
	width, height := gc.Tile(3), gc.Tile(3)
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

	// 全体を床で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("floor")
	}

	// プランナーを実行
	converter := NewConvertIsolatedWallsToFloor()
	err := converter.PlanMeta(planData)
	assert.NoError(t, err)

	// 全て床のまま
	for i, tile := range planData.Tiles {
		assert.Equal(t, "floor", tile.Name, "タイル%dは床のまま", i)
	}
}
