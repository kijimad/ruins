package components

import "github.com/kijimaD/ruins/internal/consts"

// Perishable は腐敗する食料が持つ。累積した劣化量から鮮度段階を求める。
// 劣化は時間経過で進み、速度は置き場所で変わりうる。速度の適用と実効量の算出は
// query 層が担い、ここは劣化量から段階を求める段階付けだけを持つ。
type Perishable struct {
	RotAccrued     consts.Turn // 累積した劣化量。0 が生成直後
	StageLength    consts.Turn // 1段階の長さ。新鮮 [0,SL) 劣化 [SL,2SL) 腐敗 [2SL,)
	RotUpdatedTurn consts.Turn // RotAccrued を最後に前進させた GameTime.TotalTurns
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
// 新鮮 [0, StageLength) → 劣化 [StageLength, 2*StageLength) → 腐敗 [2*StageLength, )。
func (p Perishable) Stage(rot consts.Turn) FreshnessStage {
	switch {
	case rot < p.StageLength:
		return FreshnessFresh
	case rot < 2*p.StageLength:
		return FreshnessStale
	default:
		return FreshnessRotten
	}
}

// Rank は段階の並び順を返す。新鮮→劣化→腐敗の固定順で、小さいほど先に並ぶ。
// 段階の定義・順序・表示は鮮度という1つの軸の属性なので、このファイルに同居させる。
// 段階は Stage が返す既知の3値だけで、未知値はここへ来ない不変条件なので panic させる
func (s FreshnessStage) Rank() int {
	switch s {
	case FreshnessFresh:
		return 1
	case FreshnessStale:
		return 2
	case FreshnessRotten:
		return 3
	}
	panic("unknown FreshnessStage: " + string(s))
}

// Label は段階の表示に使う英語 msgid を返す。訳出は表示側が query.T で行う
func (s FreshnessStage) Label() string {
	switch s {
	case FreshnessFresh:
		return "Fresh"
	case FreshnessStale:
		return "Stale"
	case FreshnessRotten:
		return "Rotten"
	}
	panic("unknown FreshnessStage: " + string(s))
}
