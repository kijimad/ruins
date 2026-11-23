package overworld

import (
	"cmp"
	"container/heap"
	"slices"
	"sync"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// 地上を南北に計画区間へ分け、計画区間ごとに道と集落を一度に計画する。区域と墜落現場を先に置き、計画区間の境目の通り道・区域の門・
// 墜落現場を行き先として、網に近い行き先から順にチャンク単位の経路でつなぐ。集落は道に接する空いたチャンクに置く。
// 計画は計画区間の番号と seed の純関数なので、どのチャンクから引いても同じ道と集落になる。

// 計画区間の境目の通り道と、集落の置き場所の抽選に混ぜる値
const (
	crossingSalt       uint64 = 0x9e3779b97f4a7c15
	settlementSiteSalt uint64 = 0xc2b2ae3d27d4eb4f
)

// 経路探索の費用。曲がるごとに費用を足し、まっすぐな道を好む
const (
	roadStepCost = 1
	roadTurnCost = 2
)

// portLane は出入り口が縁に沿って通る位置の決め方
type portLane string

const (
	laneCenter   portLane = ""         // 縁の中央。ゼロ値
	laneGate     portLane = "gate"     // 区域の門の通り
	laneEntrance portLane = "entrance" // 隣の集落の施設の玄関
)

// roadPort は道のチャンクの縁を道が越える所
type roadPort struct {
	dir  RoadDir
	lane portLane
}

// roadPlan は計画区間1つの計画
type roadPlan struct {
	roads       map[consts.Coord[consts.Chunk]][]roadPort
	settlements map[consts.Coord[consts.Chunk]]RoadDir // 集落と、道のある側
	// reserved は行き先が網につながる所。道が通らなかった所も、門や墜落現場の前を塞がないよう集落と点在ランドマークを置かない
	reserved map[consts.Coord[consts.Chunk]]bool
}

// planKey は計画を覚えておく鍵。raws は表の中身へのポインタで見分けるので、同じ raws の表をその場で書き換えると古い計画を引く。
// raws を変えるときはコピーしてから変える
type planKey struct {
	road      *rawschema.Road
	features  *[]rawschema.PointFeature
	districts *[]rawschema.District
	runSeed   uint64
	cols      consts.Chunk
	spawnRow  consts.Chunk
	span      consts.Chunk
}

// planCacheLimit を超えて覚えたら全て忘れる。北へ進む1回の遊びで引く計画区間は少ないので、テストで多くの seed を引くときだけ効く
const planCacheLimit = 512

var (
	planMu    sync.Mutex
	planCache = map[planKey]*roadPlan{}
)

// planAt は行 y を持つ計画区間の計画を返す。計画区間ごとに一度だけ計画し、覚えておく
func planAt(raws rawschema.Raws, runSeed uint64, y, cols, spawnRow consts.Chunk) *roadPlan {
	road, ok := roadRow(raws)
	if !ok {
		return &roadPlan{}
	}
	key := planKey{road: raws.Road, features: raws.PointFeatures, districts: raws.Districts, runSeed: runSeed, cols: cols, spawnRow: spawnRow, span: consts.FloorDiv(y, consts.Chunk(road.SpanRows))}
	planMu.Lock()
	p, found := planCache[key]
	planMu.Unlock()
	if found {
		return p
	}
	p = planSpan(raws, runSeed, road, key.span, cols, spawnRow)
	planMu.Lock()
	if len(planCache) >= planCacheLimit {
		clear(planCache)
	}
	planCache[key] = p
	planMu.Unlock()
	return p
}

// roadRow は raw の道の表を返す。道の表が無ければ ok=false
func roadRow(raws rawschema.Raws) (rawschema.Road, bool) {
	if raws.Road == nil {
		return rawschema.Road{}, false
	}
	return *raws.Road, true
}

// blockedChunk は c が道の通れないチャンクかを返す。区域と墜落現場は持ち主が決まっているので道を通さない
func blockedChunk(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk) bool {
	if c == spawnChunkOf(cols, spawnRow) {
		return true
	}
	_, ok := districtAt(raws, runSeed, c, cols, spawnRow)
	return ok
}

// districtsBetween は絶対チャンク行 top から bottom に掛かる区域を北から順に返す
func districtsBetween(raws rawschema.Raws, runSeed uint64, top, bottom, cols, spawnRow consts.Chunk) []districtRegion {
	var out []districtRegion
	for _, d := range districtsFrom(raws) {
		spacing := consts.Chunk(d.row.Spacing)
		for ry := consts.FloorDiv(top, spacing) - 1; ry <= consts.FloorDiv(bottom, spacing); ry++ {
			r := d.regionIn(runSeed, ry, cols)
			if r.anchor.Y > bottom || r.anchor.Y+r.h-1 < top {
				continue
			}
			if got, ok := districtAt(raws, runSeed, r.anchor, cols, spawnRow); ok && got == r {
				out = append(out, r)
			}
		}
	}
	slices.SortFunc(out, func(a, b districtRegion) int {
		return cmp.Or(cmp.Compare(a.anchor.Y, b.anchor.Y), cmp.Compare(a.anchor.X, b.anchor.X))
	})
	return out
}

// crossingAt は計画区間の境目の行 b、すなわち行 b-1 と行 b の間を道が越える列を返す。両側のチャンクがどちらも通れる列から
// b と seed で引く。区域が境目をまたげば区域の門と通りが通り道になるので、列は引かず ok=false。通れる列が無くても ok=false。
// 境目の両側の計画区間が、それぞれ自分の網をこの通り道へつなげることを前提にする。区域の間に隙間を空けるので、区域の外は
// ひと続きになり、この前提が成り立つ。TestPlanAt_道の出入り口の先は道か行き先につながる が確かめる
func crossingAt(raws rawschema.Raws, runSeed uint64, b, cols, spawnRow consts.Chunk) (consts.Chunk, bool) {
	for _, r := range districtsBetween(raws, runSeed, b-1, b, cols, spawnRow) {
		if r.anchor.Y <= b-1 && r.anchor.Y+r.h-1 >= b {
			return 0, false
		}
	}
	var free []consts.Chunk
	for x := range cols {
		if !blockedChunk(raws, runSeed, consts.Coord[consts.Chunk]{X: x, Y: b - 1}, cols, spawnRow) &&
			!blockedChunk(raws, runSeed, consts.Coord[consts.Chunk]{X: x, Y: b}, cols, spawnRow) {
			free = append(free, x)
		}
	}
	if len(free) == 0 {
		return 0, false
	}
	return free[consts.Mix2D(runSeed^crossingSalt, b, 0)%uint64(len(free))], true
}

// planTerminal は行き先が網につながる所。c は道のチャンクになり、port は行き先の側の出入り口
type planTerminal struct {
	c    consts.Coord[consts.Chunk]
	port roadPort
}

// planNode は道の行き先。terms のどれかが網につながれば、行き先の他の terms も網につながる
type planNode struct {
	terms []planTerminal
}

// spanGrid は計画する計画区間の範囲と、道の通れないチャンク
type spanGrid struct {
	top, bottom, cols consts.Chunk
	blocked           map[consts.Coord[consts.Chunk]]bool
}

// newSpanGrid は rows 行の計画区間 k の範囲を作り、区域と墜落現場のチャンクを集める
func newSpanGrid(raws rawschema.Raws, runSeed uint64, k, rows, cols, spawnRow consts.Chunk) spanGrid {
	g := spanGrid{top: k * rows, bottom: (k+1)*rows - 1, cols: cols, blocked: map[consts.Coord[consts.Chunk]]bool{}}
	for y := g.top; y <= g.bottom; y++ {
		for x := range cols {
			c := consts.Coord[consts.Chunk]{X: x, Y: y}
			g.blocked[c] = blockedChunk(raws, runSeed, c, cols, spawnRow)
		}
	}
	return g
}

// passable は c が計画区間の中で道の通れるチャンクかを返す
func (g spanGrid) passable(c consts.Coord[consts.Chunk]) bool {
	return c.X >= 0 && c.X < g.cols && c.Y >= g.top && c.Y <= g.bottom && !g.blocked[c]
}

// nodeOf は terms のうち計画区間の中で道の通れるものだけを持つ行き先を返す。1つも無ければ ok=false
func (g spanGrid) nodeOf(terms ...planTerminal) (planNode, bool) {
	var n planNode
	for _, t := range terms {
		if g.passable(t.c) {
			n.terms = append(n.terms, t)
		}
	}
	return n, len(n.terms) > 0
}

// addPort は道のチャンク c に出入り口 p を足す。同じ出入り口は重ねない
func (plan *roadPlan) addPort(c consts.Coord[consts.Chunk], p roadPort) {
	if !slices.Contains(plan.roads[c], p) {
		plan.roads[c] = append(plan.roads[c], p)
	}
}

// planSpan は道の表 road の行数で区切った計画区間 k の道と集落を計画する
func planSpan(raws rawschema.Raws, runSeed uint64, road rawschema.Road, k, cols, spawnRow consts.Chunk) *roadPlan {
	plan := &roadPlan{roads: map[consts.Coord[consts.Chunk]][]roadPort{}, settlements: map[consts.Coord[consts.Chunk]]RoadDir{}, reserved: map[consts.Coord[consts.Chunk]]bool{}}
	g := newSpanGrid(raws, runSeed, k, consts.Chunk(road.SpanRows), cols, spawnRow)
	var start, nodes []planNode
	if x, ok := crossingAt(raws, runSeed, g.top, cols, spawnRow); ok {
		start = append(start, planNode{terms: []planTerminal{{c: consts.Coord[consts.Chunk]{X: x, Y: g.top}, port: roadPort{dir: RoadN}}}})
	}
	districtStart, districtNodes := districtNodes(raws, runSeed, g, spawnRow)
	start = append(start, districtStart...)
	nodes = append(nodes, districtNodes...)
	s := spawnChunkOf(cols, spawnRow)
	if n, ok := g.nodeOf(
		planTerminal{c: consts.Coord[consts.Chunk]{X: s.X, Y: s.Y - 1}, port: roadPort{dir: RoadS}},
		planTerminal{c: consts.Coord[consts.Chunk]{X: s.X, Y: s.Y + 1}, port: roadPort{dir: RoadN}},
	); ok && s.Y >= g.top && s.Y <= g.bottom {
		nodes = append(nodes, n)
	}
	if x, ok := crossingAt(raws, runSeed, g.bottom+1, cols, spawnRow); ok {
		nodes = append(nodes, planNode{terms: []planTerminal{{c: consts.Coord[consts.Chunk]{X: x, Y: g.bottom}, port: roadPort{dir: RoadS}}}})
	}
	// 網の始まりは1つにし、残りの始まりも行き先として経路でつなぐ。始まりのままだと、経路の出発点に使われなかった門の先に道が届かない
	if len(start) > 1 {
		start, nodes = start[:1], append(start[1:], nodes...)
	}
	// 北の境目から網を始められなければ、最初の行き先から始める
	if len(start) == 0 && len(nodes) > 0 {
		start, nodes = nodes[:1], nodes[1:]
	}
	// 網の始まりは、経路の出発点に使われなくても道にする。北の計画区間から来た道や通りがここで途切れない
	for _, n := range start {
		for _, t := range n.terms {
			plan.addPort(t.c, t.port)
		}
	}
	for _, n := range append(slices.Clone(start), nodes...) {
		for _, t := range n.terms {
			plan.reserved[t.c] = true
		}
	}
	connectNodes(plan, g, start, nodes)
	placeSettlements(raws, runSeed, plan, g)
	return plan
}

// districtNodes は門の外のチャンクが計画区間の中にある区域を行き先にする。計画区間の北の境目をまたぐ区域は、北の計画区間の網から
// 通りを通ってつながっているので網の始まり start にする
func districtNodes(raws rawschema.Raws, runSeed uint64, g spanGrid, spawnRow consts.Chunk) (start, nodes []planNode) {
	cat := zoneCatalogFrom(raws)
	for _, r := range districtsBetween(raws, runSeed, g.top-1, g.bottom+1, g.cols, spawnRow) {
		north, south := gateColumns(r, lotsOf(raws, cat, r, spawnRow))
		n, ok := g.nodeOf(
			planTerminal{c: consts.Coord[consts.Chunk]{X: north, Y: r.anchor.Y - 1}, port: roadPort{dir: RoadS, lane: laneGate}},
			planTerminal{c: consts.Coord[consts.Chunk]{X: south, Y: r.anchor.Y + r.h}, port: roadPort{dir: RoadN, lane: laneGate}},
		)
		switch {
		case !ok:
		case r.anchor.Y < g.top && r.anchor.Y+r.h-1 >= g.top:
			start = append(start, n)
		default:
			nodes = append(nodes, n)
		}
	}
	return start, nodes
}

// connectNodes は網 start に、行き先 nodes を網に近い順に経路でつなぐ。経路が無い行き先は残して戻る。区域の間に隙間を空けるので
// 区域の外はひと続きになり、どの行き先にも経路がある前提にする。TestPlanAt_道の網は区域の門と墜落現場を通ってひとつながり が確かめる
func connectNodes(plan *roadPlan, g spanGrid, start, nodes []planNode) {
	// attach は網につながった所。値は、そこを道のチャンクにするときに足す行き先の出入り口
	attach := map[consts.Coord[consts.Chunk]][]roadPort{}
	join := func(n planNode) {
		for _, t := range n.terms {
			attach[t.c] = append(attach[t.c], t.port)
		}
	}
	for _, n := range start {
		join(n)
	}
	for len(nodes) > 0 {
		targets := map[consts.Coord[consts.Chunk]]int{}
		for i, n := range nodes {
			for _, t := range n.terms {
				if _, ok := targets[t.c]; !ok {
					targets[t.c] = i
				}
			}
		}
		path := shortestRoad(attach, targets, g.passable)
		if path == nil {
			return
		}
		for _, p := range attach[path[0]] {
			plan.addPort(path[0], p)
		}
		for i := 0; i+1 < len(path); i++ {
			plan.addPort(path[i], roadPort{dir: roadDirToward(path[i], path[i+1])})
			plan.addPort(path[i+1], roadPort{dir: roadDirToward(path[i+1], path[i])})
		}
		end := path[len(path)-1]
		i := targets[end]
		for _, t := range nodes[i].terms {
			if t.c == end {
				plan.addPort(end, t.port)
			}
		}
		join(nodes[i])
		for _, c := range path {
			if _, ok := attach[c]; !ok {
				attach[c] = nil
			}
		}
		nodes = slices.Delete(nodes, i, i+1)
	}
}

// placeSettlements は集落のリージョンごとに、道のチャンクに接する空いたチャンクから集落を1つ引き、道から集落の側へ出入り口を足す
func placeSettlements(raws rawschema.Raws, runSeed uint64, plan *roadPlan, g spanGrid) {
	p := placementOf(raws, rawschema.PointFeatureKindSettlement)
	for ry := consts.FloorDiv(g.top, p.Spacing); ry <= consts.FloorDiv(g.bottom, p.Spacing); ry++ {
		type site struct {
			c, road consts.Coord[consts.Chunk]
		}
		// 1つのチャンクは、接する道が何本でも候補に1回だけ入れる。道は roadDirs の順で最初に接するものにする
		var sites []site
		for y := max(ry*p.Spacing, g.top); y <= min((ry+1)*p.Spacing-1, g.bottom); y++ {
			for x := range g.cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if _, road := plan.roads[c]; road || g.blocked[c] || plan.reserved[c] {
					continue
				}
				for _, d := range roadDirs {
					n := c.Add(stepOf(d))
					if _, road := plan.roads[n]; road {
						sites = append(sites, site{c: c, road: n})
						break
					}
				}
			}
		}
		if len(sites) == 0 {
			continue
		}
		s := sites[consts.Mix2D(runSeed^settlementSiteSalt, ry, 0)%uint64(len(sites))]
		plan.settlements[s.c] = roadDirToward(s.c, s.road)
		plan.addPort(s.road, roadPort{dir: roadDirToward(s.road, s.c), lane: laneEntrance})
	}
}

// roadState は経路探索の状態。曲がる費用を数えるため、来た向きを持つ
type roadState struct {
	c   consts.Coord[consts.Chunk]
	dir RoadDir // 来た向き。0 は出発点
}

type roadItem struct {
	s    roadState
	cost int
	seq  int
}

type roadQueue []roadItem

func (q roadQueue) Len() int { return len(q) }
func (q roadQueue) Less(i, j int) bool {
	return cmp.Or(cmp.Compare(q[i].cost, q[j].cost), cmp.Compare(q[i].seq, q[j].seq)) < 0
}
func (q roadQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *roadQueue) Push(x any) {
	it, ok := x.(roadItem)
	if !ok {
		panic("roadQueue: not a roadItem")
	}
	*q = append(*q, it)
}

func (q *roadQueue) Pop() any {
	old := *q
	it := old[len(old)-1]
	*q = old[:len(old)-1]
	return it
}

// roadDirs は経路が隣へ進む方角を、探索で試す順に並べる
var roadDirs = [4]RoadDir{RoadN, RoadE, RoadS, RoadW}

// stepOf は方角 d の隣への差を返す
func stepOf(d RoadDir) consts.Coord[consts.Chunk] {
	switch d {
	case RoadN:
		return consts.Coord[consts.Chunk]{X: 0, Y: -1}
	case RoadE:
		return consts.Coord[consts.Chunk]{X: 1, Y: 0}
	case RoadS:
		return consts.Coord[consts.Chunk]{X: 0, Y: 1}
	case RoadW:
		return consts.Coord[consts.Chunk]{X: -1, Y: 0}
	}
	panic("stepOf: not a single direction")
}

// shortestRoad は網 from のどれかから行き先 targets のどれかまでの、費用の最も小さい経路を出発点から順に返す。
// passable なチャンクだけを通る。経路が無ければ nil
func shortestRoad(from map[consts.Coord[consts.Chunk]][]roadPort, targets map[consts.Coord[consts.Chunk]]int, passable func(consts.Coord[consts.Chunk]) bool) []consts.Coord[consts.Chunk] {
	starts := make([]consts.Coord[consts.Chunk], 0, len(from))
	for c := range from {
		starts = append(starts, c)
	}
	slices.SortFunc(starts, func(a, b consts.Coord[consts.Chunk]) int { return cmp.Or(cmp.Compare(a.Y, b.Y), cmp.Compare(a.X, b.X)) })
	dist := map[roadState]int{}
	prev := map[roadState]roadState{}
	q := &roadQueue{}
	seq := 0
	for _, c := range starts {
		s := roadState{c: c}
		dist[s] = 0
		heap.Push(q, roadItem{s: s, seq: seq})
		seq++
	}
	for q.Len() > 0 {
		it, ok := heap.Pop(q).(roadItem)
		if !ok {
			panic("roadQueue: not a roadItem")
		}
		if it.cost > dist[it.s] {
			continue
		}
		if _, ok := targets[it.s.c]; ok {
			var path []consts.Coord[consts.Chunk]
			for s := it.s; ; s = prev[s] {
				path = append(path, s.c)
				if s.dir == 0 {
					break
				}
			}
			slices.Reverse(path)
			return path
		}
		for _, d := range roadDirs {
			n := it.s.c.Add(stepOf(d))
			if !passable(n) {
				continue
			}
			cost := it.cost + roadStepCost
			if it.s.dir != 0 && it.s.dir != d {
				cost += roadTurnCost
			}
			ns := roadState{c: n, dir: d}
			if old, ok := dist[ns]; ok && old <= cost {
				continue
			}
			dist[ns] = cost
			prev[ns] = it.s
			heap.Push(q, roadItem{s: ns, cost: cost, seq: seq})
			seq++
		}
	}
	return nil
}

// roadAt は c が道のチャンクなら出入り口と道の行を返す
func roadAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk) ([]roadPort, rawschema.Road, bool) {
	road, ok := roadRow(raws)
	if !ok {
		return nil, rawschema.Road{}, false
	}
	ports, ok := planAt(raws, runSeed, c.Y, cols, spawnRow).roads[c]
	return ports, road, ok
}

// settlementFacing は集落 c から見て道のある側を返す。集落でなければ0
func settlementFacing(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk], cols, spawnRow consts.Chunk) RoadDir {
	return planAt(raws, runSeed, c.Y, cols, spawnRow).settlements[c]
}
