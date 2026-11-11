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
	StageLength    consts.Turn // 1段階の長さ。鮮度の段階の境目は、raw の段階の行がこの何倍かで持つ
	RotUpdatedTurn consts.Turn // RotAccrued を最後に前進させた GameTime.TotalTurns
}

// StagesElapsed は累積劣化量 rot が経過させた段階数を返す。now でなく実効の rot を渡す。
// 劣化速度は置き場所で変わるため rot の算出は query 層が担う
func (p Perishable) StagesElapsed(rot consts.Turn) int {
	return int(rot / p.StageLength)
}

// FreshnessDef は鮮度の段階1つの定義。raw の freshnessStages の1行から組み立てる
type FreshnessDef struct {
	Level                oapi.FreshnessStageId
	From                 int    // 経過した段階数がこれ以上の段階のうち From が最大のものになる
	Name                 string // 表示名 msgid
	NutritionPercent     int    // 食べたとき得る栄養の満額に対する割合。百分率
	EatLog               string // 食べたときのログの msgid。%s は食べた者と食べ物
	Marker               bool   // メニューの行に鮮度を添えるか
	IllnessChancePercent int    // 食べたとき食中毒を起こす確率。百分率
}
