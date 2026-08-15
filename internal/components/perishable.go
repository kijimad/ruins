package components

import "github.com/kijimaD/ruins/internal/consts"

// Perishable は腐敗する食料が持つ。累積した劣化量から鮮度段階を求める。
// 劣化は時間経過で進み、速度は置き場所で変わりうる。速度の適用と実効量の算出は
// query 層が担い、ここは劣化量から段階を求める段階付けだけを持つ。
type Perishable struct {
	Rot       consts.Turn // 累積した劣化量。0 が生成直後
	ShelfLife consts.Turn // 1段階の長さ。新鮮 [0,SL) 劣化 [SL,2SL) 腐敗 [2SL,)
	LastCheck consts.Turn // Rot を最後に前進させた GameTime.TotalTurns
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

// Stage は累積劣化量 rot から鮮度段階を返す。now でなく実効の rot を渡す。
// 劣化速度は置き場所で変わるため rot の算出は query 層が担い、段階付けだけをここに置く。
// 新鮮 [0, ShelfLife) → 劣化 [ShelfLife, 2*ShelfLife) → 腐敗 [2*ShelfLife, )。
func (p Perishable) Stage(rot consts.Turn) FreshnessStage {
	switch {
	case rot < p.ShelfLife:
		return FreshnessFresh
	case rot < 2*p.ShelfLife:
		return FreshnessStale
	default:
		return FreshnessRotten
	}
}

// MergeRot は個数で加重平均した劣化量に自身を更新する。合流の survivor 側で呼ぶ。
// 呼ぶ前に双方を同じ基準時刻まで前進させ、Rot を実効値にしておく前提。
func (p *Perishable) MergeRot(selfCount int, other Perishable, otherCount int) {
	total := selfCount + otherCount
	if total <= 0 {
		return
	}
	p.Rot = (p.Rot*consts.Turn(selfCount) + other.Rot*consts.Turn(otherCount)) / consts.Turn(total)
}
