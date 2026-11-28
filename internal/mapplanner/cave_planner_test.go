package mapplanner

import (
	"math/rand/v2"
	"strings"
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

// newPlanFromRows は '#' を壁、'.' を床として行の絵から MetaPlan を作る
func newPlanFromRows(rows []string) *MetaPlan {
	width, height := len(rows[0]), len(rows)
	mp := newFilledPlan(width, height, testLayoutTiles.Wall)
	floor := mp.GetTile(testLayoutTiles.Floor)
	for y, row := range rows {
		for x, c := range row {
			if c == '.' {
				mp.Tiles[y*width+x] = floor
			}
		}
	}
	return mp
}

// renderRows は盤面を、壁を '#'、床を '.'、それ以外を '?' とした行の絵にする
func renderRows(mp *MetaPlan) []string {
	width := int(mp.Level.TileWidth)
	rows := make([]string, int(mp.Level.TileHeight))
	for y := range rows {
		var b strings.Builder
		for x := range width {
			switch mp.Tiles[y*width+x].Id {
			case testLayoutTiles.Wall:
				b.WriteByte('#')
			case testLayoutTiles.Floor:
				b.WriteByte('.')
			default:
				b.WriteByte('?')
			}
		}
		rows[y] = b.String()
	}
	return rows
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

			mp := newPlanFromRows([]string{
				"######",
				"#...##",
				"#...##",
				"######",
				"######",
			})

			// Iterations: 0 でセルラーオートマトンを走らせずに部屋抽出だけ検証する
			automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 0, MinRoomTiles: tt.minRoomTiles}, Tiles: testLayoutTiles}
			require.NoError(t, automata.PlanMeta(mp))

			assert.Equal(t, tt.want, mp.Rooms)
		})
	}
}

func TestCaveCellularAutomata_PlanMeta_長方形でない床領域は外接矩形を部屋にする(t *testing.T) {
	t.Parallel()

	// 走査で最初に見つかる x=1,y=2 より上に x=2,y=1 がある L 字
	mp := newPlanFromRows([]string{
		"####",
		"##.#",
		"#..#",
		"####",
	})

	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 0, MinRoomTiles: 1}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	assert.Equal(t, []gc.Rect{
		{Min: consts.Coord[consts.Tile]{X: 1, Y: 1}, Max: consts.Coord[consts.Tile]{X: 2, Y: 2}},
	}, mp.Rooms)
}

func TestCaveCellularAutomata_PlanMeta_斜めにだけ接する床は別の部屋になる(t *testing.T) {
	t.Parallel()

	mp := newPlanFromRows([]string{
		"####",
		"#.##",
		"##.#",
		"####",
	})

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

	// WallNeighbors を到達不可能な大きさにして、内側のマスは常に床判定になるようにする
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 1, WallNeighbors: 100}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	assert.Equal(t, []string{
		"#...#",
		"#...#",
		"#...#",
		"#...#",
	}, renderRows(mp))
}

func TestCaveCellularAutomata_PlanMeta_境界外は壁として数えられ上下端が壁になる(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)

	// 自マスを含む3x3を数え、境界外は壁とみなす。全面床なので、上端と下端は
	// 境界外の3マスだけで WallNeighbors=3 に達して壁になり、y=1 は境界外を含まず床のまま残る
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 1, WallNeighbors: 3}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	assert.Equal(t, []string{
		"#####",
		"#...#",
		"#...#",
		"#####",
	}, renderRows(mp))
}

func TestCaveCellularAutomata_PlanMeta_2回目は1回目の結果を数える(t *testing.T) {
	t.Parallel()

	const width, height = 5, 4
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)

	// 1回目で外周がすべて壁になり、2回目は内側のどのマスも外周の壁を3マス以上数える
	automata := CaveCellularAutomata{Row: rawschema.AutomataStep{Iterations: 2, WallNeighbors: 3}, Tiles: testLayoutTiles}
	require.NoError(t, automata.PlanMeta(mp))

	assert.Equal(t, []string{
		"#####",
		"#####",
		"#####",
		"#####",
	}, renderRows(mp))
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

	roomA := gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}
	roomB := gc.Rect{Min: consts.Coord[consts.Tile]{X: 14, Y: 6}, Max: consts.Coord[consts.Tile]{X: 16, Y: 8}}

	// 1歩進んでから掘るので始点の中心は掘られない。水平の帯は終点の x まで、垂直の帯は終点の y まで伸びる
	tests := []struct {
		name  string
		rooms []gc.Rect
		want  []string
	}{
		{
			name:  "中心3,3から中心15,7へ右下に繋ぐ",
			rooms: []gc.Rect{roomA, roomB},
			want: []string{
				"####################",
				"####################",
				"####............####",
				"####............####",
				"####.............###",
				"##############...###",
				"##############...###",
				"##############...###",
				"####################",
				"####################",
				"####################",
			},
		},
		{
			name:  "中心15,7から中心3,3へ左上に繋ぐ",
			rooms: []gc.Rect{roomB, roomA},
			want: []string{
				"####################",
				"####################",
				"####################",
				"##...###############",
				"##...###############",
				"##...###############",
				"##.............#####",
				"###............#####",
				"###............#####",
				"####################",
				"####################",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mp := newFilledPlan(20, 11, testLayoutTiles.Wall)
			mp.Rooms = tt.rooms

			connect := ConnectRooms{Tiles: testLayoutTiles}
			require.NoError(t, connect.PlanMeta(mp))

			assert.Equal(t, tt.want, renderRows(mp))
		})
	}
}

func TestConnectRooms_PlanMeta_3つの部屋は隣り合う組だけ繋ぐ(t *testing.T) {
	t.Parallel()

	mp := newFilledPlan(30, 10, testLayoutTiles.Wall)
	// room2 の中心の y を room1・room3 とずらす。room1→room3 を直接繋ぐ誤った実装なら
	// x=21,y=3 が一直線に掘られるが、room1→room2→room3 の経路では壁のまま残る
	room1 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 2, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 4}}
	room2 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 14, Y: 6}, Max: consts.Coord[consts.Tile]{X: 16, Y: 8}}
	room3 := gc.Rect{Min: consts.Coord[consts.Tile]{X: 26, Y: 2}, Max: consts.Coord[consts.Tile]{X: 28, Y: 4}}
	mp.Rooms = []gc.Rect{room1, room2, room3}

	connect := ConnectRooms{Tiles: testLayoutTiles}
	require.NoError(t, connect.PlanMeta(mp))

	assert.Equal(t, []string{
		"##############################",
		"##############################",
		"####............##############",
		"####............##########...#",
		"####.............#########...#",
		"##############...#########...#",
		"##############...............#",
		"##############..............##",
		"################............##",
		"##############################",
	}, renderRows(mp))
}

func TestConnectRooms_PlanMeta_トンネルは外周の壁を侵さない(t *testing.T) {
	t.Parallel()

	at := func(x, y consts.Tile) gc.Rect {
		return gc.Rect{Min: consts.Coord[consts.Tile]{X: x, Y: y}, Max: consts.Coord[consts.Tile]{X: x, Y: y}}
	}

	tests := []struct {
		name  string
		rooms []gc.Rect
		want  []string
	}{
		{
			// 太さの方向が外周へはみ出そうとする
			name:  "中心が外周の1マス内側を一周する",
			rooms: []gc.Rect{at(1, 1), at(8, 1), at(8, 8), at(1, 8), at(1, 1)},
			want: []string{
				"##########",
				"#........#",
				"#........#",
				"#..####..#",
				"#..####..#",
				"#..####..#",
				"#..####..#",
				"#........#",
				"#........#",
				"##########",
			},
		},
		{
			// 進む方向が外周の上を通る
			name:  "中心が外周の上を一周する",
			rooms: []gc.Rect{at(0, 0), at(9, 0), at(9, 9), at(0, 9), at(0, 0)},
			want: []string{
				"##########",
				"#........#",
				"#.######.#",
				"#.######.#",
				"#.######.#",
				"#.######.#",
				"#.######.#",
				"#.######.#",
				"#........#",
				"##########",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mp := newFilledPlan(10, 10, testLayoutTiles.Wall)
			mp.Rooms = tt.rooms

			connect := ConnectRooms{Tiles: testLayoutTiles}
			require.NoError(t, connect.PlanMeta(mp))

			assert.Equal(t, tt.want, renderRows(mp))
		})
	}
}

func TestScatterWalls_PlanMeta_確率1なら外周から2マス内側の床を壁にする(t *testing.T) {
	t.Parallel()

	const width, height = 10, 10
	mp := newFilledPlan(width, height, testLayoutTiles.Floor)

	scatter := ScatterWalls{Row: rawschema.ScatterWallStep{Chance: 1.0}, Tiles: testLayoutTiles}
	require.NoError(t, scatter.PlanMeta(mp))

	assert.Equal(t, []string{
		"..........",
		"..........",
		"..######..",
		"..######..",
		"..######..",
		"..######..",
		"..######..",
		"..######..",
		"..........",
		"..........",
	}, renderRows(mp))
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

	// x=3,y=3 は左右の2マスで床に接する。x=0,y=3 は上下の2マスで接するが外周なので対象外。
	// x=3,y=4 は元の盤面では x=3,y=5 の1マスにしか接しない。x=3,y=3 が床になった後の盤面を
	// 数える実装でだけ2マスになる
	mp := newPlanFromRows([]string{
		"#######",
		"#######",
		".######",
		"##.#.##",
		".######",
		"###.###",
		"#######",
	})

	widen := Widen{Row: rawschema.WidenStep{Chance: 1.0}, Tiles: testLayoutTiles}
	require.NoError(t, widen.PlanMeta(mp))

	assert.Equal(t, []string{
		"#######",
		"#######",
		".######",
		"##...##",
		".######",
		"###.###",
		"#######",
	}, renderRows(mp))
}

func TestWiden_PlanMeta_確率0なら壁を変化させない(t *testing.T) {
	t.Parallel()

	rows := []string{
		"#######",
		"#######",
		"#######",
		"##.#.##",
		"#######",
		"#######",
		"#######",
	}
	mp := newPlanFromRows(rows)

	widen := Widen{Row: rawschema.WidenStep{Chance: 0.0}, Tiles: testLayoutTiles}
	require.NoError(t, widen.PlanMeta(mp))

	assert.Equal(t, rows, renderRows(mp))
}
