package activity

import (
	"cmp"
	"slices"

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
	// cost は各マスへ入る重み。0 は通れない。作るたびに空間の索引から一度だけ書き出す
	cost []int32
	// blocked は通れない地形か置物のマス。斜めの角の判定に使う
	blocked []bool
	// buckets は距離ごとのバケツの輪。作るたびに使い回す
	buckets [raidObstacleCost + 1][]int32
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
	if cap(m.cost) < n {
		m.cost = make([]int32, n)
		m.blocked = make([]bool, n)
	}
	m.cost, m.blocked = m.cost[:n], m.blocked[:n]

	m.fillCosts(world, si)

	// 重みは1か raidObstacleCost なので、距離ごとのバケツを輪にして持てば足りる
	for i := range m.buckets {
		m.buckets[i] = m.buckets[i][:0]
	}
	pending := 0
	for _, t := range targets {
		if !m.inBounds(t) {
			continue
		}
		i := m.index(t)
		m.dist[i] = 0
		m.buckets[0] = append(m.buckets[0], int32(i))
		pending++
	}
	w := int(m.width)
	for d := int32(0); pending > 0; d++ {
		slot := int(d) % len(m.buckets)
		cur := m.buckets[slot]
		pending -= len(cur)
		for _, ci := range cur {
			if m.dist[ci] != d {
				continue
			}
			cx, cy := int(ci)%w, int(ci)/w
			for _, dir := range raidMapDirs {
				nx, ny := cx+int(dir.X), cy+int(dir.Y)
				if nx < 0 || ny < 0 || nx >= w || ny >= int(m.height) {
					continue
				}
				ni := ny*w + nx
				step := m.cost[ni]
				if step == 0 {
					continue
				}
				if dir.X != 0 && dir.Y != 0 && m.blocked[cy*w+nx] && m.blocked[ny*w+cx] {
					continue
				}
				nd := d + step
				if m.dist[ni] >= 0 && m.dist[ni] <= nd {
					continue
				}
				m.dist[ni] = nd
				ns := int(nd) % len(m.buckets)
				m.buckets[ns] = append(m.buckets[ns], int32(ni))
				pending++
			}
		}
		// 広げる先は d+1 から d+raidObstacleCost で、同じ輪の位置へは積まれないので、空にして使い回す
		m.buckets[slot] = cur[:0]
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

// Steps は from の隣のうち、from に誰もいなかったときの距離より小さいマスを、距離の小さい順に buf へ足して返す。
// 同じ距離なら raidMapDirs の順にする。斜めは CanMoveTo と同じ規則で選ぶ。下る隣が無ければ空を返す
func (m *RaidMap) Steps(world w.World, from consts.Coord[consts.Tile], buf []consts.Coord[consts.Tile]) []consts.Coord[consts.Tile] {
	steps := buf[:0]
	d, ok := m.Dist(from)
	if !ok {
		return steps
	}
	// 立っている自分のマスは重く数えられているので、重さ1で入ったときの距離と比べる。比べないと後ろのマスまで下る隣に入る
	here := d - m.cost[m.index(from)] + 1
	si := query.GetSpatialIndex(world)
	if si == nil {
		return steps
	}
	for _, dir := range raidMapDirs {
		next := from.Add(dir)
		if d, ok := m.Dist(next); ok && d < here && m.canPassDiagonal(si, from, dir) {
			steps = append(steps, next)
		}
	}
	slices.SortStableFunc(steps, func(a, b consts.Coord[consts.Tile]) int {
		return cmp.Compare(m.dist[m.index(a)], m.dist[m.index(b)])
	})
	return steps
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

// fillCosts は各マスへ入る重みと、通れないマスを空間の索引から書き出す。通れるマスは1、HP を持たず通れない
// 地形と置物は0、HP を持つ通れない置物は raidObstacleCost にする。キャラクターのマスは、プレイヤーと味方なら1、
// それ以外なら raidObstacleCost にする
func (m *RaidMap) fillCosts(world w.World, si *gc.SpatialIndex) {
	for i := range m.cost {
		m.cost[i] = 1
		m.blocked[i] = false
	}
	for g, blocked := range si.Blocked {
		if !blocked || !m.inBounds(g.Coord) {
			continue
		}
		i := m.index(g.Coord)
		m.cost[i] = 0
		m.blocked[i] = true
	}
	q := query.ActiveFilter3[gc.Prop, gc.HP, gc.GridElement](world).Query()
	for q.Next() {
		_, _, g := q.Get()
		if m.inBounds(g.Coord) && m.blocked[m.index(g.Coord)] {
			m.cost[m.index(g.Coord)] = raidObstacleCost
		}
	}
	for g, e := range si.Characters {
		if !m.inBounds(g.Coord) || m.blocked[m.index(g.Coord)] {
			continue
		}
		if query.IsAlly(world, e) {
			m.cost[m.index(g.Coord)] = 1
		} else {
			m.cost[m.index(g.Coord)] = raidObstacleCost
		}
	}
}
