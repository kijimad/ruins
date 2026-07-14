package route

import (
	"fmt"
	"math/rand/v2"
)

// Generate は遠征と乱数シードからレーン方式のルート網を生成する純関数。
// 同じ (expedition, seed) は同じグラフを返すため、save はグラフ本体でなくシードを
// 保存し、ロード時に再構築できる（docs/design/20260713_55.md 実装計画 Phase 2）。
//
// 構造（レーン方式）: 母港から複数の並走レーンへ分岐し、レーン内は一本道で
// 「しばらく合流できない」。全レーンは合流点(隊商宿)で一度だけ合流して再び分岐する。
// これでメッシュ（毎ステップ合流/分岐して経路が絡む）を避け、「どのレーン＝地形の連なりに
// 賭けるか」を commit する選択になる。
//
// 不変条件（generate_test.go で検証）:
//   - 母港から目標地点へ必ず到達可能
//   - 合流点を全経路が通る（分岐→合流→再分岐）
//   - 分岐点（母港・合流点）は選択肢が2以上（ルートを選べる）
func Generate(expedition ExpeditionType, seed uint64) *Graph {
	rng := rand.New(rand.NewPCG(seed, 0))

	g := &Graph{}
	var nextID NodeID
	var flexible []NodeID // 種別を後処理で差し替えてよい地形ノード（遺跡最低保証に使う）
	layer := 0

	addNode := func(nt NodeType) NodeID {
		id := nextID
		nextID++
		g.Nodes = append(g.Nodes, Node{
			ID: id, Type: nt, Layer: layer,
			Label: fmt.Sprintf("%s-%d", nodeTypeName(nt), id),
		})
		return id
	}

	// addSegment は from（分岐元）から lanes 本の並走レーンを生やし、各レーン末尾を返す。
	// レーン内は一本道で交差しない＝合流点まで合流できない。地形ノードは flexible に積む。
	// レーン数・レーン長はセグメントごとに変え、毎回同じ対称形にならないようにする。
	addSegment := func(from NodeID, lanes, laneLen int) []NodeID {
		laneNodes := make([][]NodeID, lanes)
		for range laneLen {
			for j := range lanes {
				id := addNode(weightedMiddleType(expedition, rng))
				flexible = append(flexible, id)
				laneNodes[j] = append(laneNodes[j], id)
			}
			layer++
		}
		for j := range lanes {
			g.Edges = append(g.Edges, newEdge(from, laneNodes[j][0], rng))
			for i := 0; i+1 < laneLen; i++ {
				g.Edges = append(g.Edges, newEdge(laneNodes[j][i], laneNodes[j][i+1], rng))
			}
		}
		ends := make([]NodeID, lanes)
		for j := range lanes {
			ends[j] = laneNodes[j][laneLen-1]
		}
		return ends
	}

	// converge は複数レーン末尾を1つの収束ノードへ合流させる。
	converge := func(ends []NodeID, nt NodeType) NodeID {
		node := addNode(nt)
		layer++
		for _, e := range ends {
			g.Edges = append(g.Edges, newEdge(e, node, rng))
		}
		return node
	}

	// 合流点(隊商宿)は「全ルートが通る唯一の括れ」なので1つに固定（＝2セグメント）。
	// 形の変化はレーン数・レーン長・合流点位置（セグメント長の非対称）で出す。
	const numSegments = 2
	g.Home = addNode(NodeHome)
	layer++
	from := g.Home
	for s := range numSegments {
		lanes := 2 + rng.IntN(3)   // 2〜4 レーン（分岐の選択肢数）
		laneLen := 2 + rng.IntN(2) // 2〜3 ノード（合流までの長さ。過密回避で控えめ）
		ends := addSegment(from, lanes, laneLen)
		if s < numSegments-1 {
			from = converge(ends, NodeJunction) // 合流点で合流→再分岐
		} else {
			outpost := converge(ends, NodeOutpost) // 最終セグメントは前哨で合流
			g.Goal = addNode(NodeGoal)
			g.Edges = append(g.Edges, newEdge(outpost, g.Goal, rng))
		}
	}

	// 遺跡（＝ミクロ潜行の入口）の最低数を保証する。辺は種別に依存しないため
	// 接続の後に行い、辺生成の乱数列を乱さない（遺跡が足りるランはここで rng を消費しない）。
	ensureMinRuins(g, flexible, minRuinsFor(expedition), rng)
	return g
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
// 地形の頻度は「平原/山脈＝ベース（歩く地形の主役）＞ 遺跡＝たまの潜行スポット ＞ 店/村＝稀」。
// 遺跡は重い潜行ダンジョンなので少数に留め、道中は平原/山脈で歩かせる。合流点(隊商宿)・前哨は
// 層構造で必ず入り交易もそこで足りるため、明示的な街ノードは稀でよい。
func weightedMiddleType(exp ExpeditionType, rng *rand.Rand) NodeType {
	// 基本プール：平原/山脈 8（ベース）／遺跡 1（たまの潜行）／街 2（稀）。
	pool := []NodeType{
		NodePlain, NodePlain, NodePlain, NodePlain,
		NodeMountain, NodeMountain, NodeMountain, NodeMountain,
		NodeRuin,
		NodeMarket, NodeShop,
	}
	switch exp {
	case ExpeditionDeepVault:
		pool = append(pool, NodeRuin) // 潜行重心（それでも平原/山脈が主役の範囲）
	case ExpeditionTradeCity:
		pool = append(pool, NodeMarket, NodeShop) // 交易重心（街が少し増える）
	case ExpeditionPatron, ExpeditionFrontier:
		pool = append(pool, NodeMountain) // 辺境＝険路重心
	}
	return pool[rng.IntN(len(pool))]
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
