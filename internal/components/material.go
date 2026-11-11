package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/oapi"
)

// Material はアイテムの材質を表す。可燃性と燃焼熱量の算出根拠で、生成時に raw から付く。
// Kind は raw の Material enum の値。金属や石など不燃の材質も保持し、
// 観察メニューで材質を見せて、なぜ燃える/燃えないかを読み取れるようにする。
type Material struct {
	Kind oapi.Material // 材質の種類。WOOD・METAL・FOOD など。enum で比較のタイプミスを型が弾く
}

// MaterialDef は材質1つの定義。raw の materials の1行から組み立てる
type MaterialDef struct {
	Name      i18n.Msgid // 表示名
	HeatPerKg int        // kgあたり燃焼熱量。0 なら不燃
}

// MaterialDefs は材質ごとの表示名と燃焼熱量の表。読み込み時に raw から1度だけ組み立て、以後は変えない
type MaterialDefs map[oapi.Material]MaterialDef

// HeatOf は材質と重量から燃焼熱量を導く。不燃の材質は0を返す。
// 整数で閉じるため mg 基準で計算し 1kg=1_000_000mg で割って kg 換算する。切り捨てで軽い物は0になる
func (d MaterialDefs) HeatOf(material oapi.Material, weight consts.Milligram) consts.Heat {
	return consts.Heat(d[material].HeatPerKg * int(weight) / consts.MilligramPerKg)
}
