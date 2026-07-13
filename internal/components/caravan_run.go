package components

import "github.com/kijimaD/ruins/internal/route"

// 寒波前線のモデル定数（面数）。移動は前線と等速なのでリードは変わらず、
// 潜行・野営で道草を食うと前線だけが詰める＝引き際の圧を生む（漏れバケツ回避の設計）。
const (
	// InitialFrontLead はラン開始時のキャラバンの頭金（寒波前線への初期リード）
	InitialFrontLead = 12
	// CampFrontCost は野営1回で寒波前線が詰める面数
	CampFrontCost = 3
	// RuinFrontCost は遺跡潜行1回で寒波前線が詰める面数
	RuinFrontCost = 4
)

// CaravanSupply はキャラバンの供給在庫。食料・燃料は束ねず独立に扱う（緩さ4原則）。
// 積載は1面あたりの食料消費に効く（運搬役が積荷を食う＝物量で頂点が生まれる）。
type CaravanSupply struct {
	// Food は糧食在庫。レグごとに消費する
	Food int
	// Fuel は燃料在庫（炉・凍晶）。レグごとに消費する
	Fuel int
	// Cargo は積載重量。重いほど1面の食料消費が増える
	Cargo route.Weight
}

// CaravanRun はラン単位のマクロ状態を保持するシングルトン。
// ルートグラフ本体はシード（Seed）から決定的に再生成できるため保存せず（json:"-"）、
// ロード時に reestablishSingleton で Generate し直す（Dungeon.ExploredTiles と同要領）。
type CaravanRun struct {
	// Seed はルート網の生成シード。Expedition と対で Graph を決定的に再構築する
	Seed uint64
	// Expedition は選んだ遠征（背骨）。ノード型の重み付けと Graph 再構築に使う
	Expedition route.ExpeditionType
	// Graph は今回生成されたルート網。Seed から再構築できるため保存しない
	Graph *route.Graph `json:"-"`
	// Current はキャラバンの現在ノード
	Current route.NodeID
	// Visited は通過済みノード。来た道は凍って戻れない（一方向）
	Visited []route.NodeID
	// CaravanProgress はキャラバンの累積面数（前進した距離）
	CaravanProgress int
	// FrontProgress は寒波前線の累積面数。CaravanProgress に追いつけばラン失敗
	FrontProgress int
	// Supply は供給在庫（食料・燃料・積載）
	Supply CaravanSupply
}

// NewCaravanRun はシードと遠征からルート網を生成し、母港を起点にランを初期化する。
// 供給の初期値はループを通すための暫定値で、バランスは後段で調整する。
func NewCaravanRun(seed uint64, expedition route.ExpeditionType) *CaravanRun {
	g := route.Generate(expedition, seed)
	return &CaravanRun{
		Seed:       seed,
		Expedition: expedition,
		Graph:      g,
		Current:    g.Home,
		Visited:    []route.NodeID{g.Home},
		Supply:     CaravanSupply{Food: 100, Fuel: 50, Cargo: 0},
	}
}

// FrontLead は寒波前線に対するリード（余裕）を面数で返す。0以下で呑まれ＝ラン失敗。
// 初期頭金（InitialFrontLead）から始まり、移動では前線と等速で変わらず、道草で縮む。
func (r *CaravanRun) FrontLead() int {
	return InitialFrontLead + r.CaravanProgress - r.FrontProgress
}

// Dawdle は前進せず時間を費やす（潜行・野営）ぶん、寒波前線だけを前進させてリードを縮める。
func (r *CaravanRun) Dawdle(faces int) {
	r.FrontProgress += faces
}

// Swallowed は寒波前線に追いつかれた（リード0以下）かを返す。
func (r *CaravanRun) Swallowed() bool {
	return r.FrontLead() <= 0
}

// AdvanceAlong は辺を踏破し、供給消費・前進・現在ノード更新を適用して結果を返す。
// 純計算は route.ResolveLeg に委譲し、ここは状態への適用のみを行う。
// 体温変動・遭遇判定・呑まれ判定は LegResult を使って後段（system/state）が行う。
func (r *CaravanRun) AdvanceAlong(edge route.Edge) route.LegResult {
	res := route.ResolveLeg(edge, r.Supply.Cargo)
	r.Supply.Food -= res.Cost.Food
	r.Supply.Fuel -= res.Cost.Fuel
	r.CaravanProgress += edge.Faces
	r.FrontProgress += res.FrontAdvance
	r.Current = edge.To
	r.Visited = append(r.Visited, edge.To)
	return res
}
