package components

import "github.com/kijimaD/ruins/internal/consts"

// Perishable は腐敗する食料が持つ。生成時刻と保存期間から鮮度を遅延評価する。
type Perishable struct {
	SpawnedAtTurn consts.Turn // 生成時の GameTime.TotalTurns
	ShelfLife     consts.Turn // 新鮮でいられるターン数。これを基準に段階が決まる
}

// FreshnessStage は鮮度の段階
type FreshnessStage string

const (
	// FreshnessFresh は新鮮
	FreshnessFresh FreshnessStage = "fresh"
	// FreshnessStale は劣化
	FreshnessStale FreshnessStage = "stale"
	// FreshnessRotten は腐敗
	FreshnessRotten FreshnessStage = "rotten"
)

// Stage は経過ターンから鮮度段階を返す。now は GameTime.TotalTurns。
// 新鮮 [0, ShelfLife) → 劣化 [ShelfLife, 2*ShelfLife) → 腐敗 [2*ShelfLife, )。
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
