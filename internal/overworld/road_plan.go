package overworld

import (
	"cmp"
	"container/heap"
	"slices"
	"sync"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// 地上を南北に区画へ分け、区画ごとに道と集落を一度に計画する。区域と墜落現場を先に置き、区画の境目の通り道・区域の門・
// 墜落現場を行き先として、網に近い行き先から順にチャンク単位の経路でつなぐ。集落は道に接する空いたチャンクに置く。
// 計画は区画の番号と seed の純関数なので、どのチャンクから引いても同じ道と集落になる。

// planRows は区画の行数。集落のリージョンの行数で割り切れる値にし、集落のリージョンが区画の境目をまたがないようにする
const planRows consts.Chunk = 40

// roadSalt は区画の境目の通り道と集落の置き場所の抽選に混ぜる値
const roadSalt uint64 = 0x9e3779b97f4a7c15

// 経路探索の費用。曲がるごとに費用を足し、まっすぐな道を好む
const (
	roadStepCost = 1
	roadTurnCost = 2
)

// roadPort は道のチャンクの縁を道が越える所。gate なら区域の門に面し、縁の中央でなく門の通りの位置を通る
type roadPort struct {
	dir  RoadDir
	gate bool
}

// regionPlan は区画1つの計画
type regionPlan struct {
	roads       map[consts.Coord[consts.Chunk]][]roadPort
	settlements map[consts.Coord[consts.Chunk]]RoadDir // 集落と、道のある側
}

// planKey は計画を覚えておく鍵。raws は表の中身へのポインタで見分ける
type planKey struct {
	features  *[]rawschema.PointFeature
	districts *[]rawschema.District
	runSeed   uint64
	cols      consts.Chunk
	spawnRow  consts.Chunk
	region    consts.Chunk
}

// planCacheLimit を超えて覚えたら全て忘れる。北へ進む1回の遊びで引く区画は少ないので、テストで多くの seed を引くときだけ効く
const planCacheLimit = 512

var (
	planMu    sync.Mutex
	planCache = map[planKey]*regionPlan{}
)

// planAt は行 y を持つ区画の計画を返す。区画ごとに一度だけ計画し、覚えておく
func planAt(raws rawschema.Raws, runSeed uint64, y, cols, spawnRow consts.Chunk) *regionPlan {
	key := planKey{features: raws.PointFeatures, districts: raws.Districts, runSeed: runSeed, cols: cols, spawnRow: spawnRow, region: consts.FloorDiv(y, planRows)}
	planMu.Lock()
	p, ok := planCache[key]
	planMu.Unlock()
	if ok {
		return p
	}
	p = planRegion(raws, runSeed, key.region, cols, spawnRow)
	planMu.Lock()
	if len(planCache) >= planCacheLimit {
		clear(planCache)
	}
	planCache[key] = p
	planMu.Unlock()
	return p
}

// roadRow は道を持つ地物の道の行を返す。道を持つ地物が無ければ ok=false
func roadRow(raws rawschema.Raws) (rawschema.Road, bool) {
	for _, f := range raw.PtrSlice(raws.PointFeatures) {
		if f.Road != nil {
			return *f.Road, true
		}
	}
	return rawschema.Road{}, false
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

// crossingAt は区画の境目の行 b、すなわち行 b-1 と行 b の間を道が越える列を返す。両側のチャンクがどちらも通れる列から
// b と seed で引く。区域が境目をまたげば区域の門と通りが通り道になるので、列は引かず ok=false。通れる列が無くても ok=false
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
	return free[consts.Mix2D(runSeed^roadSalt, b, 0)%uint64(len(free))], true
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

// regionGrid は計画する区画の範囲と、道の通れないチャンク
type regionGrid struct {
	top, bottom, cols consts.Chunk
	blocked           map[consts.Coord[consts.Chunk]]bool
}

// newRegionGrid は区画 k の範囲を作り、区域と墜落現場のチャンクを集める
func newRegionGrid(raws rawschema.Raws, runSeed uint64, k, cols, spawnRow consts.Chunk) regionGrid {
	g := regionGrid{top: k * planRows, bottom: (k+1)*planRows - 1, cols: cols, blocked: map[consts.Coord[consts.Chunk]]bool{}}
	for y := g.top; y <= g.bottom; y++ {
		for x := range cols {
			c := consts.Coord[consts.Chunk]{X: x, Y: y}
			g.blocked[c] = blockedChunk(raws, runSeed, c, cols, spawnRow)
		}
	}
	return g
}

// passable は c が区画の中で道の通れるチャンクかを返す
func (g regionGrid) passable(c consts.Coord[consts.Chunk]) bool {
	return c.X >= 0 && c.X < g.cols && c.Y >= g.top && c.Y <= g.bottom && !g.blocked[c]
}

// nodeOf は terms のうち区画の中で道の通れるものだけを持つ行き先を返す。1つも無ければ ok=false
func (g regionGrid) nodeOf(terms ...planTerminal) (planNode, bool) {
	var n planNode
	for _, t := range terms {
		if g.passable(t.c) {
			n.terms = append(n.terms, t)
		}
	}
	return n, len(n.terms) > 0
}

// addPort は道のチャンク c に出入り口 p を足す。同じ出入り口は重ねない
func (plan *regionPlan) addPort(c consts.Coord[consts.Chunk], p roadPort) {
	if !slices.Contains(plan.roads[c], p) {
		plan.roads[c] = append(plan.roads[c], p)
	}
}

// planRegion は区画 k の道と集落を計画する
func planRegion(raws rawschema.Raws, runSeed uint64, k, cols, spawnRow consts.Chunk) *regionPlan {
	plan := &regionPlan{roads: map[consts.Coord[consts.Chunk]][]roadPort{}, settlements: map[consts.Coord[consts.Chunk]]RoadDir{}}
	if _, ok := roadRow(raws); !ok {
		return plan
	}
	g := newRegionGrid(raws, runSeed, k, cols, spawnRow)
	var start, nodes []planNode
	if x, ok := crossingAt(raws, runSeed, g.top, cols, spawnRow); ok {
		t := planTerminal{c: consts.Coord[consts.Chunk]{X: x, Y: g.top}, port: roadPort{dir: RoadN}}
		// 境目の通り道は、区画をまたいで道を続けるため初めから道にする
		plan.addPort(t.c, t.port)
		start = append(start, planNode{terms: []planTerminal{t}})
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
	if len(start) == 0 && len(nodes) > 0 {
		start, nodes = nodes[:1], nodes[1:]
	}
	connectNodes(plan, g, start, nodes)
	placeSettlements(raws, runSeed, plan, g)
	return plan
}

// districtNodes は門の外のチャンクが区画の中にある区域を行き先にする。区画の北の境目をまたぐ区域は、北の区画の網から
// 通りを通ってつながっているので網の始まり start にする
func districtNodes(raws rawschema.Raws, runSeed uint64, g regionGrid, spawnRow consts.Chunk) (start, nodes []planNode) {
	cat := zoneCatalogFrom(raws)
	for _, r := range districtsBetween(raws, runSeed, g.top-1, g.bottom+1, g.cols, spawnRow) {
		north, south := gateColumns(r, lotsOf(raws, cat, r, spawnRow))
		n, ok := g.nodeOf(
			planTerminal{c: consts.Coord[consts.Chunk]{X: north, Y: r.anchor.Y - 1}, port: roadPort{dir: RoadS, gate: true}},
			planTerminal{c: consts.Coord[consts.Chunk]{X: south, Y: r.anchor.Y + r.h}, port: roadPort{dir: RoadN, gate: true}},
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

// connectNodes は網 start に、行き先 nodes を網に近い順に経路でつなぐ。つながらない行き先は残す
func connectNodes(plan *regionPlan, g regionGrid, start, nodes []planNode) {
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
func placeSettlements(raws rawschema.Raws, runSeed uint64, plan *regionPlan, g regionGrid) {
	p := placementOf(raws, rawschema.PointFeatureKindSettlement)
	for ry := consts.FloorDiv(g.top, p.Spacing); ry <= consts.FloorDiv(g.bottom, p.Spacing); ry++ {
		type site struct {
			c, road consts.Coord[consts.Chunk]
		}
		var sites []site
		for y := max(ry*p.Spacing, g.top); y <= min((ry+1)*p.Spacing-1, g.bottom); y++ {
			for x := range g.cols {
				c := consts.Coord[consts.Chunk]{X: x, Y: y}
				if _, road := plan.roads[c]; road || g.blocked[c] {
					continue
				}
				for _, d := range roadDirs {
					n := c.Add(stepOf(d))
					if _, ok := plan.roads[n]; ok {
						sites = append(sites, site{c: c, road: n})
					}
				}
			}
		}
		if len(sites) == 0 {
			continue
		}
		s := sites[consts.Mix2D(runSeed^p.Salt^roadSalt, ry, 0)%uint64(len(sites))]
		plan.settlements[s.c] = roadDirToward(s.c, s.road)
		plan.addPort(s.road, roadPort{dir: roadDirToward(s.road, s.c)})
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
