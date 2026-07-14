package components

import "github.com/kijimaD/ruins/internal/route"

// グリッド広域マップの寸法（CDDA 風オーバーマップ。まず画面に収まる小さめ）。
const (
	// GridW は広域マップの横セル数（母港=左端・目標=右端）
	GridW = 12
	// GridH は広域マップの縦セル数
	GridH = 9
)

// 寒波前線のモデル定数（列数）。移動は前線と等速（右へ進めばリード不変）、
// 潜行・野営で道草を食うと前線だけが詰める＝引き際の圧を生む（漏れバケツ回避の設計）。
const (
	// InitialFrontLead はラン開始時の寒波前線への初期リード（列数）
	InitialFrontLead = 12
	// CampFrontCost は野営1回で寒波前線が余分に詰める列数
	CampFrontCost = 3
	// RuinFrontCost は遺跡潜行1回で寒波前線が余分に詰める列数
	RuinFrontCost = 4
	// StarvationFrontPenalty は食料が尽きた状態で1移動すると寒波前線が余分に詰める列数
	StarvationFrontPenalty = 2

	// moveFoodCost は1移動あたりの糧食消費
	moveFoodCost = 3
	// moveFuelCost は1移動あたりの燃料消費
	moveFuelCost = 1
)

// CaravanSupply はキャラバンの供給在庫。食料・燃料は束ねず独立に扱う（緩さ4原則）。
// 積載は1移動あたりの食料消費に効く（運搬役が積荷を食う＝物量で頂点が生まれる）。
type CaravanSupply struct {
	// Food は糧食在庫。移動ごとに消費する
	Food int
	// Fuel は燃料在庫（炉・凍晶）。移動ごとに消費する
	Fuel int
	// Cargo は積載重量。重いほど1移動の食料消費が増える
	Cargo route.Weight
}

// CaravanRun はラン単位のマクロ状態を保持するシングルトン。
// グリッド本体はシード（Seed）から決定的に再生成できるため保存せず（json:"-"）、
// ロード時に reestablishSingleton で GenerateGrid し直す（Dungeon.ExploredTiles と同要領）。
type CaravanRun struct {
	// Seed はグリッド生成シード。Expedition と対で Grid を決定的に再構築する
	Seed uint64
	// Expedition は選んだ遠征（背骨）。地形の重み付けと Grid 再構築に使う
	Expedition route.ExpeditionType
	// Grid は今回生成された広域マップ。Seed から再構築できるため保存しない
	Grid *route.Grid `json:"-"`
	// Pos はキャラバンの現在セル
	Pos route.Coord
	// FrontCol は寒波前線が到達した列。これ以下の列は凍結（戻れない）。Pos.X に追いつけば失敗
	FrontCol int
	// Supply は供給在庫（食料・燃料・積載）
	Supply CaravanSupply
}

// NewCaravanRun はシードと遠征から広域マップを生成し、母港を起点にランを初期化する。
// 供給の初期値はループを通すための暫定値で、バランスは後段で調整する。
func NewCaravanRun(seed uint64, expedition route.ExpeditionType) *CaravanRun {
	grid := route.GenerateGrid(expedition, seed, GridW, GridH)
	return &CaravanRun{
		Seed:       seed,
		Expedition: expedition,
		Grid:       grid,
		Pos:        grid.Home,
		FrontCol:   grid.Home.X - InitialFrontLead,
		Supply:     CaravanSupply{Food: 100, Fuel: 50, Cargo: 0},
	}
}

// FrontLead は寒波前線に対するリード（余裕）を列数で返す。0以下で呑まれ＝ラン失敗。
func (r *CaravanRun) FrontLead() int {
	return r.Pos.X - r.FrontCol
}

// Swallowed は寒波前線に追いつかれた（リード0以下）かを返す。
func (r *CaravanRun) Swallowed() bool {
	return r.FrontLead() <= 0
}

// IsStarving は食料が尽きている（飢餓）かを返す。飢餓中の移動は寒波前線を余分に詰める。
func (r *CaravanRun) IsStarving() bool {
	return r.Supply.Food <= 0
}

// Dawdle は前進せず時間を費やす（潜行・野営）ぶん、寒波前線だけを前進させてリードを縮める。
func (r *CaravanRun) Dawdle(cols int) {
	r.FrontCol += cols
}

// CanMoveTo は隣接セル c へ移動できるか返す（グリッド内・前線より前・上下左右で隣接）。
func (r *CaravanRun) CanMoveTo(c route.Coord) bool {
	if !r.Grid.In(c) {
		return false
	}
	if c.X <= r.FrontCol {
		return false // 凍結した後方セルへは戻れない（一方向）
	}
	dx, dy := c.X-r.Pos.X, c.Y-r.Pos.Y
	return abs(dx)+abs(dy) == 1
}

// MoveTo は隣接セル c へ移動し、供給消費・前線前進を適用する。CanMoveTo で検証済みを前提。
// 移動自体はマクロ横断のみ（ミクロには入らない）。飢餓中は寒波前線が余分に詰める。
func (r *CaravanRun) MoveTo(c route.Coord) {
	starving := r.IsStarving()

	food := moveFoodCost + int(r.Supply.Cargo)/10 // 積載が重いほど余分に食う
	r.Supply.Food -= food
	if r.Supply.Food < 0 {
		r.Supply.Food = 0
	}
	r.Supply.Fuel -= moveFuelCost
	if r.Supply.Fuel < 0 {
		r.Supply.Fuel = 0
	}

	r.Pos = c
	r.FrontCol++ // 前線は等速で前進（右へ進めばリード不変・寄り道でリードが縮む）
	if starving {
		r.FrontCol += StarvationFrontPenalty
	}
}

// abs は整数の絶対値を返す。
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
