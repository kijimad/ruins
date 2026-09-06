package components

const (
	// DefaultMaxHunger はデフォルトの最大満腹度
	DefaultMaxHunger = 500
	// DefaultInitialHunger はデフォルトの初期満腹度
	DefaultInitialHunger = 400
	// HungerDrainTurns は満腹度が1減るまでの平均ターン数の基準。空腹進行100%のときの値。
	// 大きくするほど空腹の進行が緩やかになる。progressHunger が確率ゲートの分母に使う。
	HungerDrainTurns = 3
)

// HungerLevel は空腹度の段階を表す
type HungerLevel int

const (
	// HungerSatiated は満腹状態
	HungerSatiated HungerLevel = iota
	// HungerNormal は普通状態
	HungerNormal
	// HungerHungry は空腹状態
	HungerHungry
	// HungerStarving は飢餓状態
	HungerStarving
)

// String はHungerLevelの文字列表現を返す
func (h HungerLevel) String() string {
	switch h {
	case HungerSatiated:
		return "Full"
	case HungerNormal:
		return "Normal"
	case HungerHungry:
		return "Hungry"
	case HungerStarving:
		return "Starving"
	default:
		panic("invalid HungerLevel value")
	}
}

// Hunger は空腹度を表す。プレイヤーなどのキャラクターが保持する。
// 0が飢餓状態、値が大きいほど満腹
type Hunger Pool[int]

// GetLevel は現在の空腹度レベルを取得する
func (h *Hunger) GetLevel() HungerLevel {
	if h.Max <= 0 {
		return HungerSatiated
	}

	ratio := float64(h.Current) / float64(h.Max)
	switch {
	case ratio >= 0.95: // 95%以上
		return HungerSatiated
	case ratio >= 0.66: // 66%以上
		return HungerNormal
	case ratio >= 0.33: // 33%以上
		return HungerHungry
	default: // 33%未満
		return HungerStarving
	}
}

// HungerConsciousnessPenalty は空腹段階が意識へ与える低下量を返す。意識は master 乗数なので、この1つの値が
// 命中・行動速度・回復すべてへ波及する。保存せず読み取り時に量から導出する。値は実プレイで調整する
func HungerConsciousnessPenalty(level HungerLevel) int {
	switch level {
	case HungerSatiated, HungerNormal:
		return 0
	case HungerHungry:
		return 10
	case HungerStarving:
		return 20
	}
	return 0
}

// Increase は満腹度を増加させる（食事によって満腹になる）
func (h *Hunger) Increase(amount int) {
	h.Current += amount
	if h.Current > h.Max {
		h.Current = h.Max
	}
	if h.Current < 0 {
		h.Current = 0
	}
}

// Decrease は満腹度を減少させる（行動によって腹が減る）
func (h *Hunger) Decrease(amount int) {
	h.Current -= amount
	if h.Current < 0 {
		h.Current = 0
	}
}

// NewHunger は新しいHungerを作成する
func NewHunger() *Hunger {
	return &Hunger{
		Max:     DefaultMaxHunger,     // 最大満腹度
		Current: DefaultInitialHunger, // 初期状態は満腹
	}
}
