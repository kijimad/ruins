package activity

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// raidObstacleCost は HP を持つ置物と、襲撃の敵が待つことになるキャラクターのマスを通る重み
const raidObstacleCost = 5

// raidMapDirs は距離の地図を広げ、下る8方向
var raidMapDirs = [...]consts.Coord[consts.Tile]{
	{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0},
	{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1},
}

// RaidMap は狙う的から現ステージの各マスへの、重み付きの距離。襲撃の敵はこの値が小さくなる隣へ進む
type RaidMap struct {
	width, height consts.Tile
	// dist は各マスの距離。添字は y*width+x。届かないマスは負
	dist []int32
}

// BuildRaidMap は的 targets から重み付きの距離の地図を作る。buf があればその配列を使い回す。
// 通れるマスと、プレイヤーと味方のマスは重み1、HP を持つ通れない置物と、それ以外のキャラクターのマスは
// 重み raidObstacleCost で通れるものとして数える。HP を持たず通れない壁や地形は数えない。
// 斜めは CanMoveTo と同じく、縦横の隣がどちらも通れないマスなら抜けない
func BuildRaidMap(world w.World, targets []consts.Coord[consts.Tile], buf *RaidMap) *RaidMap {
	m := buf
	if m == nil {
		m = &RaidMap{}
	}
	si := query.GetSpatialIndex(world)
	if si == nil {
		m.width, m.height, m.dist = 0, 0, m.dist[:0]
		return m
	}
	m.width, m.height = si.MapWidth, si.MapHeight
	n := int(m.width) * int(m.height)
	if cap(m.dist) < n {
		m.dist = make([]int32, n)
	}
	m.dist = m.dist[:n]
	for i := range m.dist {
		m.dist[i] = -1
	}

	breakable := breakableTiles(world, si)
	cost := func(c consts.Coord[consts.Tile]) int32 {
		key := gc.GridElement{Coord: c}
		if si.Blocked[key] {
			if breakable[c] {
				return raidObstacleCost
			}
			return 0
		}
		if e, ok := si.Characters[key]; ok && !query.IsAlly(world, e) {
			return raidObstacleCost
		}
		return 1
	}

	// 重みは1か raidObstacleCost なので、距離ごとのバケツを輪にして持てば足りる
	buckets := make([][]consts.Coord[consts.Tile], raidObstacleCost+1)
	for _, t := range targets {
		if !m.inBounds(t) {
			continue
		}
		m.dist[m.index(t)] = 0
		buckets[0] = append(buckets[0], t)
	}
	pending := len(buckets[0])
	for d := int32(0); pending > 0; d++ {
		slot := int(d) % len(buckets)
		cur := buckets[slot]
		buckets[slot] = nil
		pending -= len(cur)
		for _, c := range cur {
			if m.dist[m.index(c)] != d {
				continue
			}
			for _, dir := range raidMapDirs {
				next := c.Add(dir)
				if !m.inBounds(next) || !m.canPassDiagonal(si, c, dir) {
					continue
				}
				step := cost(next)
				if step == 0 {
					continue
				}
				nd := d + step
				i := m.index(next)
				if m.dist[i] >= 0 && m.dist[i] <= nd {
					continue
				}
				m.dist[i] = nd
				ns := int(nd) % len(buckets)
				buckets[ns] = append(buckets[ns], next)
				pending++
			}
		}
	}
	return m
}

// Dist は c の距離を返す。届かないマスと地図の外は false
func (m *RaidMap) Dist(c consts.Coord[consts.Tile]) (int32, bool) {
	if !m.inBounds(c) {
		return 0, false
	}
	d := m.dist[m.index(c)]
	return d, d >= 0
}

// NextStep は from の隣のうち、距離が from より小さく一番小さいマスを返す。斜めは CanMoveTo と同じ規則で選ぶ。
// 無ければ false を返す
func (m *RaidMap) NextStep(world w.World, from consts.Coord[consts.Tile]) (consts.Coord[consts.Tile], bool) {
	best, ok := m.Dist(from)
	if !ok {
		return consts.Coord[consts.Tile]{}, false
	}
	si := query.GetSpatialIndex(world)
	if si == nil {
		return consts.Coord[consts.Tile]{}, false
	}
	var step consts.Coord[consts.Tile]
	found := false
	for _, dir := range raidMapDirs {
		next := from.Add(dir)
		d, ok := m.Dist(next)
		if !ok || d >= best || !m.canPassDiagonal(si, from, dir) {
			continue
		}
		best, step, found = d, next, true
	}
	return step, found
}

func (m *RaidMap) inBounds(c consts.Coord[consts.Tile]) bool {
	return c.X >= 0 && c.Y >= 0 && c.X < m.width && c.Y < m.height
}

func (m *RaidMap) index(c consts.Coord[consts.Tile]) int {
	return int(c.Y)*int(m.width) + int(c.X)
}

// canPassDiagonal は CanMoveTo と同じく、縦横の隣がどちらも通れないマスなら斜めに抜けさせない
func (m *RaidMap) canPassDiagonal(si *gc.SpatialIndex, cur, d consts.Coord[consts.Tile]) bool {
	if d.X == 0 || d.Y == 0 {
		return true
	}
	return !si.IsBlocked(cur.Add(consts.Coord[consts.Tile]{X: d.X})) || !si.IsBlocked(cur.Add(consts.Coord[consts.Tile]{Y: d.Y}))
}

// breakableTiles は通れないマスにある、HP を持つ置物の位置を返す。扉が実例になる
func breakableTiles(world w.World, si *gc.SpatialIndex) map[consts.Coord[consts.Tile]]bool {
	out := map[consts.Coord[consts.Tile]]bool{}
	q := query.ActiveFilter3[gc.Prop, gc.HP, gc.GridElement](world).Query()
	for q.Next() {
		_, _, g := q.Get()
		if si.Blocked[*g] {
			out[g.Coord] = true
		}
	}
	return out
}
