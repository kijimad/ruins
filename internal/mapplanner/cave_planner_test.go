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

	// x=1..3、y=1..2 の3x2=6マスの床領域を作る
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

	// x=1..3、y=1..2 の3x2=6マスの床領域を作る
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
		assert.Nil(t, mp.Tiles[y*width+0].Passable, "x=0 の左端は壁になる")
		assert.Nil(t, mp.Tiles[y*width+width-1].Passable, "x=width-1 の右端は壁になる")
		for x := 1; x < width-1; x++ {
			assert.NotNil(t, mp.Tiles[y*width+x].Passable, "x=%d の内側は床のままになる", x)
		}
	}
}

func TestCaveCellularAutomata_PlanMeta_境界外は壁として数えられ上下端が壁になる(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")
	for i := range mp.Tiles {
		mp.Tiles[i] = floor
	}

	// 自マスを含む3x3を数え、境界外は壁とみなす。全面床なので、上端と下端は
	// 境界外の3マスだけで WallNeighbors=3 に達して壁になり、y=1 は境界外を含まず床のまま残る
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 1, WallNeighbors: 3}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	for x := 1; x < width-1; x++ {
		assert.Nil(t, mp.Tiles[0*width+x].Passable, "上端の x=%d は境界外を3マス数えて壁になる", x)
		assert.Nil(t, mp.Tiles[(height-1)*width+x].Passable, "下端の x=%d は境界外を3マス数えて壁になる", x)
		assert.NotNil(t, mp.Tiles[1*width+x].Passable, "y=1 の x=%d は境界外を含まず床のまま", x)
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

	// 左上の向きでは中心が y=7 になる。下2マスの y=9 が外周にならない高さにする
	const width, height = 20, 11
	roomA := gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}
	roomB := gc.Rect{Min: consts.Coord[consts.Tile]{X: 14, Y: 6}, Max: consts.Coord[consts.Tile]{X: 16, Y: 8}}

	tests := []struct {
		name  string
		rooms []gc.Rect
	}{
		// 中心 3,3 から中心 15,7 へ。currentXは++、currentYは++の方向に動く
		{name: "右下方向に繋ぐ", rooms: []gc.Rect{roomA, roomB}},
		// 中心 15,7 から中心 3,3 へ。currentXは--、currentYは--の方向に動く
		{name: "左上方向に繋ぐ", rooms: []gc.Rect{roomB, roomA}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mp := newWallFilledPlan(width, height)
			mp.Rooms = tt.rooms

			connect := ConnectRooms{Tiles: testLayoutTiles}
			require.NoError(t, connect.PlanMeta(mp))

			isFloor := func(x, y int) bool {
				return mp.Tiles[y*width+x].Passable != nil
			}

			center1 := tt.rooms[0]
			center2 := tt.rooms[1]
			c1x, c1y := int(center1.Min.X+center1.Max.X)/2, int(center1.Min.Y+center1.Max.Y)/2
			c2x, c2y := int(center2.Min.X+center2.Max.X)/2, int(center2.Min.Y+center2.Max.Y)/2

			// 水平方向の中間地点。y=center1Y±1の3マス幅が掘られ、その外は壁のまま
			midX := (c1x + c2x) / 2
			assert.True(t, isFloor(midX, c1y-1), "水平方向の中間地点、中心から上1マスが掘られている")
			assert.True(t, isFloor(midX, c1y), "水平方向の中間地点、中心が掘られている")
			assert.True(t, isFloor(midX, c1y+1), "水平方向の中間地点、中心から下1マスが掘られている")
			assert.False(t, isFloor(midX, c1y-2), "水平方向の中間地点、中心から上2マスは壁のまま")
			assert.False(t, isFloor(midX, c1y+2), "水平方向の中間地点、中心から下2マスは壁のまま")

			// 水平から垂直へ折れる角、垂直方向の終点付近
			assert.True(t, isFloor(c2x, c1y), "水平から垂直へ折れる角が掘られている")
			assert.True(t, isFloor(c2x, (c1y+c2y)/2), "垂直方向の中間地点が掘られている")
			assert.True(t, isFloor(c2x, c2y), "垂直方向の終点付近が掘られている")

			// L字の折れる側と反対の内側のマスは、どちらの向きでも掘られない
			assert.False(t, isFloor(17, 2), "L字から外れた右上の内側は壁のまま")
			assert.False(t, isFloor(2, 8), "L字から外れた左下の内側は壁のまま")
		})
	}
}

func TestConnectRooms_PlanMeta_3つの部屋は隣り合う組だけ繋ぐ(t *testing.T) {
	t.Parallel()

	const width, height = 30, 10
	mp := newWallFilledPlan(width, height)
	// room2の中心yをroom1・room3とずらし、room1→room3を直接繋ぐ誤った実装だと
	// 掘られてしまうがroom1→room2→room3の経路では掘られないマスを作る
	room1 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}
	room2 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 14, Y: 6}, Max: consts.Coord[consts.Tile]{X: 16, Y: 8}}
	room3 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 26, Y: 2}, Max: consts.Coord[consts.Tile]{X: 28, Y: 4}}
	mp.Rooms = []gc.Rect{room1, room2, room3}

	connect := ConnectRooms{Tiles: testLayoutTiles}
	require.NoError(t, connect.PlanMeta(mp))

	isFloor := func(x, y int) bool {
		return mp.Tiles[y*width+x].Passable != nil
	}

	// 中心 3,3 の room1 から中心 15,7 の room2 へ。y=3の行がx=4..15、x=15の列がy=3..7で掘られる
	assert.True(t, isFloor(9, 3), "room1とroom2の間、水平方向が掘られている")
	assert.True(t, isFloor(15, 5), "room1とroom2の間、垂直方向が掘られている")
	// 中心 15,7 の room2 から中心 27,3 の room3 へ。y=7の行がx=16..27、x=27の列がy=3..7で掘られる
	assert.True(t, isFloor(21, 7), "room2とroom3の間、水平方向が掘られている")
	assert.True(t, isFloor(27, 5), "room2とroom3の間、垂直方向が掘られている")

	// room1→room3を直接繋ぐ誤った実装なら21,3はy=3のまま一直線に掘られてしまう。
	// room1→room2→room3の経路ではここは通らず壁のまま残る
	assert.False(t, isFloor(21, 3), "room1とroom3は直接繋がれていない")
}

func TestConnectRooms_PlanMeta_トンネルは外周の壁を侵さない(t *testing.T) {
	t.Parallel()

	const width, height = 10, 10
	at := func(x, y consts.Tile) gc.Rect {
		return gc.Rect{Min: consts.Coord[consts.Tile]{X: x, Y: y}, Max: consts.Coord[consts.Tile]{X: x, Y: y}}
	}

	tests := []struct {
		name  string
		rooms []gc.Rect
		band  func(x, y int) bool
	}{
		{
			// 太さの方向が外周へはみ出そうとする
			name:  "中心が外周の1マス内側を一周する",
			rooms: []gc.Rect{at(1, 1), at(8, 1), at(8, 8), at(1, 8), at(1, 1)},
			band:  func(x, y int) bool { return x <= 2 || x >= 7 || y <= 2 || y >= 7 },
		},
		{
			// 進む方向が外周の上を通る
			name:  "中心が外周の上を一周する",
			rooms: []gc.Rect{at(0, 0), at(9, 0), at(9, 9), at(0, 9), at(0, 0)},
			band:  func(x, y int) bool { return x == 1 || x == 8 || y == 1 || y == 8 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mp := newWallFilledPlan(width, height)
			mp.Rooms = tt.rooms

			connect := ConnectRooms{Tiles: testLayoutTiles}
			require.NoError(t, connect.PlanMeta(mp))

			want := make([]bool, width*height)
			got := make([]bool, width*height)
			for y := range height {
				for x := range width {
					inside := x >= 1 && x <= 8 && y >= 1 && y <= 8
					want[y*width+x] = inside && tt.band(x, y)
					got[y*width+x] = mp.Tiles[y*width+x].Passable != nil
				}
			}
			assert.Equal(t, want, got, "外周の内側に輪が掘られ、外周4辺は壁のまま")
		})
	}
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

	want := make([]bool, width*height)
	got := make([]bool, width*height)
	for y := range height {
		for x := range width {
			target := x >= 2 && x < width-2 && y >= 2 && y < height-2
			want[y*width+x] = !target
			got[y*width+x] = mp.Tiles[y*width+x].Passable != nil
		}
	}
	assert.Equal(t, want, got, "外周から2マス以上内側だけが壁になり、その外の2マス幅は床のまま")
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

func TestWiden_PlanMeta_床に2マス以上接する外周以外の壁を床にする(t *testing.T) {
	t.Parallel()

	const width, height = 7, 7
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")
	// x=3,y=3 は左右の2マスで床に接する。x=0,y=3 は上下の2マスで接するが外周なので対象外
	for _, c := range [][2]int{{2, 3}, {4, 3}, {0, 2}, {0, 4}} {
		mp.Tiles[c[1]*width+c[0]] = floor
	}

	widen := Widen{Row: rawschema.WidenStep{Chance: 1.0}, Tiles: testLayoutTiles}
	require.NoError(t, widen.PlanMeta(mp))

	want := make([]bool, width*height)
	for _, c := range [][2]int{{2, 3}, {4, 3}, {0, 2}, {0, 4}, {3, 3}} {
		want[c[1]*width+c[0]] = true
	}
	got := make([]bool, width*height)
	for i := range mp.Tiles {
		got[i] = mp.Tiles[i].Passable != nil
	}
	assert.Equal(t, want, got, "床に2マス接する内側の壁だけが床になる")
}

func TestWiden_PlanMeta_確率0なら壁を変化させない(t *testing.T) {
	t.Parallel()

	const width, height = 7, 7
	mp := newWallFilledPlan(width, height)
	floor := mp.GetTile("floor")
	mp.Tiles[3*width+2] = floor
	mp.Tiles[3*width+4] = floor
	before := append([]rawschema.Tile(nil), mp.Tiles...)

	widen := Widen{Row: rawschema.WidenStep{Chance: 0.0}, Tiles: testLayoutTiles}
	require.NoError(t, widen.PlanMeta(mp))

	assert.Equal(t, before, mp.Tiles, "確率0ならタイルは変化しない")
}
