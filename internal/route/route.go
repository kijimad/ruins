package route

// NodeID はグラフ内のノードを一意に識別する。
type NodeID int

// NodeType はノードの種別。型ごとに固有のリスク/メリットを持つ。
type NodeType int

// ノード種別の一覧。
const (
	NodeHome     NodeType = iota // 母港（出発点）
	NodeMarket                   // 集落マーケット（交易・補給）
	NodeShop                     // 専門店（改造・特殊装備）
	NodeRuin                     // 遺跡（潜行・サルベージ）
	NodeCamp                     // 途中地点（野営）
	NodeOutpost                  // 前哨（目標地点手前の最終補給・最終売却点）
	NodeJunction                 // 合流点（隊商宿。全ルートが通る括れ）
	NodeGoal                     // 目標地点（遠征のゴール）
)

// EdgeType は辺（レグ）の種別。面数と逆向き利得を型で表す。
type EdgeType int

// 辺種別の一覧。
const (
	EdgeNormal   EdgeType = iota // 本道（標準）
	EdgeShortcut                 // 凍える近道（面少・寒い・襲撃濃い）
	EdgeDetour                   // 暖かい迂回（面多・資源・安全）
	EdgeDanger                   // 危険路
)

// baseFaces は辺種別ごとの基準面数を返す。面数＝距離＝供給消費・寒波接近の量。
func (t EdgeType) baseFaces() int {
	switch t {
	case EdgeShortcut:
		return 2
	case EdgeDetour:
		return 5
	default: // EdgeNormal, EdgeDanger
		return 3
	}
}

// tempDelta は辺種別ごとの体温変化量を返す（負＝冷える）。
func (t EdgeType) tempDelta() int {
	switch t {
	case EdgeShortcut:
		return -3 // 凍える近道
	case EdgeDetour:
		return 0 // 暖かい迂回
	default:
		return -1
	}
}

// encounterChance は辺種別ごとの遭遇（襲撃）確率をパーセントで返す。
func (t EdgeType) encounterChance() int {
	switch t {
	case EdgeDanger:
		return 50
	case EdgeShortcut:
		return 30
	case EdgeDetour:
		return 15
	default:
		return 10
	}
}

// ExpeditionType は遠征（背骨）の種別。生成時のノード型重み付けに使う。
type ExpeditionType int

// 遠征種別の一覧（docs/design/20260712_54.md §13）。
const (
	ExpeditionDeepVault ExpeditionType = iota // 深層ヴォールト（潜行重心）
	ExpeditionTradeCity                       // 交易都市（交易重心）
	ExpeditionPatron                          // 庇護者/派閥拠点（護送）
	ExpeditionFrontier                        // 辺境/未踏（探索）
)

// Node はルートグラフ上の地点。
type Node struct {
	ID    NodeID
	Type  NodeType
	Layer int    // 列（母港=0、目標地点=最終層）。生成と描画の骨格
	Label string // 表示名
	Merit string // 見えているメリット情報（未開示は空。Fog で管理）
	Risk  string // 見えているリスク情報（未開示は空。Fog で管理）
}

// Edge はノード間の有向辺（レグ）。隣接層のみを前進方向に接続する。
type Edge struct {
	From  NodeID
	To    NodeID
	Type  EdgeType
	Faces int // 面数。距離＝供給消費・寒波接近の量
}

// Graph は母港から目標地点への層状 DAG（ルート網）。
type Graph struct {
	Nodes []Node
	Edges []Edge
	Home  NodeID
	Goal  NodeID
}

// Outgoing は現在ノードから前進できる辺の一覧を返す。
func (g *Graph) Outgoing(from NodeID) []Edge {
	var out []Edge
	for _, e := range g.Edges {
		if e.From == from {
			out = append(out, e)
		}
	}
	return out
}

// NodeByID は ID からノードを引く。存在しなければ nil。
func (g *Graph) NodeByID(id NodeID) *Node {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}
