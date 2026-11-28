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

// newFilledPlan は width×height をすべて tileName のタイルで満たした MetaPlan を返す
func newFilledPlan(width, height int, tileName rawschema.EntityID) *MetaPlan {
	mp := &MetaPlan{
		Level:     gc.Level{TileWidth: consts.Tile(width), TileHeight: consts.Tile(height)},
		RawMaster: CreateTestRawMaster(),
		RNG:       rand.New(rand.NewPCG(1, 2)),
	}
	tiles := make([]rawschema.Tile, width*height)
	tile := mp.GetTile(tileName)
	for i := range tiles {
		tiles[i] = tile
	}
	mp.Tiles = tiles
	return mp
}

func TestCaveCellularAutomata_PlanMeta_閾値以上の床領域だけを部屋として抽出する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		minRoomTiles int
		want         []gc.Rect
	}{
		{name: "6マスは閾値7に届かない", minRoomTiles: 7, want: nil},
		{
			name:         "6マスは閾値6に届く",
			minRoomTiles: 6,
			want:         []gc.Rect{{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 3, Y: 2}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const width, height = 6, 5
			mp := newFilledPlan(width, height, testLayoutTiles.Wall)
			floor := mp.GetTile(testLayoutTiles.Floor)
			// x=1..3、y=1..2 の3x2=6マスの床領域を作る
			for x := 1; x <= 3; x++ {
				for y := 1; y <= 2; y++ {
					mp.Tiles[y*width+x] = floor
				}
			}

			// Iterations: 0 でセルラーオートマトンを走らせずに部屋抽出だけ検証する
			automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 0, MinRoomTiles: tt.minRoomTiles}, Tiles: testLayoutTiles}
			require.NoError(t, automata.PlanMeta(mp))

			assert.Equal(t, tt.want, mp.Rooms)
		})
	}
}

func TestCaveCellularAutomata_PlanMeta_長方形でない床領域は外接矩形を部屋にする(t *testing.T) {
	t.Parallel()

	const width, height = 4, 4
	mp := newFilledPlan(width, height, testLayoutTiles.Wall)
	floor := mp.GetTile(testLayoutTiles.Floor)
	// 走査で最初に見つかる x=1,y=2 より上に x=2,y=1 がある L 字
	for _, c := range [][2]int{{1, 2}, {2, 2}, {2, 1}} {
		mp.Tiles[c[1]*width+c[0]] = floor
	}

	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 0, MinRoomTiles: 1}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	assert.Equal(t, []gc.Rect{
		{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 2, Y: 2}},
	}, mp.Rooms)
}

func TestCaveCellularAutomata_PlanMeta_斜めにだけ接する床は別の部屋になる(t *testing.T) {
	t.Parallel()

	const width, height = 4, 4
	mp := newFilledPlan(width, height, testLayoutTiles.Wall)
	floor := mp.GetTile(testLayoutTiles.Floor)
	mp.Tiles[1*width+1] = floor
	mp.Tiles[2*width+2] = floor

	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 0, MinRoomTiles: 1}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	// 部屋は x を外側に回す走査で見つけた順に並び、ConnectRooms はこの順に結ぶ
	assert.Equal(t, []gc.Rect{
		{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 1, Y: 1}},
		{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 2, Y: 2}},
	}, mp.Rooms)
}

func TestCaveCellularAutomata_PlanMeta_左右端はオートマトンの結果に関わらず壁になる(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)

	// WallNeighborsを到達不可能な大きさにして、内側のマスは常に床判定になるようにする
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 1, WallNeighbors: 100}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	want := make([]bool, width*height)
	got := make([]bool, width*height)
	for y := range height {
		for x := range width {
			want[y*width+x] = x != 0 && x != width-1
			got[y*width+x] = mp.Tiles[y*width+x].Passable != nil
		}
	}
	assert.Equal(t, want, got, "左右端だけが壁になり、内側は床のまま")
}

func TestCaveCellularAutomata_PlanMeta_境界外は壁として数えられ上下端が壁になる(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)

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

func TestCaveCellularAutomata_PlanMeta_2回目は1回目の結果を数える(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)

	// 1回目で外周がすべて壁になり、2回目は内側のどのマスも外周の壁を3マス以上数える
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 2, WallNeighbors: 3}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	got := make([]bool, width*height)
	for i := range mp.Tiles {
		got[i] = mp.Tiles[i].Passable != nil
	}
	assert.Equal(t, make([]bool, width*height), got, "2回目で内側の床もすべて壁になる")
}

func TestConnectRooms_PlanMeta_部屋が1つ以下なら何もしない(t *testing.T) {
	t.Parallel()

	connect := ConnectRooms{Tiles: testLayoutTiles}

	t.Run("部屋が0個", func(t *testing.T) {
		t.Parallel()
		mp := newFilledPlan(10, 10, testLayoutTiles.Wall)
		before := append([]rawschema.Tile(nil), mp.Tiles...)

		require.NoError(t, connect.PlanMeta(mp))

		assert.Equal(t, before, mp.Tiles, "部屋が無ければタイルは変化しない")
	})

	t.Run("部屋が1個", func(t *testing.T) {
		t.Parallel()
		mp := newFilledPlan(10, 10, testLayoutTiles.Wall)
		mp.Rooms = []gc.Rect{{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}}
		before := append([]rawschema.Tile(nil), mp.Tiles...)

		require.NoError(t, connect.PlanMeta(mp))

		assert.Equal(t, before, mp.Tiles, "部屋が1個だけなら繋ぐ相手がおらずタイルは変化しない")
	})
}

func TestConnectRooms_PlanMeta_2つの部屋をL字トンネルで繋ぐ(t *testing.T) {
	t.Parallel()

	const width, height = 20, 11
	roomA := gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}
	roomB := gc.Rect{Min: consts.Coord[consts.Tile]{X: 14, Y: 6}, Max: consts.Coord[consts.Tile]{X: 16, Y: 8}}
	band := func(minX, minY, maxX, maxY int) gc.Rect {
		return gc.Rect{
			Min: consts.Coord[consts.Tile]{X: consts.Tile(minX), Y: consts.Tile(minY)},
			Max: consts.Coord[consts.Tile]{X: consts.Tile(maxX), Y: consts.Tile(maxY)},
		}
	}

	// 1歩進んでから掘るので始点の中心は掘られない。水平の帯は終点の x まで、垂直の帯は終点の y まで伸びる
	tests := []struct {
		name  string
		rooms []gc.Rect
		bands []gc.Rect
	}{
		{
			name:  "中心3,3から中心15,7へ右下に繋ぐ",
			rooms: []gc.Rect{roomA, roomB},
			bands: []gc.Rect{band(4, 2, 15, 4), band(14, 4, 16, 7)},
		},
		{
			name:  "中心15,7から中心3,3へ左上に繋ぐ",
			rooms: []gc.Rect{roomB, roomA},
			bands: []gc.Rect{band(3, 6, 14, 8), band(2, 3, 4, 6)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mp := newFilledPlan(width, height, testLayoutTiles.Wall)
			mp.Rooms = tt.rooms

			connect := ConnectRooms{Tiles: testLayoutTiles}
			require.NoError(t, connect.PlanMeta(mp))

			want := make([]bool, width*height)
			got := make([]bool, width*height)
			for y := range height {
				for x := range width {
					for _, b := range tt.bands {
						if x >= int(b.Min.X) && x <= int(b.Max.X) && y >= int(b.Min.Y) && y <= int(b.Max.Y) {
							want[y*width+x] = true
						}
					}
					got[y*width+x] = mp.Tiles[y*width+x].Passable != nil
				}
			}
			assert.Equal(t, want, got, "太さ3の水平と垂直の帯だけが掘られる")
		})
	}
}

func TestConnectRooms_PlanMeta_3つの部屋は隣り合う組だけ繋ぐ(t *testing.T) {
	t.Parallel()

	const width, height = 30, 10
	mp := newFilledPlan(width, height, testLayoutTiles.Wall)
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

			mp := newFilledPlan(width, height, testLayoutTiles.Wall)
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
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)

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
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)
	before := append([]rawschema.Tile(nil), mp.Tiles...)

	scatter := ScatterWalls{Row: rawschema.ScatterWallStep{Chance: 0.0}, Tiles: testLayoutTiles}
	require.NoError(t, scatter.PlanMeta(mp))

	assert.Equal(t, before, mp.Tiles, "確率0ならタイルは変化しない")
}

func TestWiden_PlanMeta_床に2マス以上接する外周以外の壁を床にする(t *testing.T) {
	t.Parallel()

	const width, height = 7, 7
	mp := newFilledPlan(width, height, testLayoutTiles.Wall)
	floor := mp.GetTile(testLayoutTiles.Floor)
	// x=3,y=3 は左右の2マスで床に接する。x=0,y=3 は上下の2マスで接するが外周なので対象外。
	// x=3,y=4 は元の盤面では x=3,y=5 の1マスにしか接しない。x=3,y=3 が床になった後の盤面を
	// 数える実装でだけ2マスになる
	for _, c := range [][2]int{{2, 3}, {4, 3}, {0, 2}, {0, 4}, {3, 5}} {
		mp.Tiles[c[1]*width+c[0]] = floor
	}

	widen := Widen{Row: rawschema.WidenStep{Chance: 1.0}, Tiles: testLayoutTiles}
	require.NoError(t, widen.PlanMeta(mp))

	want := make([]bool, width*height)
	for _, c := range [][2]int{{2, 3}, {4, 3}, {0, 2}, {0, 4}, {3, 5}, {3, 3}} {
		want[c[1]*width+c[0]] = true
	}
	got := make([]bool, width*height)
	for i := range mp.Tiles {
		got[i] = mp.Tiles[i].Passable != nil
	}
	assert.Equal(t, want, got, "元の盤面で床に2マス接する内側の壁だけが床になる")
}

func TestWiden_PlanMeta_確率0なら壁を変化させない(t *testing.T) {
	t.Parallel()

	const width, height = 7, 7
	mp := newFilledPlan(width, height, testLayoutTiles.Wall)
	floor := mp.GetTile(testLayoutTiles.Floor)
	mp.Tiles[3*width+2] = floor
	mp.Tiles[3*width+4] = floor
	before := append([]rawschema.Tile(nil), mp.Tiles...)

	widen := Widen{Row: rawschema.WidenStep{Chance: 0.0}, Tiles: testLayoutTiles}
	require.NoError(t, widen.PlanMeta(mp))

	assert.Equal(t, before, mp.Tiles, "確率0ならタイルは変化しない")
}
