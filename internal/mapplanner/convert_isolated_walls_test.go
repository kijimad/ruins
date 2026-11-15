package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertIsolatedWalls_ToFloor(t *testing.T) {
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
	width, height := consts.Tile(7), consts.Tile(7)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]rawschema.Tile, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]gc.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// 内側5x5を床にする
	for x := 1; x <= 5; x++ {
		for y := 1; y <= 5; y++ {
			idx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)})
			planData.Tiles[idx] = planData.GetTile("floor")
		}
	}

	// 中央の床を壁に戻す（床に隣接する壁）
	centerIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 3, Y: 3})
	planData.Tiles[centerIdx] = planData.GetTile("wall")

	// プランナーを実行（floorに変換）
	converter := ConvertIsolatedWalls{ReplacementTile: consts.TileNameFloor}
	err := converter.PlanMeta(planData)
	require.NoError(t, err)

	// 検証1: 床に隣接する壁はそのまま壁として残る
	// 中央の壁
	assert.Equal(t, "wall", planData.Tiles[centerIdx].Name, "床に隣接する中央の壁はwallのまま")

	// 周囲の壁（床に隣接）
	topWallIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 3, Y: 0})
	bottomWallIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 3, Y: 6})
	leftWallIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 0, Y: 3})
	rightWallIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 6, Y: 3})

	assert.Equal(t, "wall", planData.Tiles[topWallIdx].Name, "床に隣接する上壁はwallのまま")
	assert.Equal(t, "wall", planData.Tiles[bottomWallIdx].Name, "床に隣接する下壁はwallのまま")
	assert.Equal(t, "wall", planData.Tiles[leftWallIdx].Name, "床に隣接する左壁はwallのまま")
	assert.Equal(t, "wall", planData.Tiles[rightWallIdx].Name, "床に隣接する右壁はwallのまま")

	// 検証2: 四隅の壁も斜めで床に隣接しているのでwallのまま
	topLeftIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 0, Y: 0})
	topRightIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 6, Y: 0})
	bottomLeftIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 0, Y: 6})
	bottomRightIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 6, Y: 6})

	assert.Equal(t, "wall", planData.Tiles[topLeftIdx].Name, "四隅の壁も斜めで床に隣接")
	assert.Equal(t, "wall", planData.Tiles[topRightIdx].Name, "四隅の壁も斜めで床に隣接")
	assert.Equal(t, "wall", planData.Tiles[bottomLeftIdx].Name, "四隅の壁も斜めで床に隣接")
	assert.Equal(t, "wall", planData.Tiles[bottomRightIdx].Name, "四隅の壁も斜めで床に隣接")

	// 検証3: 床タイルはそのまま床として残る
	floorIdx := planData.Level.CoordToIndex(consts.Coord[consts.Tile]{X: 1, Y: 1})
	assert.Equal(t, "floor", planData.Tiles[floorIdx].Name, "床タイルはそのまま")
}

func TestConvertIsolatedWalls_AllWalls_ToVoid(t *testing.T) {
	t.Parallel()

	// 全て壁のマップ（床が無い）
	width, height := consts.Tile(5), consts.Tile(5)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]rawschema.Tile, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]gc.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を壁で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("wall")
	}

	// プランナーを実行（voidに変換）
	converter := ConvertIsolatedWalls{ReplacementTile: consts.TileNameVoid}
	err := converter.PlanMeta(planData)
	require.NoError(t, err)

	// 検証: 端の壁はwallのまま、内側の壁はvoidに変換される
	for i, tile := range planData.Tiles {
		pos := planData.Level.IndexToCoord(gc.TileIdx(i))
		isEdge := int(pos.X) == 0 || int(pos.X) == int(width)-1 || int(pos.Y) == 0 || int(pos.Y) == int(height)-1

		if isEdge {
			assert.Equal(t, "wall", tile.Name, "端のタイル(%d, %d)は境界として壁のまま", pos.X, pos.Y)
		} else {
			assert.Equal(t, "void", tile.Name, "内側のタイル(%d, %d)は床に隣接しないためvoidに変換", pos.X, pos.Y)
		}
	}
}

func TestConvertIsolatedWalls_NoWalls(t *testing.T) {
	t.Parallel()

	// 壁が無いマップ（全て床）
	width, height := consts.Tile(3), consts.Tile(3)
	planData := &MetaPlan{
		Level: gc.Level{
			TileWidth:  width,
			TileHeight: height,
		},
		Tiles:     make([]rawschema.Tile, int(width)*int(height)),
		Rooms:     []gc.Rect{},
		Corridors: [][]gc.TileIdx{},
		RawMaster: CreateTestRawMaster(),
	}

	// 全体を床で埋める
	for i := range planData.Tiles {
		planData.Tiles[i] = planData.GetTile("floor")
	}

	// プランナーを実行
	converter := ConvertIsolatedWalls{ReplacementTile: consts.TileNameVoid}
	err := converter.PlanMeta(planData)
	require.NoError(t, err)

	// 全て床のまま
	for i, tile := range planData.Tiles {
		assert.Equal(t, "floor", tile.Name, "タイル%dは床のまま", i)
	}
}

func TestConvertIsolatedWalls_置き換え先のgroundは計画の下地(t *testing.T) {
	t.Parallel()

	// 5x5 を全て壁にすると、端でない中央の壁は床に接しない孤立した壁になる
	newPlan := func(ground string) *MetaPlan {
		planData := &MetaPlan{
			Level:     gc.Level{TileWidth: 5, TileHeight: 5},
			Tiles:     make([]rawschema.Tile, 25),
			RawMaster: CreateTestRawMaster(),
			Ground:    ground,
		}
		for i := range planData.Tiles {
			planData.Tiles[i] = planData.GetTile("wall")
		}
		return planData
	}
	// 5x5 の中央は添字 2*5+2
	const center = 12

	t.Run("孤立した壁を計画の下地にする", func(t *testing.T) {
		t.Parallel()
		planData := newPlan("dirt")
		require.NoError(t, ConvertIsolatedWalls{ReplacementTile: maptemplate.GroundTerrain}.PlanMeta(planData))
		assert.Equal(t, "dirt", planData.Tiles[center].Id)
	})
}

func TestConvertIsolatedWalls_壁でない通れないタイルは置き換えない(t *testing.T) {
	t.Parallel()

	// 5x5 の中央は添字 2*5+2。全てを同じタイルにすると、中央は通れるマスに接しない
	const center = 12
	tests := []struct {
		fill string
		want string
	}{
		{"wall", "floor"},
		{"void", "void"},
	}
	for _, tt := range tests {
		t.Run(tt.fill, func(t *testing.T) {
			t.Parallel()
			planData := &MetaPlan{
				Level:     gc.Level{TileWidth: 5, TileHeight: 5},
				Tiles:     make([]rawschema.Tile, 25),
				RawMaster: CreateTestRawMaster(),
			}
			for i := range planData.Tiles {
				planData.Tiles[i] = planData.GetTile(tt.fill)
			}
			require.NoError(t, ConvertIsolatedWalls{ReplacementTile: "floor"}.PlanMeta(planData))
			assert.Equal(t, tt.want, planData.Tiles[center].Id)
		})
	}
}
