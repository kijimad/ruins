package components

import "github.com/kijimaD/ruins/internal/consts"

// DistanceMapObstacleCost は、壊せる置物やキャラクターのいるマスへ入る重み。空いたマスの重み1より大きいので、下る者は
// 近い回り道があればそちらを通り、無ければそのマスの手前で待つ
const DistanceMapObstacleCost = 5

// distanceMapDirs は距離の地図を広げ、下る8方向
var distanceMapDirs = [...]consts.Coord[consts.Tile]{
	{X: 0, Y: -1}, {X: 0, Y: 1}, {X: -1, Y: 0}, {X: 1, Y: 0},
	{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1},
}

// DistanceMap は狙う的から現ステージの各マスへの、重み付きの距離。下る者はこの値が小さくなる隣へ進む。全員が同じ1枚を
// 下るので、1体ごとに道を探さずに済む
type DistanceMap struct {
	Width, Height consts.Tile
	// Cost は各マスへ入る重み。添字は y*Width+x。0 は通れない。Reset と Spread の間でだけ書く
	Cost []int32
	// Blocked は通れない地形か置物のマス。斜めの角の判定に使う。Reset と Spread の間でだけ書く
	Blocked []bool
	// Built は構築済みか。偽なら次に引いたときに作る
	Built bool
	// dist は各マスの距離。届かないマスは負
	dist []int32
	// buckets は距離ごとのバケツの輪。作るたびに使い回す
	buckets [DistanceMapObstacleCost + 1][]int32
}

// CubeDistanceMap はキューブからの距離の地図を保持するシングルトン。キューブが無ければプレイヤーを的にする
type CubeDistanceMap struct {
	DistanceMap
}

// NewCubeDistanceMap は未構築の空の地図を作る
func NewCubeDistanceMap() *CubeDistanceMap {
	return &CubeDistanceMap{}
}

// PlayerDistanceMap はプレイヤーからの距離の地図を保持するシングルトン
type PlayerDistanceMap struct {
	DistanceMap
}

// NewPlayerDistanceMap は未構築の空の地図を作る
func NewPlayerDistanceMap() *PlayerDistanceMap {
	return &PlayerDistanceMap{}
}

// Reset は寸法を width と height にし、全てのマスを重み1の通れるマスに戻す。配列は使い回す
func (m *DistanceMap) Reset(width, height consts.Tile) {
	m.Width, m.Height = width, height
	n := int(width) * int(height)
	if cap(m.Cost) < n {
		m.Cost = make([]int32, n)
		m.Blocked = make([]bool, n)
		m.dist = make([]int32, n)
	}
	m.Cost, m.Blocked, m.dist = m.Cost[:n], m.Blocked[:n], m.dist[:n]
	for i := range m.Cost {
		m.Cost[i] = 1
		m.Blocked[i] = false
		m.dist[i] = -1
	}
}

// Index は c の添字を返す
func (m *DistanceMap) Index(c consts.Coord[consts.Tile]) int {
	return int(c.Y)*int(m.Width) + int(c.X)
}

// InBounds は c が地図の中かを返す
func (m *DistanceMap) InBounds(c consts.Coord[consts.Tile]) bool {
	return c.X >= 0 && c.Y >= 0 && c.X < m.Width && c.Y < m.Height
}

// Spread は書き出した重みで、的 targets から距離を広げる。斜めは縦横の隣がどちらも Blocked なら抜けない
func (m *DistanceMap) Spread(targets []consts.Coord[consts.Tile]) {
	// 重みは1か DistanceMapObstacleCost なので、距離ごとのバケツを輪にして持てば足りる
	for i := range m.buckets {
		m.buckets[i] = m.buckets[i][:0]
	}
	pending := 0
	for _, t := range targets {
		if !m.InBounds(t) {
			continue
		}
		i := m.Index(t)
		m.dist[i] = 0
		m.buckets[0] = append(m.buckets[0], int32(i))
		pending++
	}
	width := int(m.Width)
	for d := int32(0); pending > 0; d++ {
		slot := int(d) % len(m.buckets)
		cur := m.buckets[slot]
		pending -= len(cur)
		for _, ci := range cur {
			if m.dist[ci] != d {
				continue
			}
			cx, cy := int(ci)%width, int(ci)/width
			for _, dir := range distanceMapDirs {
				nx, ny := cx+int(dir.X), cy+int(dir.Y)
				if nx < 0 || ny < 0 || nx >= width || ny >= int(m.Height) {
					continue
				}
				ni := ny*width + nx
				step := m.Cost[ni]
				if step == 0 {
					continue
				}
				if dir.X != 0 && dir.Y != 0 && m.Blocked[cy*width+nx] && m.Blocked[ny*width+cx] {
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
		// 広げる先は d+1 から d+DistanceMapObstacleCost で、同じ輪の位置へは積まれないので、空にして使い回す
		m.buckets[slot] = cur[:0]
	}
}

// Dist は c の距離を返す。届かないマスと地図の外は false
func (m *DistanceMap) Dist(c consts.Coord[consts.Tile]) (int32, bool) {
	if !m.InBounds(c) {
		return 0, false
	}
	d := m.dist[m.Index(c)]
	return d, d >= 0
}

// Steps は from の隣のうち、距離が一番小さいマスを全て、向きの順に buf へ足して返す。from の距離は隣の
// 一番小さい値に from へ入る重さを足したものなので、from に誰もいなかったときの距離と比べれば、一番小さい隣だけが
// 残る。斜めは広げるときと同じ規則で選ぶ。下る隣が無ければ空を返す。from は通れるマスである前提で、通れないマスに
// 立つと距離の同じ隣まで返す
func (m *DistanceMap) Steps(from consts.Coord[consts.Tile], buf []consts.Coord[consts.Tile]) []consts.Coord[consts.Tile] {
	steps := buf[:0]
	d, ok := m.Dist(from)
	if !ok {
		return steps
	}
	here := d - m.Cost[m.Index(from)] + 1
	for _, dir := range distanceMapDirs {
		next := from.Add(dir)
		if d, ok := m.Dist(next); ok && d < here && m.canPassDiagonal(from, dir) {
			steps = append(steps, next)
		}
	}
	return steps
}

// canPassDiagonal は広げるときと同じく、縦横の隣がどちらも通れないマスなら斜めに抜けさせない。下る先が地図の
// 中にあるので、縦横の隣も地図の中にある
func (m *DistanceMap) canPassDiagonal(from, d consts.Coord[consts.Tile]) bool {
	if d.X == 0 || d.Y == 0 {
		return true
	}
	return !m.Blocked[m.Index(from.Add(consts.Coord[consts.Tile]{X: d.X}))] || !m.Blocked[m.Index(from.Add(consts.Coord[consts.Tile]{Y: d.Y}))]
}
