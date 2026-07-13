package route

import (
	"fmt"
	"math/rand/v2"
)

// Generate は遠征と乱数シードから層状 DAG のルート網を生成する純関数。
// 同じ (expedition, seed) は同じグラフを返すため、save はグラフ本体でなくシードを
// 保存し、ロード時に再構築できる（docs/design/20260713_55.md 実装計画 Phase 2）。
//
// 不変条件（generate_test.go で検証）:
//   - 母港から目標地点へ必ず到達可能
//   - 合流点を全経路が通る（分岐→合流→再分岐）
//   - 分岐区間の各層は有効選択肢が2以上（最短一択にしない）
//
// Phase 1 は層構成を固定する: 母港 → 分岐 → 分岐 → 合流点 → 分岐 → 目標地点。
func Generate(expedition ExpeditionType, seed uint64) *Graph {
	rng := rand.New(rand.NewPCG(seed, 0))

	// 層ごとのノード数。1 の層は収束（母港・合流点・目標地点）。
	layerSizes := []int{1, 3, 3, 1, 3, 1}
	junctionLayer := 3
	lastLayer := len(layerSizes) - 1

	g := &Graph{}
	layers := make([][]NodeID, len(layerSizes))
	var nextID NodeID
	for l, size := range layerSizes {
		for i := range size {
			id := nextID
			nextID++
			nt := nodeTypeFor(l, i, junctionLayer, lastLayer, expedition, rng)
			g.Nodes = append(g.Nodes, Node{
				ID:    id,
				Type:  nt,
				Layer: l,
				Label: fmt.Sprintf("%s-%d", nodeTypeName(nt), id),
			})
			layers[l] = append(layers[l], id)
		}
	}
	g.Home = layers[0][0]
	g.Goal = layers[lastLayer][0]

	// 隣接層のみ前進方向に接続する（一方向を構造で担保）。
	for l := range lastLayer {
		connectLayers(g, layers[l], layers[l+1], rng)
	}
	return g
}

// nodeTypeFor は層とインデックスからノード種別を決める。
func nodeTypeFor(layer, idx, junctionLayer, lastLayer int, exp ExpeditionType, rng *rand.Rand) NodeType {
	switch {
	case layer == 0:
		return NodeHome
	case layer == lastLayer:
		return NodeGoal
	case layer == junctionLayer:
		return NodeJunction
	case layer == lastLayer-1 && idx == 0:
		return NodeOutpost // 目標地点手前に前哨（最終補給・最終売却点）を1つ置く
	default:
		return weightedMiddleType(exp, rng)
	}
}

// weightedMiddleType は中間ノードの種別を遠征で重み付けして選ぶ。
func weightedMiddleType(exp ExpeditionType, rng *rand.Rand) NodeType {
	pool := []NodeType{NodeMarket, NodeRuin, NodeCamp, NodeShop}
	switch exp {
	case ExpeditionDeepVault:
		pool = append(pool, NodeRuin, NodeRuin) // 潜行重心
	case ExpeditionTradeCity:
		pool = append(pool, NodeMarket, NodeMarket) // 交易重心
	case ExpeditionPatron, ExpeditionFrontier:
		// 重心なし（基本プールのまま）
	}
	return pool[rng.IntN(len(pool))]
}

// connectLayers は from 層の各ノードを to 層へ前進接続する。
// to 層が分岐（2ノード以上）なら各ノードから2本以上引き（選択肢を担保）、
// to 層の全ノードに最低1本の入辺を保証する（到達可能性）。
func connectLayers(g *Graph, from, to []NodeID, rng *rand.Rand) {
	covered := make(map[NodeID]bool)
	for _, f := range from {
		for _, tgt := range pickTargets(to, rng) {
			g.Edges = append(g.Edges, newEdge(f, tgt, rng))
			covered[tgt] = true
		}
	}
	for _, t := range to {
		if !covered[t] {
			f := from[rng.IntN(len(from))]
			g.Edges = append(g.Edges, newEdge(f, t, rng))
			covered[t] = true
		}
	}
}

// pickTargets は接続先を選ぶ。to が収束層（1ノード）なら1件、分岐層なら2件以上。
func pickTargets(to []NodeID, rng *rand.Rand) []NodeID {
	if len(to) == 1 {
		return []NodeID{to[0]}
	}
	perm := rng.Perm(len(to))
	k := 2 + rng.IntN(len(to)-1) // [2, len(to)]
	chosen := make([]NodeID, 0, k)
	for i := range k {
		chosen = append(chosen, to[perm[i]])
	}
	return chosen
}

// newEdge は辺種別をランダムに選び、その基準面数を持つ辺を作る。
func newEdge(from, to NodeID, rng *rand.Rand) Edge {
	t := EdgeType(rng.IntN(4))
	return Edge{From: from, To: to, Type: t, Faces: t.baseFaces()}
}

// nodeTypeName はラベル用の英字名を返す。
func nodeTypeName(t NodeType) string {
	switch t {
	case NodeHome:
		return "home"
	case NodeMarket:
		return "market"
	case NodeShop:
		return "shop"
	case NodeRuin:
		return "ruin"
	case NodeCamp:
		return "camp"
	case NodeOutpost:
		return "outpost"
	case NodeJunction:
		return "junction"
	case NodeGoal:
		return "goal"
	default:
		return "node"
	}
}
