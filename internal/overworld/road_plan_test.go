package overworld

import (
	"math/rand/v2"
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planSpan は計画を確かめる行の範囲。開始の行を挟んで北へ3計画区間ほど見る
const (
	planTestTop    consts.Chunk = -100
	planTestBottom consts.Chunk = 19
	planTestCols   consts.Chunk = 9
	planTestSpawn  consts.Chunk = 4
)

func TestPlanAt_道の出入り口の先は道か行き先につながる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	opposite := map[RoadDir]RoadDir{RoadN: RoadS, RoadS: RoadN, RoadE: RoadW, RoadW: RoadE}
	for seed := range uint64(20) {
		for y := planTestTop; y <= planTestBottom; y++ {
			for x := range planTestCols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				ports, _, ok := roadAt(raws, seed, c, planTestCols, planTestSpawn)
				if !ok {
					continue
				}
				for _, p := range ports {
					n := c.Add(stepOf(p.dir))
					kind := chunkTypeAt(raws, seed, n, planTestCols, planTestSpawn)
					if p.lane == laneGate {
						assert.Equal(t, chunkUrban, kind, "seed=%d %v の門の出入り口の先は区域", seed, c)
						continue
					}
					switch kind {
					case chunkRoad:
						back, _, _ := roadAt(raws, seed, n, planTestCols, planTestSpawn)
						assert.Contains(t, back, roadPort{dir: opposite[p.dir]}, "seed=%d %v から %v へ道が続く", seed, c, n)
					case chunkSettlement:
						assert.Equal(t, opposite[p.dir], settlementFacing(raws, seed, n, planTestCols, planTestSpawn), "seed=%d 集落 %v は道の側を向く", seed, n)
					case chunkCrashSite:
					default:
						assert.Fail(t, "出入り口の先が道でも行き先でもない", "seed=%d %v -> %v %s", seed, c, n, kind)
					}
				}
			}
		}
	}
}

func TestPlanAt_区域と集落と墜落現場に道が届く(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	reached := func(seed uint64, c consts.Coord[consts.Chunk], want RoadDir, lane portLane) bool {
		ports, _, ok := roadAt(raws, seed, c, planTestCols, planTestSpawn)
		return ok && slices.Contains(ports, roadPort{dir: want, lane: lane})
	}
	var districts, settlements int
	for seed := range uint64(20) {
		// 区域は北か南の門の外に、門へ向かう出入り口がある
		for _, r := range districtsBetween(raws, seed, planTestTop+1, planTestBottom-1, planTestCols, planTestSpawn) {
			if r.anchor.Y-1 < planTestTop || r.anchor.Y+r.h > planTestBottom {
				continue
			}
			districts++
			north, south := gateColumns(r, lotsOf(raws, zoneCatalogFrom(raws), r, planTestSpawn))
			assert.True(t,
				reached(seed, consts.Coord[consts.Chunk]{X: north, Y: r.anchor.Y - 1}, RoadS, laneGate) ||
					reached(seed, consts.Coord[consts.Chunk]{X: south, Y: r.anchor.Y + r.h}, RoadN, laneGate),
				"seed=%d 区域 %s %v", seed, r.kind.Id, r.anchor)
		}
		// 墜落現場は北か南の隣に、墜落現場へ向かう出入り口がある
		s := spawnChunkOf(planTestCols, planTestSpawn)
		assert.True(t,
			reached(seed, consts.Coord[consts.Chunk]{X: s.X, Y: s.Y - 1}, RoadS, laneCenter) ||
				reached(seed, consts.Coord[consts.Chunk]{X: s.X, Y: s.Y + 1}, RoadN, laneCenter),
			"seed=%d 墜落現場", seed)
		// 集落は道のある側の隣に、集落へ向かう出入り口がある
		for y := planTestTop; y <= planTestBottom; y++ {
			for c, facing := range planAt(raws, seed, y, planTestCols, planTestSpawn).settlements {
				if c.Y != y {
					continue
				}
				settlements++
				back := map[RoadDir]RoadDir{RoadN: RoadS, RoadS: RoadN, RoadE: RoadW, RoadW: RoadE}
				assert.True(t, reached(seed, c.Add(stepOf(facing)), back[facing], laneEntrance), "seed=%d 集落 %v", seed, c)
			}
		}
	}
	require.Positive(t, districts, "前提: 区域がある")
	require.Positive(t, settlements, "前提: 集落がある")
}

func TestPlanAt_計画区間の境目で道が途切れない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	crossings := 0
	for seed := range uint64(20) {
		for k := consts.FloorDiv(planTestTop, planRows) + 1; k <= consts.FloorDiv(planTestBottom, planRows); k++ {
			b := k * planRows
			x, ok := crossingAt(raws, seed, b, planTestCols, planTestSpawn)
			if !ok {
				// 区域の間に隙間を空けるので、通り道が引けないのは区域が境目をまたぐときだけ
				straddled := false
				for _, r := range districtsBetween(raws, seed, b-1, b, planTestCols, planTestSpawn) {
					straddled = straddled || (r.anchor.Y <= b-1 && r.anchor.Y+r.h-1 >= b)
				}
				assert.True(t, straddled, "seed=%d 境目 %d", seed, b)
				continue
			}
			crossings++
			below, _, _ := roadAt(raws, seed, consts.Coord[consts.Chunk]{X: x, Y: b}, planTestCols, planTestSpawn)
			above, _, _ := roadAt(raws, seed, consts.Coord[consts.Chunk]{X: x, Y: b - 1}, planTestCols, planTestSpawn)
			assert.Contains(t, below, roadPort{dir: RoadN}, "seed=%d 境目 %d の南", seed, b)
			assert.Contains(t, above, roadPort{dir: RoadS}, "seed=%d 境目 %d の北", seed, b)
		}
	}
	require.Positive(t, crossings, "前提: 計画区間の境目を越える道がある")
}

func TestFacingCells_扉が道の側を向く向きで置く(t *testing.T) {
	t.Parallel()

	// 北の縁の真ん中に扉がある 5x3 の絵
	cells := make([][]maptemplate.MapCell, 3)
	for y := range cells {
		cells[y] = make([]maptemplate.MapCell, 5)
	}
	cells[0][2].Prop = "door"
	require.Equal(t, RoadN, doorSide(cells), "前提: 扉は北を向く")

	for _, facing := range roadDirs {
		for seed := range uint64(8) {
			got := facingCells(cells, facing, rand.New(rand.NewPCG(seed, 0)))
			assert.Equal(t, facing, doorSide(got), "facing=%d seed=%d", facing, seed)
		}
	}
}

func TestNewChunkGen_集落の扉は道の側を向き道が集落の縁まで届く(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	var seed uint64
	var s consts.Coord[consts.Chunk]
	var facing RoadDir
	for sd := uint64(1); sd < 50 && facing == 0; sd++ {
		for y := consts.Chunk(1); y < 9 && facing == 0; y++ {
			for x := range planTestCols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				// 帯を作らずに生成するので、生成の側は開始の行を0とみなす
				if f := settlementFacing(raws, sd, c, planTestCols, 0); f != 0 && chunkTypeAt(raws, sd, c, planTestCols, 0) == chunkSettlement {
					seed, s, facing = sd, c, f
					break
				}
			}
		}
	}
	require.NotZero(t, facing, "前提: 集落が見つかる")

	world := testutil.InitTestWorld(t)
	gen := NewChunkGen(world, seed, chunkW, chunkH, planTestCols, mapplanner.PlannerTypeOverworldField)
	road := s.Add(stepOf(facing))
	for _, c := range []consts.Coord[consts.Chunk]{s, road} {
		require.NoError(t, gen(c, c.X.Tiles(chunkW), c.Y.Tiles(chunkH)))
	}

	// 道は、道のチャンクの集落に面する縁から、集落の施設の絵の手前まで、玄関の位置を通って舗装される
	l, err := settlementLayoutAt(raws, seed, s, planTestCols, 0, chunkW, chunkH)
	require.NoError(t, err)
	s0 := consts.Coord[consts.Tile]{X: s.X.Tiles(chunkW), Y: s.Y.Tiles(chunkH)}
	w, h := consts.Tile(len(l.cells[0])), consts.Tile(len(l.cells))
	// path は道のチャンクの縁のマスから、施設の絵の手前のマスまで、集落のチャンクの中の座標で並べる
	var path []consts.Coord[consts.Tile]
	switch facing {
	case RoadN:
		for y := consts.Tile(-1); y < l.origin.Y; y++ {
			path = append(path, consts.Coord[consts.Tile]{X: l.entrance, Y: y})
		}
	case RoadS:
		for y := l.origin.Y + h; y <= chunkH; y++ {
			path = append(path, consts.Coord[consts.Tile]{X: l.entrance, Y: y})
		}
	case RoadW:
		for x := consts.Tile(-1); x < l.origin.X; x++ {
			path = append(path, consts.Coord[consts.Tile]{X: x, Y: l.entrance})
		}
	case RoadE:
		for x := l.origin.X + w; x <= chunkW; x++ {
			path = append(path, consts.Coord[consts.Tile]{X: x, Y: l.entrance})
		}
	}
	tiles := (&tileIndex{world: world, loX: 0, hiX: planTestCols.Tiles(chunkW)}).get()
	row, ok := roadRow(raws)
	require.True(t, ok)
	paved := make([]string, 0, len(path))
	want := make([]string, 0, len(path))
	for _, p := range path {
		paved = append(paved, tileIDAt(world, tiles, s0.Add(p)))
		want = append(want, row.Tile)
	}
	assert.Equal(t, want, paved, "道は集落の施設の玄関の前まで届く")

	// 集落の施設の扉のうち縁に最も近いものが道の側にある
	best, side := chunkW+chunkH, RoadDir(0)
	q := ecs.NewFilter2[gc.Door, gc.GridElement](world.ECS).Query()
	for q.Next() {
		_, g := q.Get()
		p := g.Sub(s0)
		if p.X < 0 || p.X >= chunkW || p.Y < 0 || p.Y >= chunkH {
			continue
		}
		for _, e := range []struct {
			d    consts.Tile
			side RoadDir
		}{{p.Y, RoadN}, {chunkH - 1 - p.Y, RoadS}, {p.X, RoadW}, {chunkW - 1 - p.X, RoadE}} {
			if e.d < best {
				best, side = e.d, e.side
			}
		}
	}
	assert.Equal(t, facing, side, "扉は道の側を向く")
}

func TestPlanRows_集落のリージョンの行数で割り切れる(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	// 割り切れないと、計画区間の境目をまたぐ集落のリージョンに両側の計画区間が集落を1つずつ置く
	assert.Zero(t, planRows%placementOf(raws, rawschema.PointFeatureKindSettlement).Spacing)
}

func TestEntranceOf(t *testing.T) {
	t.Parallel()

	grid := func(rows ...string) [][]maptemplate.MapCell {
		cells := make([][]maptemplate.MapCell, len(rows))
		for y, r := range rows {
			cells[y] = make([]maptemplate.MapCell, len(r))
			for x, ch := range r {
				switch ch {
				case 'f':
					cells[y][x].Prop = "fence"
				case '+':
					cells[y][x].Prop = "door"
				}
			}
		}
		return cells
	}
	tests := []struct {
		name  string
		cells [][]maptemplate.MapCell
		side  RoadDir
		want  consts.Tile
	}{
		{"柵の切れ目の真ん中を大きい側へ寄せる", grid("fffff,,fff", "f........f", "ffffffffff"), RoadN, 6},
		{"南の縁の柵の切れ目", grid("ffffffffff", "f........f", "ff,,,fffff"), RoadS, 3},
		{"西の縁の柵の切れ目は縦に数える", grid("fff", "f..", ",..", ",..", "fff"), RoadW, 3},
		{"柵が無ければ最も近い扉の並び", grid(",,,,,,,,", ",,,,,,,,", ",,,++,,,", ",,,,,,,,"), RoadN, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, entranceOf(tt.cells, tt.side))
		})
	}
}
