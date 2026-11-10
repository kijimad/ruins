package components

import "github.com/kijimaD/ruins/internal/oapi"

const (
	// DefaultMaxHunger はデフォルトの最大満腹度
	DefaultMaxHunger = 500
	// DefaultInitialHunger はデフォルトの初期満腹度
	DefaultInitialHunger = 400
	// HungerDrainTurns は満腹度が1減るまでの平均ターン数の基準。空腹進行100%のときの値。
	// 大きくするほど空腹の進行が緩やかになる。progressHunger が確率ゲートの分母に使う。
	HungerDrainTurns = 3
)

// Hunger は空腹度を表す。プレイヤーなどのキャラクターが保持する。
// 0が飢餓状態、値が大きいほど満腹
type Hunger Pool[int]

// GetLevel は現在の空腹度レベルを取得する。段階の境目は defs が持つ
func (h *Hunger) GetLevel(defs NeedDefs) oapi.HungerStageId {
	if h.Max <= 0 {
		return oapi.HungerStageIdSatiated
	}
	return defs.hungerStage(float64(h.Current) / float64(h.Max)).Level
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
