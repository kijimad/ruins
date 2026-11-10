package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
)

// Perishable は腐敗する食料が持つ。累積した劣化量から鮮度段階を求める。
// 劣化は時間経過で進み、速度は置き場所で変わりうる。速度の適用と実効量の算出は
// query 層が担い、ここは劣化量から段階を求める段階付けだけを持つ。
type Perishable struct {
	RotAccrued     consts.Turn // 累積した劣化量。0 が生成直後
	StageLength    consts.Turn // 1段階の長さ。新鮮 [0,SL) 劣化 [SL,2SL) 腐敗 [2SL,)
	RotUpdatedTurn consts.Turn // RotAccrued を最後に前進させた GameTime.TotalTurns
}

// Stage は累積劣化量 rot から鮮度段階を返す。now でなく実効の rot を渡す。
// 劣化速度は置き場所で変わるため rot の算出は query 層が担い、段階付けだけをここに置く。
// 新鮮 [0, StageLength) → 劣化 [StageLength, 2*StageLength) → 腐敗 [2*StageLength, )。
func (p Perishable) Stage(rot consts.Turn) oapi.FreshnessStageId {
	switch {
	case rot < p.StageLength:
		return oapi.Fresh
	case rot < 2*p.StageLength:
		return oapi.Stale
	default:
		return oapi.Rotten
	}
}

// FreshnessRank は鮮度の段階の並び順を返す。新鮮→劣化→腐敗の固定順で、小さいほど先に並ぶ。
// 段階は Stage が返す既知の3値だけで、未知値はここへ来ない不変条件なので panic させる
func FreshnessRank(s oapi.FreshnessStageId) int {
	switch s {
	case oapi.Fresh:
		return 1
	case oapi.Stale:
		return 2
	case oapi.Rotten:
		return 3
	}
	panic("unknown FreshnessStageId: " + string(s))
}

// FreshnessDef は鮮度の段階1つの定義。raw の freshnessStages の1行から組み立てる
type FreshnessDef struct {
	Name             string // 表示名 msgid
	NutritionPercent int    // 食べたとき得る栄養の満額に対する割合。百分率
}
