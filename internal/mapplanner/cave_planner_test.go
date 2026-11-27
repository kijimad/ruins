package mapplanner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newWallFilledPlan は width×height をすべて壁で満たした MetaPlan を返す
func newWallFilledPlan(width, height int) *MetaPlan {
	mp := &MetaPlan{
		Level:     gc.Level{TileWidth: consts.Tile(width), TileHeight: consts.Tile(height)},
		RawMaster: CreateTestRawMaster(),
		RNG:       rand.New(rand.NewPCG(1, 2)),
	}
	tiles := make([]rawschema.Tile, width*height)
	wall := mp.GetTile("wall")
	for i := range tiles {
		tiles[i] = wall
	}
	mp.Tiles = tiles
	return mp
}

func TestCaveCellularAutomata_PlanMeta_閾値未満の床領域は部屋として登録しない(t *testing.T) {
	t.Parallel()

	const width, height = 6, 5
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")

	// (1,1)-(3,2) の3x2=6マスの床領域を作る
	for x := 1; x <= 3; x++ {
		for y := 1; y <= 2; y++ {
			mp.Tiles[y*width+x] = floor
		}
	}

	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 0, MinRoomTiles: 7}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	assert.Empty(t, mp.Rooms, "6マスの領域はMinRoomTiles=7未満なので部屋として登録されない")
}

func TestCaveCellularAutomata_PlanMeta_閾値以上の床領域を部屋として抽出する(t *testing.T) {
	t.Parallel()

	const width, height = 6, 5
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")

	// (1,1)-(3,2) の3x2=6マスの床領域を作る
	for x := 1; x <= 3; x++ {
		for y := 1; y <= 2; y++ {
			mp.Tiles[y*width+x] = floor
		}
	}

	// Iterations: 0 でセルラーオートマトンを走らせずに部屋抽出だけ検証する
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 0, MinRoomTiles: 6}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	require.Len(t, mp.Rooms, 1)
	assert.Equal(t, gc.Rect{
		Min: consts.Coord[consts.Tile]{X: 1, Y: 1},
		Max: consts.Coord[consts.Tile]{X: 3, Y: 2},
	}, mp.Rooms[0])
}

func TestCaveCellularAutomata_PlanMeta_左右端はオートマトンの結果に関わらず壁になる(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")
	for i := range mp.Tiles {
		mp.Tiles[i] = floor
	}

	// WallNeighborsを到達不可能な大きさにして、内側のマスは常に床判定になるようにする
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 1, WallNeighbors: 100}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	for y := range height {
		assert.Nil(t, mp.Tiles[y*width+0].Passable, "左端(x=0)は壁になる")
		assert.Nil(t, mp.Tiles[y*width+width-1].Passable, "右端(x=width-1)は壁になる")
		for x := 1; x < width-1; x++ {
			assert.NotNil(t, mp.Tiles[y*width+x].Passable, "内側(x=%d)は床のままになる", x)
		}
	}
}

func TestCaveCellularAutomata_PlanMeta_壁の近傍数が閾値以上なら内側も壁になる(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")
	for i := range mp.Tiles {
		mp.Tiles[i] = floor
	}

	// WallNeighbors: 0なら近傍の壁数が常に閾値以上になり、内側も壁に変わる
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 1, WallNeighbors: 0}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	for x := 1; x < width-1; x++ {
		for y := range height {
			assert.Nil(t, mp.Tiles[y*width+x].Passable, "WallNeighbors=0なら内側(%d,%d)も壁になる", x, y)
		}
	}
}

func TestConnectRooms_PlanMeta_部屋が1つ以下なら何もしない(t *testing.T) {
	t.Parallel()

	connect := ConnectRooms{Tiles: testLayoutTiles}

	t.Run("部屋が0個", func(t *testing.T) {
		t.Parallel()
		mp := newWallFilledPlan(10, 10)
		before := append([]rawschema.Tile(nil), mp.Tiles...)

		require.NoError(t, connect.PlanMeta(mp))

		assert.Equal(t, before, mp.Tiles, "部屋が無ければタイルは変化しない")
	})

	t.Run("部屋が1個", func(t *testing.T) {
		t.Parallel()
		mp := newWallFilledPlan(10, 10)
		mp.Rooms = []gc.Rect{{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}}
		before := append([]rawschema.Tile(nil), mp.Tiles...)

		require.NoError(t, connect.PlanMeta(mp))

		assert.Equal(t, before, mp.Tiles, "部屋が1個だけなら繋ぐ相手がおらずタイルは変化しない")
	})
}

func TestConnectRooms_PlanMeta_2つの部屋をL字トンネルで繋ぐ(t *testing.T) {
	t.Parallel()

	const width, height = 20, 10
	mp := newWallFilledPlan(width, height)
	room1 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}
	room2 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 14, Y: 6}, Max: consts.Coord[consts.Tile]{X: 16, Y: 8}}
	mp.Rooms = []gc.Rect{room1, room2}

	connect := ConnectRooms{Tiles: testLayoutTiles}
	require.NoError(t, connect.PlanMeta(mp))

	isFloor := func(x, y int) bool {
		return mp.Tiles[y*width+x].Passable != nil
	}

	// center1=(3,3) center2=(15,7)。水平移動でy=3の行、垂直移動でx=15の列が掘られる
	assert.True(t, isFloor(4, 3), "水平方向のトンネルが掘られている")
	assert.True(t, isFloor(9, 3), "水平方向の中間地点が掘られている")
	assert.True(t, isFloor(15, 3), "水平から垂直へ折れる角が掘られている")
	assert.True(t, isFloor(15, 5), "垂直方向の中間地点が掘られている")
	assert.True(t, isFloor(15, 7), "垂直方向の終点付近が掘られている")

	assert.False(t, isFloor(0, 0), "トンネルから離れた角は壁のまま")
	assert.False(t, isFloor(width-1, height-1), "トンネルから離れた角は壁のまま")
}

func TestConnectRooms_PlanMeta_3つの部屋は隣り合う組だけ繋ぐ(t *testing.T) {
	t.Parallel()

	const width, height = 30, 10
	mp := newWallFilledPlan(width, height)
	room1 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}
	room2 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 14, Y: 2}, Max: consts.Coord[consts.Tile]{X: 16, Y: 4}}
	room3 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 26, Y: 2}, Max: consts.Coord[consts.Tile]{X: 28, Y: 4}}
	mp.Rooms = []gc.Rect{room1, room2, room3}

	connect := ConnectRooms{Tiles: testLayoutTiles}
	require.NoError(t, connect.PlanMeta(mp))

	isFloor := func(x, y int) bool {
		return mp.Tiles[y*width+x].Passable != nil
	}

	// room1(center x=3) と room2(center x=15) の間、room2 と room3(center x=27) の間がそれぞれ掘られる
	assert.True(t, isFloor(9, 3), "room1とroom2の間が掘られている")
	assert.True(t, isFloor(21, 3), "room2とroom3の間が掘られている")
}

func TestScatterWalls_PlanMeta_確率1なら境界以外の床を壁にする(t *testing.T) {
	t.Parallel()

	const width, height = 10, 10
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")
	for i := range mp.Tiles {
		mp.Tiles[i] = floor
	}

	scatter := ScatterWalls{Row: rawschema.ScatterWallStep{Chance: 1.0}, Tiles: testLayoutTiles}
	require.NoError(t, scatter.PlanMeta(mp))

	for x := 2; x < width-2; x++ {
		for y := 2; y < height-2; y++ {
			assert.Nil(t, mp.Tiles[y*width+x].Passable, "確率1なら対象範囲(%d,%d)は壁になる", x, y)
		}
	}
	// 境界から2マス未満は対象外なので床のまま
	assert.NotNil(t, mp.Tiles[0*width+0].Passable, "境界に近いマスは対象外で床のまま")
}

func TestScatterWalls_PlanMeta_確率0なら床を変化させない(t *testing.T) {
	t.Parallel()

	const width, height = 10, 10
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")
	for i := range mp.Tiles {
		mp.Tiles[i] = floor
	}
	before := append([]rawschema.Tile(nil), mp.Tiles...)

	scatter := ScatterWalls{Row: rawschema.ScatterWallStep{Chance: 0.0}, Tiles: testLayoutTiles}
	require.NoError(t, scatter.PlanMeta(mp))

	assert.Equal(t, before, mp.Tiles, "確率0ならタイルは変化しない")
}
