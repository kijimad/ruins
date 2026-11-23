package overworld

import (
	"math/rand/v2"
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planSpan は計画を確かめる行の範囲。開始の行を挟んで北へ3区画ほど見る
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
					n := c.Add(roadSteps[slices.Index([]RoadDir{RoadN, RoadE, RoadS, RoadW}, p.dir)])
					kind := chunkTypeAt(raws, seed, n, planTestCols, planTestSpawn)
					if p.gate {
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
	reached := func(seed uint64, c consts.Coord[consts.Chunk], want RoadDir, gate bool) bool {
		ports, _, ok := roadAt(raws, seed, c, planTestCols, planTestSpawn)
		return ok && slices.Contains(ports, roadPort{dir: want, gate: gate})
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
				reached(seed, consts.Coord[consts.Chunk]{X: north, Y: r.anchor.Y - 1}, RoadS, true) ||
					reached(seed, consts.Coord[consts.Chunk]{X: south, Y: r.anchor.Y + r.h}, RoadN, true),
				"seed=%d 区域 %s %v", seed, r.kind.Id, r.anchor)
		}
		// 墜落現場は北か南の隣に、墜落現場へ向かう出入り口がある
		s := spawnChunkOf(planTestCols, planTestSpawn)
		assert.True(t,
			reached(seed, consts.Coord[consts.Chunk]{X: s.X, Y: s.Y - 1}, RoadS, false) ||
				reached(seed, consts.Coord[consts.Chunk]{X: s.X, Y: s.Y + 1}, RoadN, false),
			"seed=%d 墜落現場", seed)
		// 集落は道のある側の隣に、集落へ向かう出入り口がある
		for y := planTestTop; y <= planTestBottom; y++ {
			for c, facing := range planAt(raws, seed, y, planTestCols, planTestSpawn).settlements {
				if c.Y != y {
					continue
				}
				settlements++
				steps := map[RoadDir]consts.Coord[consts.Chunk]{RoadN: {X: 0, Y: -1}, RoadE: {X: 1, Y: 0}, RoadS: {X: 0, Y: 1}, RoadW: {X: -1, Y: 0}}
				back := map[RoadDir]RoadDir{RoadN: RoadS, RoadS: RoadN, RoadE: RoadW, RoadW: RoadE}
				assert.True(t, reached(seed, c.Add(steps[facing]), back[facing], false), "seed=%d 集落 %v", seed, c)
			}
		}
	}
	require.Positive(t, districts, "前提: 区域がある")
	require.Positive(t, settlements, "前提: 集落がある")
}

func TestPlanAt_区画の境目で道が途切れない(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	crossings := 0
	for seed := range uint64(20) {
		for k := consts.FloorDiv(planTestTop, planRows) + 1; k <= consts.FloorDiv(planTestBottom, planRows); k++ {
			b := k * planRows
			x, ok := crossingAt(raws, seed, b, planTestCols, planTestSpawn)
			if !ok {
				continue
			}
			crossings++
			below, _, _ := roadAt(raws, seed, consts.Coord[consts.Chunk]{X: x, Y: b}, planTestCols, planTestSpawn)
			above, _, _ := roadAt(raws, seed, consts.Coord[consts.Chunk]{X: x, Y: b - 1}, planTestCols, planTestSpawn)
			assert.Contains(t, below, roadPort{dir: RoadN}, "seed=%d 境目 %d の南", seed, b)
			assert.Contains(t, above, roadPort{dir: RoadS}, "seed=%d 境目 %d の北", seed, b)
		}
	}
	require.Positive(t, crossings, "前提: 区画の境目を越える道がある")
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

	for _, facing := range []RoadDir{RoadN, RoadE, RoadS, RoadW} {
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
	steps := map[RoadDir]consts.Coord[consts.Chunk]{RoadN: {X: 0, Y: -1}, RoadE: {X: 1, Y: 0}, RoadS: {X: 0, Y: 1}, RoadW: {X: -1, Y: 0}}
	road := s.Add(steps[facing])
	for _, c := range []consts.Coord[consts.Chunk]{s, road} {
		require.NoError(t, gen(c, c.X.Tiles(chunkW), c.Y.Tiles(chunkH)))
	}

	// 道のチャンクの、集落に面する縁の真ん中が舗装される
	edge := map[RoadDir]consts.Coord[consts.Tile]{
		RoadN: {X: chunkW / 2, Y: chunkH - 1}, RoadS: {X: chunkW / 2, Y: 0},
		RoadE: {X: 0, Y: chunkH / 2}, RoadW: {X: chunkW - 1, Y: chunkH / 2},
	}[facing]
	tiles := (&tileIndex{world: world, loX: 0, hiX: planTestCols.Tiles(chunkW)}).get()
	road0 := consts.Coord[consts.Tile]{X: road.X.Tiles(chunkW), Y: road.Y.Tiles(chunkH)}
	row, ok := roadRow(raws)
	require.True(t, ok)
	assert.Equal(t, row.Tile, tileIDAt(world, tiles, road0.Add(edge)), "道は集落の縁まで届く")

	// 集落の施設の扉のうち縁に最も近いものが道の側にある
	s0 := consts.Coord[consts.Tile]{X: s.X.Tiles(chunkW), Y: s.Y.Tiles(chunkH)}
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
