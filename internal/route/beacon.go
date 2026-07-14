package route

import "math/rand/v2"

// Beacon は FTL 風の停留点。到達すると Event（NodeType）が起きる。
type Beacon struct {
	ID     NodeID
	Column int      // 列（母港=0、目標=最終列）。前進の骨格
	Event  NodeType // 停留点で起きること（遺跡=潜行/村=交易/山脈=吹雪/平原=無風/…）
}

// BeaconMap は FTL 風の停留点マップ。プレイヤーは隣接する次の停留点へ「ジャンプ」する
// （連続移動はしない）。各停留点はイベント/選択で、寒波前線が背後の列から迫る＝艦隊圧。
//
// Grid（連続移動）に代わるモデル。移動を消して決定を主役にし、広域の作業感を避ける。
type BeaconMap struct {
	Beacons    []Beacon
	Home, Goal NodeID
	edges      map[NodeID][]NodeID // 前方（次列）への接続
}

// BeaconByID は ID から停留点を返す（無ければ nil）。
func (m *BeaconMap) BeaconByID(id NodeID) *Beacon {
	for i := range m.Beacons {
		if m.Beacons[i].ID == id {
			return &m.Beacons[i]
		}
	}
	return nil
}

// Outgoing は現在の停留点からジャンプできる次の停留点を返す。
func (m *BeaconMap) Outgoing(id NodeID) []NodeID {
	return m.edges[id]
}

// GenerateBeacons は遠征とシードから FTL/StS 風の停留点マップを生成する純関数。
// 列ごとに 2〜3 の停留点を置き、各停留点から次列へ 1〜2 本つないで「毎列が選択」になるようにする。
// 中間停留点のイベントは遠征で重み付けし、遺跡（潜行）は疎に置く。
func GenerateBeacons(expedition ExpeditionType, seed uint64) *BeaconMap {
	rng := rand.New(rand.NewPCG(seed, 0))

	const middleCols = 5 // 中間列数（母港・目標を除く）
	m := &BeaconMap{edges: map[NodeID][]NodeID{}}
	cols := make([][]NodeID, middleCols+2)
	var nextID NodeID

	add := func(col int, ev NodeType) NodeID {
		id := nextID
		nextID++
		m.Beacons = append(m.Beacons, Beacon{ID: id, Column: col, Event: ev})
		cols[col] = append(cols[col], id)
		return id
	}

	m.Home = add(0, NodeHome)
	for c := 1; c <= middleCols; c++ {
		n := 2 + rng.IntN(2) // 2〜3 停留点
		for range n {
			add(c, beaconEvent(expedition, rng))
		}
	}
	m.Goal = add(middleCols+1, NodeGoal)

	// 前方接続（各停留点→次列の1〜2箇所。次列の全停留点に入辺を保証＝到達可能）
	for c := 0; c <= middleCols; c++ {
		connectBeaconColumn(m, cols[c], cols[c+1], rng)
	}
	return m
}

// connectBeaconColumn は from 列の各停留点を to 列へ前方接続する。
func connectBeaconColumn(m *BeaconMap, from, to []NodeID, rng *rand.Rand) {
	covered := map[NodeID]bool{}
	for _, f := range from {
		k := 1
		if len(to) > 1 {
			k = 1 + rng.IntN(2) // 1〜2 本
		}
		perm := rng.Perm(len(to))
		for i := 0; i < k && i < len(to); i++ {
			t := to[perm[i]]
			m.edges[f] = append(m.edges[f], t)
			covered[t] = true
		}
	}
	// 入辺の無い停留点を救済（到達可能性）
	for _, t := range to {
		if !covered[t] {
			f := from[rng.IntN(len(from))]
			m.edges[f] = append(m.edges[f], t)
		}
	}
}

// beaconEvent は中間停留点のイベント種別を選ぶ。大半は軽いイベント（無風/吹雪/野営）で、
// 遺跡（潜行）・村（交易）は疎に置く。遠征で重み付けする。
func beaconEvent(exp ExpeditionType, rng *rand.Rand) NodeType {
	// 平原=無風・山脈=吹雪(選択)・野営=休息 を厚めに、遺跡/村/専門店は少数。
	pool := []NodeType{
		NodePlain, NodePlain, NodeMountain, NodeMountain, NodeCamp,
		NodeRuin, NodeMarket, NodeShop,
	}
	switch exp {
	case ExpeditionDeepVault:
		pool = append(pool, NodeRuin) // 潜行重心
	case ExpeditionTradeCity:
		pool = append(pool, NodeMarket, NodeShop) // 交易重心
	case ExpeditionPatron, ExpeditionFrontier:
		pool = append(pool, NodeMountain) // 険路重心
	}
	return pool[rng.IntN(len(pool))]
}
