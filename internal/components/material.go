package components

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// Material はアイテムの材質を表す。可燃性と燃焼熱量の算出根拠で、生成時に raw から付く。
// Kind は raw の Material enum の値。金属や石など不燃の材質も保持し、
// 観察メニューで材質を見せて、なぜ燃える/燃えないかを読み取れるようにする。
type Material struct {
	Kind rawschema.Material // 材質の種類。WOOD・METAL・FOOD など。enum で比較のタイプミスを型が弾く
}

// MaterialDef は材質1つの定義。raw の materials の1行から組み立てる
type MaterialDef struct {
	Name        i18n.Msgid           // 表示名
	HeatPerKg   int                  // kgあたり燃焼熱量。0 なら不燃
	Disassembly *MaterialDisassembly // 分解で戻る素材。nil なら分解できない
}

// MaterialDisassembly は材質のアイテムを分解したときに戻る素材
type MaterialDisassembly struct {
	Item      string           // 戻る素材の id
	PerWeight consts.Milligram // 素材1個に要るアイテムの重さ
}

// MaterialDefs は材質ごとの表示名と燃焼熱量の表。読み込み時に raw から1度だけ組み立て、以後は変えない
type MaterialDefs map[rawschema.Material]MaterialDef

// HeatOf は材質と重量から燃焼熱量を導く。不燃の材質は0を返す。
// 整数で閉じるため mg 基準で計算し 1kg=1_000_000mg で割って kg 換算する。切り捨てで軽い物は0になる
func (d MaterialDefs) HeatOf(material rawschema.Material, weight consts.Milligram) consts.Heat {
	return consts.Heat(d[material].HeatPerKg * int(weight) / consts.MilligramPerKg)
}

// DisassemblyOf は材質と重さのアイテム itemID を分解したときに戻る素材の id と個数を返す。
// 個数は重さを PerWeight で割って切り捨てる。分解できない材質、戻し先の素材そのもの、1個に満たない軽い物は ok が false
func (d MaterialDefs) DisassemblyOf(itemID string, material rawschema.Material, weight consts.Milligram) (id string, count int, ok bool) {
	dis := d[material].Disassembly
	if dis == nil || dis.Item == itemID {
		return "", 0, false
	}
	count = int(weight / dis.PerWeight)
	if count < 1 {
		return "", 0, false
	}
	return dis.Item, count, true
}
