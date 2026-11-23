package overworld

import (
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planTest* は計画を確かめる範囲。開始の行を挟んで北へ3計画区間ほど見る
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
		rows := consts.Chunk(raws.Road.SpanRows)
		for k := consts.FloorDiv(planTestTop, rows) + 1; k <= consts.FloorDiv(planTestBottom, rows); k++ {
			b := k * rows
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

func TestFacingCells_全ての集落の絵は玄関を4方向へ向けられ玄関の縁に柵が無い(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	loader, err := maptemplate.FacilityTemplates()
	require.NoError(t, err)
	for _, st := range raw.PtrSlice(raws.Settlements) {
		for _, name := range loader.FacilityChunkNames(st.Facility) {
			_, _, cells, err := loader.LoadTemplateByName(name, 1)
			require.NoError(t, err, name)
			var sides []RoadDir
			for turns := range 4 {
				for _, mirror := range []bool{false, true} {
					tc := maptemplate.TransformCells(cells, turns, mirror)
					side := entranceSide(tc)
					require.NotZero(t, side, "%s turns=%d mirror=%v の玄関の側が決まる", name, turns, mirror)
					if !slices.Contains(sides, side) {
						sides = append(sides, side)
					}
					// 玄関の位置の縁のマスは柵でない。道はここから施設へ入る
					at := entranceOf(tc, side)
					assert.False(t, isFence(edgeLine(tc, side, 0)[at]), "%s turns=%d mirror=%v side=%d", name, turns, mirror, side)
				}
			}
			assert.ElementsMatch(t, []RoadDir{RoadN, RoadE, RoadS, RoadW}, sides, name)
		}
	}
}

func TestEntranceSide(t *testing.T) {
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
		want  RoadDir
	}{
		{"柵のある絵は扉の近さでなく柵の切れ目の側", grid("ffffffff", "f+.....f", "f......f", "f.....+f", "fff,,fff"), RoadS},
		{"柵の無い絵は縁に最も近い扉の側", grid(",,,,,,", ",,,,,+", ",,,,,,"), RoadE},
		{"柵に切れ目が無ければ決まらない", grid("ffff", "f++f", "ffff"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, entranceSide(tt.cells))
		})
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
		{"縁の角の柵の無いマスは切れ目と取り違えない", grid(".ffff..ffff.", ".f........f.", ".ffffffffff."), RoadN, 6},
		{"柵が無ければ最も近い扉の並び", grid(",,,,,,,,", ",,,,,,,,", ",,,++,,,", ",,,,,,,,"), RoadN, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, entranceOf(tt.cells, tt.side))
		})
	}
}

func TestPlanAt_道の網は区域の門と墜落現場を通ってひとつながり(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	// 範囲の端の計画区間は範囲の外とつながるので、計画区間をまるごと含む範囲を見る
	rows := consts.Chunk(raws.Road.SpanRows)
	top, bottom := -5*rows, rows-1
	for seed := range uint64(20) {
		// 区域は門の出入り口どうしを、墜落現場は隣の出入り口どうしを、通り抜けとしてつなぐ
		roads := map[consts.Coord[consts.Chunk]][]roadPort{}
		for y := top; y <= bottom; y++ {
			for x := range planTestCols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if ports, _, ok := roadAt(raws, seed, c, planTestCols, planTestSpawn); ok {
					roads[c] = ports
				}
			}
		}
		require.NotEmpty(t, roads, "前提: seed=%d に道がある", seed)
		through := map[string][]consts.Coord[consts.Chunk]{}
		for c, ports := range roads {
			for _, p := range ports {
				n := c.Add(stepOf(p.dir))
				switch chunkTypeAt(raws, seed, n, planTestCols, planTestSpawn) {
				case chunkUrban:
					r, _ := districtAt(raws, seed, n, planTestCols, planTestSpawn)
					key := r.kind.Id + r.anchor.String()
					through[key] = append(through[key], c)
				case chunkCrashSite:
					through["crash"] = append(through["crash"], c)
				case chunkWasteland, chunkSettlement, chunkRoad, chunkLandmark:
				}
			}
		}
		var start consts.Coord[consts.Chunk]
		for c := range roads {
			start = c
			break
		}
		seen := map[consts.Coord[consts.Chunk]]bool{start: true}
		queue := []consts.Coord[consts.Chunk]{start}
		visit := func(n consts.Coord[consts.Chunk]) {
			if _, ok := roads[n]; ok && !seen[n] {
				seen[n] = true
				queue = append(queue, n)
			}
		}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			for _, p := range roads[c] {
				visit(c.Add(stepOf(p.dir)))
			}
			for _, group := range through {
				if slices.Contains(group, c) {
					for _, n := range group {
						visit(n)
					}
				}
			}
		}
		assert.Len(t, seen, len(roads), "seed=%d の道の網がひとつながり", seed)
	}
}

func TestPlanAt_行き先の前に集落と点在ランドマークを置かない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	reserved := 0
	for seed := range uint64(20) {
		for y := planTestTop; y <= planTestBottom; y++ {
			for c := range planAt(raws, seed, y, planTestCols, planTestSpawn).reserved {
				if c.Y != y {
					continue
				}
				reserved++
				kind := chunkTypeAt(raws, seed, c, planTestCols, planTestSpawn)
				assert.Contains(t, []chunkType{chunkRoad, chunkWasteland}, kind, "seed=%d %v", seed, c)
			}
		}
	}
	require.Positive(t, reserved, "前提: 予約したチャンクがある")
}

func TestPlanSpan_記録しても計画は変わらず最後の写しが仕上がりと一致する(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	road, ok := roadRow(raws)
	require.True(t, ok)
	for seed := range uint64(5) {
		plain := planSpan(raws, seed, road, 0, planTestCols, planTestSpawn, false)
		recorded := planSpan(raws, seed, road, 0, planTestCols, planTestSpawn, true)
		require.GreaterOrEqual(t, len(recorded.snapshots), 3, "前提: 区域、行き先、集落の段階がある")
		last := recorded.snapshots[len(recorded.snapshots)-1]
		assert.Equal(t, "settlements", last.label)
		assert.Equal(t, plain.roads, last.roads, "seed=%d", seed)
		assert.Equal(t, plain.settlements, last.settlements, "seed=%d", seed)
		assert.Equal(t, plain.reserved, last.reserved, "seed=%d", seed)
		assert.Empty(t, plain.snapshots, "記録しなければ写しを持たない")
	}
}
