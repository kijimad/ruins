package components

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/i18n"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// Ability は変動するパラメータ値
type Ability struct {
	Base     int // 固有の値
	Modifier int // 装備や健康状態で変動する値
	Total    int // 計算した現在値。算出される値のメモ
}

// Abilities はエンティティが持つ能力値。各種計算式で使う
type Abilities struct {
	Vitality  Ability // 体力。丈夫さ、持久力、しぶとさ。HPやSPに影響する
	Strength  Ability // 筋力。主に近接攻撃のダメージに影響する
	Sensation Ability // 感覚。主に射撃攻撃のダメージに影響する
	Dexterity Ability // 器用。攻撃時の命中率に影響する
	Agility   Ability // 敏捷。回避率、行動の速さに影響する
	Defense   Ability // 防御。被弾ダメージを軽減させる
}

// bodyWeightBaseKg は体重の基準値。体格を持たない状態での下限にあたる
const bodyWeightBaseKg = 40

// BodyWeight は能力値から算出する体の重量を返す。基準体重に体格分、Vitality と Strength、を足す。
// member や NPC の体重に使い、価格と同じく体格を反映させる
func (a Abilities) BodyWeight() consts.Milligram {
	physiqueKg := a.Vitality.Base + a.Strength.Base
	return consts.Milligram((bodyWeightBaseKg + physiqueKg) * consts.MilligramPerKg)
}

// abilityName は能力値IDの表示名マップ
var abilityName = map[rawschema.AbilityId]i18n.Msgid{
	rawschema.Str: "STR",
	rawschema.Sen: "SEN",
	rawschema.Dex: "DEX",
	rawschema.Agi: "AGI",
	rawschema.Vit: "VIT",
	rawschema.Def: "DEF",
}

// AbilityName は能力値IDの表示名を返す。未定義ならpanicする
func AbilityName(id rawschema.AbilityId) i18n.Msgid {
	name, ok := abilityName[id]
	if !ok {
		panic(fmt.Sprintf("undefined ability ID: %q", id))
	}
	return name
}

// ValueOf は指定された能力値IDに対応するTotal値を返す
func (a *Abilities) ValueOf(id rawschema.AbilityId) int {
	switch id {
	case rawschema.Str:
		return a.Strength.Total
	case rawschema.Sen:
		return a.Sensation.Total
	case rawschema.Dex:
		return a.Dexterity.Total
	case rawschema.Agi:
		return a.Agility.Total
	case rawschema.Vit:
		return a.Vitality.Total
	case rawschema.Def:
		return a.Defense.Total
	default:
		return 0
	}
}
