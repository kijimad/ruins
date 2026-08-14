package components

import "github.com/kijimaD/ruins/internal/consts"

// Perishable は腐敗する食料が持つ。生成時刻と保存期間から鮮度を遅延評価する。
// 気温非依存の固定速度で、保存する値は consts.Turn だけなので serde 安全。
// SpawnedAtTurn は生成時に必ず刻印される前提
type Perishable struct {
	SpawnedAtTurn consts.Turn // 生成時の GameTime.TotalTurns
	ShelfLife     consts.Turn // 新鮮でいられるターン数。これを基準に段階が決まる
}

// FreshnessStage は鮮度の段階
type FreshnessStage string

const (
	// FreshnessFresh は新鮮。栄養は満額
	FreshnessFresh FreshnessStage = "fresh"
	// FreshnessStale は劣化。栄養が減るが無害
	FreshnessStale FreshnessStage = "stale"
	// FreshnessRotten は腐敗。栄養は得られず、食べると不調になる
	FreshnessRotten FreshnessStage = "rotten"
)

// Stage は経過ターンから鮮度段階を返す。now は GameTime.TotalTurns。
// 新鮮 [0, ShelfLife) → 劣化 [ShelfLife, 2*ShelfLife) → 腐敗 [2*ShelfLife, )。
// 値レシーバで十分。int 2つしか持たず変更もしない
func (p Perishable) Stage(now consts.Turn) FreshnessStage {
	age := now - p.SpawnedAtTurn
	switch {
	case age < p.ShelfLife:
		return FreshnessFresh
	case age < 2*p.ShelfLife:
		return FreshnessStale
	default:
		return FreshnessRotten
	}
}
