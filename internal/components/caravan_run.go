package components

import "github.com/kijimaD/ruins/internal/route"

// 寒波前線のモデル定数（列数）。ジャンプ（前進）は前線と等速なのでリード不変、
// 潜行・野営で道草を食うと前線だけが詰める＝引き際の圧を生む（漏れバケツ回避の設計）。
const (
	// InitialFrontLead はラン開始時の寒波前線への初期リード（列数）
	InitialFrontLead = 12
	// CampFrontCost は野営1回で寒波前線が余分に詰める列数
	CampFrontCost = 3
	// RuinFrontCost は遺跡潜行1回で寒波前線が余分に詰める列数
	RuinFrontCost = 4
	// StarvationFrontPenalty は食料が尽きた状態で1ジャンプすると寒波前線が余分に詰める列数
	StarvationFrontPenalty = 2

	// jumpFoodCost は1ジャンプあたりの糧食消費
	jumpFoodCost = 4
	// jumpFuelCost は1ジャンプあたりの燃料消費
	jumpFuelCost = 2
)

// CaravanSupply はキャラバンの供給在庫。食料・燃料は束ねず独立に扱う（緩さ4原則）。
type CaravanSupply struct {
	// Food は糧食在庫。ジャンプごとに消費する
	Food int
	// Fuel は燃料在庫（炉・凍晶）。ジャンプごとに消費する
	Fuel int
	// Cargo は積載重量。重いほど1ジャンプの食料消費が増える
	Cargo route.Weight
}

// CaravanRun はラン単位のマクロ状態を保持するシングルトン。
// 停留点マップ本体はシード（Seed）から決定的に再生成できるため保存せず（json:"-"）、
// ロード時に reestablishSingleton で GenerateBeacons し直す。
type CaravanRun struct {
	// Seed は停留点マップの生成シード。Expedition と対で BeaconMap を決定的に再構築する
	Seed uint64
	// Expedition は選んだ遠征（背骨）。イベントの重み付けと再構築に使う
	Expedition route.ExpeditionType
	// Beacons は今回生成された停留点マップ。Seed から再構築できるため保存しない
	Beacons *route.BeaconMap `json:"-"`
	// Current はキャラバンが今いる停留点
	Current route.NodeID
	// CaravanProgress はジャンプで前進した列数
	CaravanProgress int
	// FrontProgress は寒波前線の累積列数。CaravanProgress+初期リードに追いつけば失敗
	FrontProgress int
	// Supply は供給在庫（食料・燃料・積載）
	Supply CaravanSupply
}

// NewCaravanRun はシードと遠征から停留点マップを生成し、母港を起点にランを初期化する。
func NewCaravanRun(seed uint64, expedition route.ExpeditionType) *CaravanRun {
	m := route.GenerateBeacons(expedition, seed)
	return &CaravanRun{
		Seed:       seed,
		Expedition: expedition,
		Beacons:    m,
		Current:    m.Home,
		Supply:     CaravanSupply{Food: 100, Fuel: 50, Cargo: 0},
	}
}

// FrontLead は寒波前線に対するリード（余裕）を列数で返す。0以下で呑まれ＝ラン失敗。
// 初期頭金から始まり、ジャンプ（前進）では前線と等速で変わらず、道草で縮む。
func (r *CaravanRun) FrontLead() int {
	return InitialFrontLead + r.CaravanProgress - r.FrontProgress
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
	r.FrontProgress += cols
}

// JumpTo は次の停留点 to へジャンプし、供給消費・前進を適用する。
// ジャンプ自体はマクロの前進のみ（イベント解決は呼び出し側）。飢餓中は寒波前線が余分に詰める。
func (r *CaravanRun) JumpTo(to route.NodeID) {
	starving := r.IsStarving()

	food := jumpFoodCost + int(r.Supply.Cargo)/10 // 積載が重いほど余分に食う
	r.Supply.Food -= food
	if r.Supply.Food < 0 {
		r.Supply.Food = 0
	}
	r.Supply.Fuel -= jumpFuelCost
	if r.Supply.Fuel < 0 {
		r.Supply.Fuel = 0
	}

	r.CaravanProgress++
	r.FrontProgress++ // 前線は等速で前進（前進はリード不変・道草でリードが縮む）
	if starving {
		r.FrontProgress += StarvationFrontPenalty
	}
	r.Current = to
}
