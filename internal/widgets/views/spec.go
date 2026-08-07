package views

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/ebitenui/ebitenui/widget"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"

	"github.com/kijimaD/ruins/internal/world/query"
)

// SpecRow は性能表示の1行。Header が真ならカテゴリ見出し行、偽なら ラベル/値 のデータ行。
// 詳細モーダルはこの行の並びを単位としてページ分割する。行の高さは一定なので行数が高さの目安になる。
// Color が非 nil のデータ行はその色で描く。合成の必要素材を条件可否で色分けする用途に使う
type SpecRow struct {
	Label  string
	Value  string
	Header bool
	Color  *color.RGBA
}

// specTableAligns はspec表示テーブルの揃え方向（ラベル左、値右）
var specTableAligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight}

// SpecRows はエンティティの性能表示を行の並びとして返す。
// 種別・攻撃・防具などコンポーネントごとに数行で、存在するものだけを含む
func SpecRows(world w.World, entity ecs.Entity) []SpecRow {
	var rows []SpecRow
	if cat, ok := world.Components.CategoryOf(gc.ItemTypeCategoryKey, entity); ok {
		rows = append(rows, SpecRow{Label: "種別", Value: cat})
	}
	if world.Components.Melee.Has(entity) {
		rows = append(rows, attackerRows(world.Components.Melee.Get(entity))...)
	}
	if world.Components.Fire.Has(entity) {
		fire := world.Components.Fire.Get(entity)
		rows = append(rows, attackerRows(fire)...)
		rows = append(rows, fireAmmoRows(fire)...)
	}
	if world.Components.Wearable.Has(entity) {
		rows = append(rows, wearableRows(world.Components.Wearable.Get(entity))...)
	}
	if world.Components.ProvidesHealing.Has(entity) {
		rows = append(rows, healingRows(world.Components.ProvidesHealing.Get(entity))...)
	}
	if world.Components.ProvidesNutrition.Has(entity) {
		rows = append(rows, nutritionRows(world.Components.ProvidesNutrition.Get(entity))...)
	}
	if world.Components.Book.Has(entity) {
		rows = append(rows, bookRows(world.Components.Book.Get(entity))...)
	}
	if world.Components.Value.Has(entity) {
		rows = append(rows, valueRows(world.Components.Value.Get(entity))...)
	}
	if world.Components.Weight.Has(entity) {
		rows = append(rows, weightRows(world.Components.Weight.Get(entity))...)
	}
	return rows
}

// SpecRowsFromSpec は EntitySpec の性能表示を行の並びとして返す。
// エンティティを生成せず raw 定義から詳細を出す商店などで使う
func SpecRowsFromSpec(world w.World, spec gc.EntitySpec) []SpecRow {
	var rows []SpecRow
	if cat, ok := world.Components.CategoryOfSpec(gc.ItemTypeCategoryKey, &spec); ok {
		rows = append(rows, SpecRow{Label: "種別", Value: cat})
	}
	if spec.Melee != nil {
		rows = append(rows, attackerRows(spec.Melee)...)
	}
	if spec.Fire != nil {
		rows = append(rows, attackerRows(spec.Fire)...)
		rows = append(rows, fireAmmoRows(spec.Fire)...)
	}
	if spec.Wearable != nil {
		rows = append(rows, wearableRows(spec.Wearable)...)
	}
	if spec.ProvidesHealing != nil {
		rows = append(rows, healingRows(spec.ProvidesHealing)...)
	}
	if spec.ProvidesNutrition != nil {
		rows = append(rows, nutritionRows(spec.ProvidesNutrition)...)
	}
	if spec.Book != nil {
		rows = append(rows, bookRows(spec.Book)...)
	}
	if spec.Value != nil {
		rows = append(rows, valueRows(spec.Value)...)
	}
	if spec.Weight != nil {
		rows = append(rows, weightRows(spec.Weight)...)
	}
	return rows
}

// RenderSpecRows は行の並びをコンテナへ1つのテーブルとして描く
func RenderSpecRows(targetContainer *widget.Container, rows []SpecRow, res resources.UIResources) {
	targetContainer.RemoveChildren()
	columnWidths := []int{70, 80}
	table := styled.NewTableContainer(columnWidths, res)
	for _, r := range rows {
		if r.Header {
			styled.NewTableHeaderRow(table, columnWidths, []string{r.Label, ""}, res)
			continue
		}
		if r.Color != nil {
			styled.NewTableRowColored(table, columnWidths, []string{r.Label, r.Value}, specTableAligns, *r.Color, res)
			continue
		}
		styled.NewTableRow(table, columnWidths, []string{r.Label, r.Value}, specTableAligns, nil, res)
	}
	targetContainer.AddChild(table)
}

// UpdateSpec は性能表示コンテナを更新する。全行を描く
func UpdateSpec(world w.World, targetContainer *widget.Container, entity ecs.Entity) {
	RenderSpecRows(targetContainer, SpecRows(world, entity), world.Resources.UIResources)
}

// UpdateSpecFromSpec はEntitySpecから性能表示コンテナを更新する。エンティティを生成せずに性能を表示できる
func UpdateSpecFromSpec(world w.World, targetContainer *widget.Container, spec gc.EntitySpec) {
	RenderSpecRows(targetContainer, SpecRowsFromSpec(world, spec), world.Resources.UIResources)
}

// attackerRows は攻撃パラメータの行を返す。先頭は攻撃種別の見出し
func attackerRows(attack gc.Attacker) []SpecRow {
	rows := []SpecRow{
		{Label: attack.GetAttackCategory().Label, Header: true},
		{Label: consts.DamageLabel, Value: strconv.Itoa(attack.GetDamage())},
		{Label: consts.AccuracyLabel, Value: strconv.Itoa(attack.GetAccuracy())},
		{Label: consts.AttackCountLabel, Value: strconv.Itoa(attack.GetAttackCount())},
		{Label: "コスト", Value: strconv.Itoa(attack.GetCost())},
	}
	if attack.GetElement() != gc.ElementTypeNone {
		rows = append(rows, SpecRow{Label: "属性", Value: attack.GetElement().String()})
	}
	return rows
}

// fireAmmoRows は射程・弾薬の行を返す
func fireAmmoRows(fire *gc.Fire) []SpecRow {
	var rows []SpecRow
	if rangeParams, ok := gc.GetRangeParams(fire.AttackCategory); ok {
		rows = append(rows,
			SpecRow{Label: "適射程", Value: strconv.Itoa(rangeParams.OptimalRange)},
			SpecRow{Label: "射程長", Value: strconv.Itoa(rangeParams.MaxRange)},
		)
	}
	if fire.MagazineSize > 0 {
		rows = append(rows,
			SpecRow{Label: "弾数", Value: fmt.Sprintf("%d/%d", fire.Magazine, fire.MagazineSize)},
			SpecRow{Label: "装填", Value: strconv.Itoa(fire.ReloadEffort)},
		)
	}
	return rows
}

// wearableRows は防具の行を返す。先頭は装備部位の見出し
func wearableRows(wearable *gc.Wearable) []SpecRow {
	rows := []SpecRow{
		{Label: wearable.EquipmentCategory.String(), Header: true},
		{Label: consts.DefenseLabel, Value: fmt.Sprintf("%+d", wearable.Defense)},
	}
	if wearable.InsulationCold != 0 {
		rows = append(rows, SpecRow{Label: "耐寒", Value: fmt.Sprintf("%+d", wearable.InsulationCold)})
	}
	if wearable.InsulationHeat != 0 {
		rows = append(rows, SpecRow{Label: "耐熱", Value: fmt.Sprintf("%+d", wearable.InsulationHeat)})
	}
	rows = append(rows, equipBonusRows(wearable.EquipBonus)...)
	return rows
}

// equipBonusRows は装備ボーナスの行を返す。0 の項目は出さない
func equipBonusRows(equipBonus gc.EquipBonus) []SpecRow {
	var rows []SpecRow
	add := func(label string, v int) {
		if v != 0 {
			rows = append(rows, SpecRow{Label: label, Value: fmt.Sprintf("%+d", v)})
		}
	}
	add(consts.VitalityLabel, equipBonus.Vitality)
	add(consts.StrengthLabel, equipBonus.Strength)
	add(consts.SensationLabel, equipBonus.Sensation)
	add(consts.DexterityLabel, equipBonus.Dexterity)
	add(consts.AgilityLabel, equipBonus.Agility)
	return rows
}

// healingRows は回復量の行を返す
func healingRows(healing *gc.ProvidesHealing) []SpecRow {
	var healValue string
	switch healing.Kind {
	case gc.HealNumeral:
		healValue = strconv.Itoa(int(healing.Amount))
	case gc.HealRatio:
		healValue = fmt.Sprintf("%.0f%%", healing.Amount*100)
	default:
		healValue = "-"
	}
	return []SpecRow{{Label: "体力", Value: healValue}}
}

// nutritionRows は栄養の行を返す
func nutritionRows(nutrition *gc.ProvidesNutrition) []SpecRow {
	return []SpecRow{{Label: "栄養", Value: strconv.Itoa(nutrition.Amount)}}
}

// valueRows は価値の行を返す
func valueRows(value *gc.Value) []SpecRow {
	return []SpecRow{{Label: "価値", Value: query.FormatCurrency(value.Value)}}
}

// weightRows は重量の行を返す
func weightRows(weight *gc.Weight) []SpecRow {
	return []SpecRow{{Label: "重量", Value: weight.String()}}
}

// bookRows は本の行を返す。先頭は見出し
func bookRows(book *gc.Book) []SpecRow {
	rows := []SpecRow{{Label: "本", Header: true}}
	if book.Skill != nil {
		rows = append(rows,
			SpecRow{Label: "スキル", Value: gc.SkillName(book.Skill.TargetSkill)},
			SpecRow{Label: "Lv", Value: fmt.Sprintf("%d %s %d", book.Skill.RequiredLevel, consts.IconArrowRight, book.Skill.MaxLevel)},
		)
	}
	if book.Effort.Current > 0 && book.Effort.Max > 0 {
		pct := book.Effort.Current * 100 / book.Effort.Max
		rows = append(rows, SpecRow{Label: "進捗", Value: fmt.Sprintf("%d%%", pct)})
	}
	return rows
}
