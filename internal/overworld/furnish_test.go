package overworld

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// genFacilityChunk は指定種別の建物チャンクを1つ探して生成し、その world と座標を返す。
func genFacilityChunk(t *testing.T, kind facilityKind) w.World {
	t.Helper()
	const rows consts.Chunk = 9
	for cy := range rows {
		for cx := range consts.Chunk(80) {
			c := worldstream.ChunkCoord{X: cx, Y: cy}
			if f, _, ok := cityChunkInfo(7, c, rows); ok && facilityCatalog[f].kind == kind {
				world := testutil.InitTestWorld(t)
				gen := NewChunkGen(world, 7, 20, 20, rows, worldstream.ChunkCoord{X: -1}, mapplanner.PlannerTypeOverworldField)
				require.NoError(t, gen(c, 0, 0))
				return world
			}
		}
	}
	require.Failf(t, "未発見", "種別 %d の建物チャンクが無い", kind)
	return w.World{}
}

// namedCoords は名前一致のエンティティ座標を集める。
func namedCoords(world w.World, name string) []consts.Coord[consts.Tile] {
	var out []consts.Coord[consts.Tile]
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.Name.Get(e).Name == name {
			out = append(out, world.Components.GridElement.Get(e).Coord)
		}
	}
	return out
}

func cheby(a, b consts.Coord[consts.Tile]) int {
	dx, dy := int(a.X-b.X), int(a.Y-b.Y)
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return max(dx, dy)
}

// TestFurnishRoom_商店のレジは入口の脇に置かれる は、役割ベース配置の中核ルール
// 「レジは入口付近」が成り立つことを固定する。ランダム散布への退行を検知する。
func TestFurnishRoom_商店のレジは入口の脇に置かれる(t *testing.T) {
	t.Parallel()

	world := genFacilityChunk(t, facilityStore)
	doors := namedCoords(world, "扉")
	regs := namedCoords(world, "register")
	require.NotEmpty(t, doors, "前提: 扉がある")
	require.NotEmpty(t, regs, "前提: レジがある")

	// いずれかのレジが、いずれかの扉のすぐ脇(チェビシェフ距離2以内)にある
	near := false
	for _, r := range regs {
		for _, d := range doors {
			if cheby(r, d) <= 2 {
				near = true
			}
		}
	}
	assert.True(t, near, "レジは入口の脇に置かれる。ランダム散布なら成り立たない")
}

// TestFurnishRoom_棚は通路を挟んで並ぶ は、棚が壁一面のベタ置きでなく通路を挟むことを確かめる。
// 部屋が狭いと棚は1列になるので、棚列が2列以上に広がるときだけ通路(棚の無い列)を要求する。
func TestFurnishRoom_棚は通路を挟んで並ぶ(t *testing.T) {
	t.Parallel()

	world := genFacilityChunk(t, facilityStore)
	shelves := namedCoords(world, "goods_shelf")
	require.NotEmpty(t, shelves, "前提: 店舗フロアに棚がある")

	cols := map[consts.Tile]int{}
	for _, s := range shelves {
		cols[s.X]++
	}
	minC, maxC := consts.Tile(1<<30), consts.Tile(-(1 << 30))
	for x := range cols {
		minC, maxC = min(minC, x), max(maxC, x)
	}
	if maxC-minC < 2 {
		return // 棚が1列に収まる狭い部屋。通路は問えない
	}
	gaps := 0
	for x := minC; x <= maxC; x++ {
		if cols[x] == 0 {
			gaps++
		}
	}
	assert.Positive(t, gaps, "棚の列が2列以上に広がるなら間に通路がある")
}

// TestFurnishRoom_内装は決定的 は、同じ seed と座標なら内装配置が完全に一致することを固定する。
func TestFurnishRoom_内装は決定的(t *testing.T) {
	t.Parallel()

	snapshot := func() map[string]int {
		world := genFacilityChunk(t, facilityStore)
		m := map[string]int{}
		q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
		for q.Next() {
			e := q.Entity()
			if world.Components.Tile.Has(e) {
				continue
			}
			p := world.Components.GridElement.Get(e).Coord
			m[world.Components.Name.Get(e).Name] += int(p.X)*1000 + int(p.Y)
		}
		return m
	}
	a, b := snapshot(), snapshot()
	assert.Equal(t, a, b, "内装は決定的で再生成しても一致する")
}
