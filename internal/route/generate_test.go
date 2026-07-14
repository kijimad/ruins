package route

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// seedCount は不変条件を検証するシード数。生成のランダム性に対して頑健にする。
const seedCount = 50

func TestGenerate_Deterministic(t *testing.T) {
	t.Parallel()
	// 同じ (expedition, seed) は同じグラフを返す（save のシード再構築の前提）
	a := Generate(ExpeditionDeepVault, 42)
	b := Generate(ExpeditionDeepVault, 42)
	assert.Equal(t, a, b)
}

func TestGenerate_Reachable(t *testing.T) {
	t.Parallel()
	for seed := range seedCount {
		g := Generate(ExpeditionDeepVault, uint64(seed))
		assert.Truef(t, reachableWithout(g, g.Home, g.Goal, -1),
			"seed=%d: 母港から目標地点へ到達可能であること", seed)
	}
}

func TestGenerate_JunctionOnAllPaths(t *testing.T) {
	t.Parallel()
	for seed := range seedCount {
		g := Generate(ExpeditionTradeCity, uint64(seed))
		j := junctionNode(g)
		assert.NotNilf(t, j, "seed=%d: 合流点が存在すること", seed)
		// 合流点を通らずに母港→目標地点へ行けない＝全経路が合流点を通る
		assert.Falsef(t, reachableWithout(g, g.Home, g.Goal, j.ID),
			"seed=%d: 合流点を通らない経路は存在しないこと", seed)
	}
}

func TestGenerate_BranchLayersHaveChoices(t *testing.T) {
	t.Parallel()
	for seed := range seedCount {
		g := Generate(ExpeditionDeepVault, uint64(seed))
		for _, n := range g.Nodes {
			if !hasBranchingNext(g, n) {
				continue
			}
			assert.GreaterOrEqualf(t, len(g.Outgoing(n.ID)), 2,
				"seed=%d node=%d(layer %d): 分岐区間の選択肢は2以上（最短一択にしない）",
				seed, n.ID, n.Layer)
		}
	}
}

func TestGenerate_FaceVarietyExists(t *testing.T) {
	t.Parallel()
	// 面数に差がある＝近道/迂回のトレードオフが成立する（面数だけを差にしない前提）
	g := Generate(ExpeditionDeepVault, 1)
	faces := map[int]bool{}
	for _, e := range g.Edges {
		faces[e.Faces] = true
	}
	assert.Greater(t, len(faces), 1)
}

func TestGenerate_EdgesConnectAdjacentLayersForward(t *testing.T) {
	t.Parallel()
	// 辺は隣接層のみ前進方向（一方向を構造で担保）
	for seed := range seedCount {
		g := Generate(ExpeditionPatron, uint64(seed))
		for _, e := range g.Edges {
			from := g.NodeByID(e.From)
			to := g.NodeByID(e.To)
			assert.Equalf(t, from.Layer+1, to.Layer,
				"seed=%d edge %d->%d: 隣接層を前進すること", seed, e.From, e.To)
		}
	}
}

func TestGenerate_GuaranteesMinRuins(t *testing.T) {
	t.Parallel()
	// どの遠征・シードでも遺跡（ミクロ潜行の入口）が最低数だけ生成される。
	// これを欠くと「街しか無く一度も潜れない」ラン（macro/micro の比重崩壊）が起きる。
	for _, exp := range []ExpeditionType{
		ExpeditionDeepVault, ExpeditionTradeCity, ExpeditionPatron, ExpeditionFrontier,
	} {
		want := minRuinsFor(exp)
		for seed := range seedCount {
			g := Generate(exp, uint64(seed))
			ruins := 0
			for _, n := range g.Nodes {
				if n.Type == NodeRuin {
					ruins++
				}
			}
			assert.GreaterOrEqualf(t, ruins, want,
				"exp=%v seed=%d: 遺跡が最低 %d 個あること（実際 %d）", exp, seed, want, ruins)
		}
	}
}

func TestGenerate_FieldTerrainDominates(t *testing.T) {
	t.Parallel()
	// 道中の「歩く手触り」を主役にするため、中間ノード（合流点・前哨・母港・目標を除く）は
	// フィールド地形（遺跡・平原・山脈）が過半数を占める。街ばかりで探索できないと
	// マクロ/ミクロの比重が崩れる（フィールド率が低すぎる問題への回帰防止）。
	isField := func(n NodeType) bool {
		return n == NodeRuin || n == NodePlain || n == NodeMountain
	}
	for _, exp := range []ExpeditionType{
		ExpeditionDeepVault, ExpeditionTradeCity, ExpeditionPatron, ExpeditionFrontier,
	} {
		var field, middle int
		for seed := range seedCount {
			g := Generate(exp, uint64(seed))
			for _, n := range g.Nodes {
				switch n.Type {
				case NodeHome, NodeGoal, NodeJunction, NodeOutpost:
					// 役割固定ノードは母数から除く
				default:
					middle++
					if isField(n.Type) {
						field++
					}
				}
			}
		}
		assert.Greaterf(t, field*2, middle,
			"exp=%v: 中間ノードの過半数がフィールド地形であること（field=%d middle=%d）", exp, field, middle)
	}
}

// --- テスト用ヘルパー ---

// reachableWithout は blocked ノードを避けて from→to へ到達できるかを BFS で判定する。
// blocked に -1 を渡すと通常の到達判定になる。
func reachableWithout(g *Graph, from, to, blocked NodeID) bool {
	if from == blocked {
		return false
	}
	visited := map[NodeID]bool{from: true}
	queue := []NodeID{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			return true
		}
		for _, e := range g.Outgoing(cur) {
			if e.To == blocked || visited[e.To] {
				continue
			}
			visited[e.To] = true
			queue = append(queue, e.To)
		}
	}
	return false
}

func junctionNode(g *Graph) *Node {
	for i := range g.Nodes {
		if g.Nodes[i].Type == NodeJunction {
			return &g.Nodes[i]
		}
	}
	return nil
}

// hasBranchingNext は n の次層に2ノード以上あるか（＝分岐区間か）を返す。
func hasBranchingNext(g *Graph, n Node) bool {
	next := 0
	for _, m := range g.Nodes {
		if m.Layer == n.Layer+1 {
			next++
		}
	}
	return next >= 2
}
