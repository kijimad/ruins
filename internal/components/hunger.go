package components

const (
	// DefaultMaxHunger はデフォルトの最大満腹度
	DefaultMaxHunger = 500
	// DefaultInitialHunger はデフォルトの初期満腹度
	DefaultInitialHunger = 400
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
		return "満腹"
	case HungerNormal:
		return "普通"
	case HungerHungry:
		return "空腹"
	case HungerStarving:
		return "飢餓"
	default:
		return "不明"
	}
}

// Hunger はプレイヤー専用の空腹度システム
type Hunger struct {
	Pool // 0が飢餓状態、値が大きいほど満腹
}

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

// GetStatusPenalty は空腹度によるペナルティを取得する
func (h *Hunger) GetStatusPenalty() int {
	level := h.GetLevel()
	switch level {
	case HungerStarving:
		return -20 // 飢餓状態では大きなペナルティ
	case HungerHungry:
		return -10 // 空腹状態では中程度のペナルティ
	default:
		return 0 // 満腹・普通では影響なし
	}
}

// NewHunger は新しいHungerを作成する
func NewHunger() *Hunger {
	return &Hunger{
		Pool: Pool{
			Max:     DefaultMaxHunger,     // 最大満腹度
			Current: DefaultInitialHunger, // 初期状態は満腹
		},
	}
}
