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
	// 1ループ ≈ 10回の選択になるよう層を伸ばす（母港→目標で10回の辺選択）。
	// 道中はフィールド地形が主役で、合流点(隊商宿)を中央に1つ挟んで分岐→合流→再分岐する。
	layerSizes := []int{1, 3, 3, 3, 3, 1, 3, 3, 3, 3, 1}
	junctionLayer := 5
	lastLayer := len(layerSizes) - 1

	g := &Graph{}
	layers := make([][]NodeID, len(layerSizes))
	// flexible は種別を後から差し替えてよい中間ノード（合流点・前哨・目標地点を除く）。
	var flexible []NodeID
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
			if isFlexibleMiddle(l, i, junctionLayer, lastLayer) {
				flexible = append(flexible, id)
			}
		}
	}
	g.Home = layers[0][0]
	g.Goal = layers[lastLayer][0]

	// 隣接層のみ前進方向に接続する（一方向を構造で担保）。
	for l := range lastLayer {
		connectLayers(g, layers[l], layers[l+1], rng)
	}

	// 遺跡（＝ミクロ潜行の入口）の最低数を保証する。辺は種別に依存しないため
	// 接続の後に行い、辺生成の乱数列を乱さない（遺跡が足りるランはここで rng を消費しない）。
	ensureMinRuins(g, flexible, minRuinsFor(expedition), rng)
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

// isFlexibleMiddle は種別を後処理で差し替えてよい中間ノードか判定する。
// 合流点・前哨・目標地点・母港は役割が固定なので除外し、weightedMiddleType で
// 種別を決めるノードのみ true を返す（nodeTypeFor の default ケースと対応する）。
func isFlexibleMiddle(layer, idx, junctionLayer, lastLayer int) bool {
	switch {
	case layer == 0, layer == lastLayer, layer == junctionLayer:
		return false
	case layer == lastLayer-1 && idx == 0:
		return false // 前哨
	default:
		return true
	}
}

// minRuinsFor は遠征ごとの遺跡（潜行ダンジョン）の最低保証数を返す。
// マクロ移動とミクロ潜行の比重を偏らせない設計上、どの遠征でも最低限のダンジョンを
// 保証し、「街しか無く一度も潜れない」ランを防ぐ。DeepVault は潜行重心なので多め。
func minRuinsFor(exp ExpeditionType) int {
	if exp == ExpeditionDeepVault {
		return 3
	}
	return 2
}

// ensureMinRuins は柔軟中間ノードの遺跡数が min 未満なら、非遺跡ノードを決定的に
// シャッフルして不足ぶんを遺跡へ差し替える。既に足りていれば rng を消費せず即返す
// （＝遺跡が十分なランのグラフ・乱数列は不変のまま）。
func ensureMinRuins(g *Graph, flexible []NodeID, minRuins int, rng *rand.Rand) {
	count := 0
	for _, id := range flexible {
		if g.Nodes[id].Type == NodeRuin {
			count++
		}
	}
	if count >= minRuins {
		return
	}

	convertible := make([]NodeID, 0, len(flexible))
	for _, id := range flexible {
		if g.Nodes[id].Type != NodeRuin {
			convertible = append(convertible, id)
		}
	}
	rng.Shuffle(len(convertible), func(i, j int) {
		convertible[i], convertible[j] = convertible[j], convertible[i]
	})
	for _, id := range convertible {
		if count >= minRuins {
			break
		}
		g.Nodes[id].Type = NodeRuin
		g.Nodes[id].Label = fmt.Sprintf("%s-%d", nodeTypeName(NodeRuin), id)
		count++
	}
}

// weightedMiddleType は中間ノードの種別を遠征で重み付けして選ぶ。
// 道中の「歩く手触り」を主役にするため、基本プールをフィールド地形（平原/山脈/遺跡）で固め、
// 街（集落/専門店）は「たまに」＝少数に留める。合流点(隊商宿)・前哨は層構造で必ず入り
// そこでも交易できるため、明示的な街ノードは稀でよい（フィールドがベース）。
func weightedMiddleType(exp ExpeditionType, rng *rand.Rand) NodeType {
	// 基本プール：フィールド地形 8 / 街 2（＝約8割がフィールド）。
	pool := []NodeType{
		NodePlain, NodePlain, NodePlain,
		NodeMountain, NodeMountain, NodeMountain,
		NodeRuin, NodeRuin,
		NodeMarket, NodeShop,
	}
	switch exp {
	case ExpeditionDeepVault:
		pool = append(pool, NodeRuin, NodeRuin) // 潜行重心
	case ExpeditionTradeCity:
		pool = append(pool, NodeMarket, NodeShop) // 交易重心（それでもフィールドは主役）
	case ExpeditionPatron, ExpeditionFrontier:
		pool = append(pool, NodeMountain) // 辺境＝険路重心
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
	case NodePlain:
		return "plain"
	case NodeMountain:
		return "mountain"
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
