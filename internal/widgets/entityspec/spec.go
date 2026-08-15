package entityspec

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
	if world.Components.Abilities.Has(entity) {
		rows = append(rows, abilityRows(world, world.Components.Abilities.Get(entity))...)
	}
	if world.Components.Melee.Has(entity) {
		rows = append(rows, attackerRows(world, world.Components.Melee.Get(entity))...)
	}
	if world.Components.Fire.Has(entity) {
		fire := world.Components.Fire.Get(entity)
		rows = append(rows, attackerRows(world, fire)...)
		rows = append(rows, fireAmmoRows(world, fire)...)
	}
	if world.Components.Wearable.Has(entity) {
		rows = append(rows, wearableRows(world, world.Components.Wearable.Get(entity))...)
	}
	if world.Components.ProvidesHealing.Has(entity) {
		rows = append(rows, healingRows(world, world.Components.ProvidesHealing.Get(entity))...)
	}
	if world.Components.ProvidesNutrition.Has(entity) {
		rows = append(rows, nutritionRows(world, world.Components.ProvidesNutrition.Get(entity))...)
	}
	if world.Components.Perishable.Has(entity) {
		rows = append(rows, freshnessRow(world, entity))
	}
	if world.Components.Book.Has(entity) {
		rows = append(rows, bookRows(world, world.Components.Book.Get(entity))...)
	}
	if world.Components.Value.Has(entity) {
		rows = append(rows, valueRows(world, world.Components.Value.Get(entity))...)
	}
	if world.Components.Weight.Has(entity) {
		rows = append(rows, weightRows(world, world.Components.Weight.Get(entity))...)
	}
	return rows
}

// SpecRowsFromSpec は EntitySpec の性能表示を行の並びとして返す。
// エンティティを生成せず raw 定義から詳細を出す商店などで使う
func SpecRowsFromSpec(world w.World, spec gc.EntitySpec) []SpecRow {
	var rows []SpecRow
	if spec.Melee != nil {
		rows = append(rows, attackerRows(world, spec.Melee)...)
	}
	if spec.Fire != nil {
		rows = append(rows, attackerRows(world, spec.Fire)...)
		rows = append(rows, fireAmmoRows(world, spec.Fire)...)
	}
	if spec.Wearable != nil {
		rows = append(rows, wearableRows(world, spec.Wearable)...)
	}
	if spec.ProvidesHealing != nil {
		rows = append(rows, healingRows(world, spec.ProvidesHealing)...)
	}
	if spec.ProvidesNutrition != nil {
		rows = append(rows, nutritionRows(world, spec.ProvidesNutrition)...)
	}
	// 鮮度は生成時の刻印 RotAsOfTurn が要る。spec 段階では未刻印なので出さない
	if spec.Book != nil {
		rows = append(rows, bookRows(world, spec.Book)...)
	}
	if spec.Value != nil {
		rows = append(rows, valueRows(world, spec.Value)...)
	}
	if spec.Weight != nil {
		rows = append(rows, weightRows(world, spec.Weight)...)
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
			styled.NewTableRowColored(table, columnWidths, styled.TextCells(r.Label, r.Value), specTableAligns, *r.Color, res)
			continue
		}
		styled.NewTableRow(table, columnWidths, styled.TextCells(r.Label, r.Value), specTableAligns, nil, res)
	}
	targetContainer.AddChild(table)
}

// attackerRows は攻撃パラメータの行を返す。先頭は攻撃種別の見出し
func attackerRows(world w.World, attack gc.Attacker) []SpecRow {
	rows := []SpecRow{
		{Label: query.T(world, attack.GetAttackCategory().Label), Header: true},
		{Label: query.T(world, "Attack power"), Value: strconv.Itoa(attack.GetDamage())},
		{Label: query.T(world, "Accuracy"), Value: strconv.Itoa(attack.GetAccuracy())},
		{Label: query.T(world, "Hits"), Value: strconv.Itoa(attack.GetAttackCount())},
		{Label: query.T(world, "Attack cost"), Value: strconv.Itoa(attack.GetCost())},
	}
	if attack.GetElement() != gc.ElementTypeNone {
		rows = append(rows, SpecRow{Label: query.T(world, "Element"), Value: query.T(world, attack.GetElement().String())})
	}
	return rows
}

// fireAmmoRows は射程・弾薬の行を返す
func fireAmmoRows(world w.World, fire *gc.Fire) []SpecRow {
	var rows []SpecRow
	if rangeParams, ok := gc.GetRangeParams(fire.AttackCategory); ok {
		rows = append(rows,
			SpecRow{Label: query.T(world, "Optimal range"), Value: strconv.Itoa(rangeParams.OptimalRange)},
			SpecRow{Label: query.T(world, "Max range"), Value: strconv.Itoa(rangeParams.MaxRange)},
		)
	}
	if fire.MagazineSize > 0 {
		rows = append(rows,
			SpecRow{Label: query.T(world, "Magazine"), Value: fmt.Sprintf("%d/%d", fire.Magazine, fire.MagazineSize)},
			SpecRow{Label: query.T(world, "Reload"), Value: strconv.Itoa(fire.ReloadEffort)},
		)
	}
	return rows
}

// wearableRows は防具の行を返す。先頭は装備部位の見出し
// abilityRows は能力値をキャラ画面と同じラベルで縦に並べる。能力を持つ実体の詳細で使う
func abilityRows(world w.World, a *gc.Abilities) []SpecRow {
	return []SpecRow{
		{Label: query.T(world, "Vitality"), Value: strconv.Itoa(a.Vitality.Base)},
		{Label: query.T(world, "Strength"), Value: strconv.Itoa(a.Strength.Base)},
		{Label: query.T(world, "Sensation"), Value: strconv.Itoa(a.Sensation.Base)},
		{Label: query.T(world, "Dexterity"), Value: strconv.Itoa(a.Dexterity.Base)},
		{Label: query.T(world, "Agility"), Value: strconv.Itoa(a.Agility.Base)},
		{Label: query.T(world, "Defense"), Value: strconv.Itoa(a.Defense.Base)},
	}
}

func wearableRows(world w.World, wearable *gc.Wearable) []SpecRow {
	rows := []SpecRow{
		{Label: query.T(world, wearable.EquipmentCategory.String()), Header: true},
		{Label: query.T(world, "Defense"), Value: fmt.Sprintf("%+d", wearable.Defense)},
	}
	if wearable.InsulationCold != 0 {
		rows = append(rows, SpecRow{Label: query.T(world, "Cold resist"), Value: fmt.Sprintf("%+d", wearable.InsulationCold)})
	}
	if wearable.InsulationHeat != 0 {
		rows = append(rows, SpecRow{Label: query.T(world, "Heat resist"), Value: fmt.Sprintf("%+d", wearable.InsulationHeat)})
	}
	rows = append(rows, equipBonusRows(world, wearable.EquipBonus)...)
	return rows
}

// equipBonusRows は装備ボーナスの行を返す。0 の項目は出さない
func equipBonusRows(world w.World, equipBonus gc.EquipBonus) []SpecRow {
	var rows []SpecRow
	add := func(label string, v int) {
		if v != 0 {
			rows = append(rows, SpecRow{Label: label, Value: fmt.Sprintf("%+d", v)})
		}
	}
	add(query.T(world, "Vitality"), equipBonus.Vitality)
	add(query.T(world, "Strength"), equipBonus.Strength)
	add(query.T(world, "Sensation"), equipBonus.Sensation)
	add(query.T(world, "Dexterity"), equipBonus.Dexterity)
	add(query.T(world, "Agility"), equipBonus.Agility)
	return rows
}

// healingRows は回復量の行を返す
func healingRows(world w.World, healing *gc.ProvidesHealing) []SpecRow {
	var healValue string
	switch healing.Kind {
	case gc.HealNumeral:
		healValue = strconv.Itoa(int(healing.Amount))
	case gc.HealRatio:
		healValue = fmt.Sprintf("%.0f%%", healing.Amount*100)
	default:
		healValue = "-"
	}
	return []SpecRow{{Label: query.T(world, "Vitality"), Value: healValue}}
}

// nutritionRows は栄養の行を返す
func nutritionRows(world w.World, nutrition *gc.ProvidesNutrition) []SpecRow {
	return []SpecRow{{Label: query.T(world, "Nutrition"), Value: strconv.Itoa(nutrition.Amount)}}
}

// freshnessRow は鮮度の1行を返す。鮮度の算出は query.FreshnessStageOf に委ねる
func freshnessRow(world w.World, entity ecs.Entity) SpecRow {
	stage, _ := query.FreshnessStageOf(world, entity)
	return SpecRow{Label: query.T(world, "Freshness"), Value: query.T(world, query.FreshnessLabel(stage))}
}

// valueRows は価値の行を返す
func valueRows(world w.World, value *gc.Value) []SpecRow {
	return []SpecRow{{Label: query.T(world, "Value"), Value: query.FormatCurrency(value.Value)}}
}

// weightRows は重量の行を返す
func weightRows(world w.World, weight *gc.Weight) []SpecRow {
	return []SpecRow{{Label: query.T(world, "Weight"), Value: weight.String()}}
}

// bookRows は本の行を返す。先頭は見出し
func bookRows(world w.World, book *gc.Book) []SpecRow {
	rows := []SpecRow{{Label: query.T(world, "Book"), Header: true}}
	if book.Skill != nil {
		rows = append(rows,
			SpecRow{Label: query.T(world, "Skill"), Value: query.T(world, gc.SkillName(book.Skill.TargetSkill))},
			SpecRow{Label: "Lv", Value: fmt.Sprintf("%d %s %d", book.Skill.RequiredLevel, consts.IconArrowRight, book.Skill.MaxLevel)},
		)
	}
	if book.Effort.Current > 0 && book.Effort.Max > 0 {
		pct := book.Effort.Current * 100 / book.Effort.Max
		rows = append(rows, SpecRow{Label: query.T(world, "Progress"), Value: fmt.Sprintf("%d%%", pct)})
	}
	return rows
}
