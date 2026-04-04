package views

import (
	"fmt"
	"strconv"

	"github.com/ebitenui/ebitenui/widget"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// UpdateSpec は性能表示コンテナを更新する
func UpdateSpec(world w.World, targetContainer *widget.Container, entity ecs.Entity) {
	targetContainer.RemoveChildren()

	// 各コンポーネントの情報を追加
	if entity.HasComponent(world.Components.Attack) {
		attack := world.Components.Attack.Get(entity).(*gc.Attack)
		addAttackInfo(targetContainer, attack, world)
	}
	if entity.HasComponent(world.Components.Wearable) {
		wearable := world.Components.Wearable.Get(entity).(*gc.Wearable)
		addWearableInfo(targetContainer, wearable, world)
	}
	if entity.HasComponent(world.Components.Weapon) {
		weapon := world.Components.Weapon.Get(entity).(*gc.Weapon)
		addWeaponInfo(targetContainer, weapon, world)
	}
	if entity.HasComponent(world.Components.ProvidesHealing) {
		healing := world.Components.ProvidesHealing.Get(entity).(*gc.ProvidesHealing)
		addHealingInfo(targetContainer, healing, world)
	}
	if entity.HasComponent(world.Components.ProvidesNutrition) {
		nutrition := world.Components.ProvidesNutrition.Get(entity).(*gc.ProvidesNutrition)
		addNutritionInfo(targetContainer, nutrition, world)
	}
	if entity.HasComponent(world.Components.Book) {
		book := world.Components.Book.Get(entity).(*gc.Book)
		addBookInfo(targetContainer, book, world)
	}
	if entity.HasComponent(world.Components.Value) {
		v := world.Components.Value.Get(entity).(*gc.Value)
		addValueInfo(targetContainer, v, world)
	}
	if entity.HasComponent(world.Components.Weight) {
		w := world.Components.Weight.Get(entity).(*gc.Weight)
		addWeightInfo(targetContainer, w, world)
	}
}

// UpdateSpecFromSpec はEntitySpecから性能表示コンテナを更新する
// エンティティを生成せずに性能を表示できる
func UpdateSpecFromSpec(world w.World, targetContainer *widget.Container, spec gc.EntitySpec) {
	targetContainer.RemoveChildren()

	if spec.Attack != nil {
		addAttackInfo(targetContainer, spec.Attack, world)
	}
	if spec.Wearable != nil {
		addWearableInfo(targetContainer, spec.Wearable, world)
	}
	if spec.Weapon != nil {
		addWeaponInfo(targetContainer, spec.Weapon, world)
	}
	if spec.ProvidesHealing != nil {
		addHealingInfo(targetContainer, spec.ProvidesHealing, world)
	}
	if spec.ProvidesNutrition != nil {
		addNutritionInfo(targetContainer, spec.ProvidesNutrition, world)
	}
	if spec.Book != nil {
		addBookInfo(targetContainer, spec.Book, world)
	}
	if spec.Value != nil {
		addValueInfo(targetContainer, spec.Value, world)
	}
	if spec.Weight != nil {
		addWeightInfo(targetContainer, spec.Weight, world)
	}
}

// specTableAligns はspec表示テーブルの揃え方向（ラベル左、値右）
var specTableAligns = []styled.TextAlign{styled.AlignLeft, styled.AlignRight}

// addAttackInfo はAttackコンポーネントの情報を追加する
func addAttackInfo(targetContainer *widget.Container, attack *gc.Attack, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(table, columnWidths, []string{attack.AttackCategory.Label, ""}, res)
	styled.NewTableRow(table, columnWidths, []string{consts.DamageLabel, strconv.Itoa(attack.Damage)}, specTableAligns, nil, res)
	styled.NewTableRow(table, columnWidths, []string{consts.AccuracyLabel, strconv.Itoa(attack.Accuracy)}, specTableAligns, nil, res)
	styled.NewTableRow(table, columnWidths, []string{consts.AttackCountLabel, strconv.Itoa(attack.AttackCount)}, specTableAligns, nil, res)

	if attack.Element != gc.ElementTypeNone {
		styled.NewTableRow(table, columnWidths, []string{"属性", attack.Element.String()}, specTableAligns, nil, res)
	}

	targetContainer.AddChild(table)
}

// addWearableInfo はWearableコンポーネントの情報を追加する
func addWearableInfo(targetContainer *widget.Container, wearable *gc.Wearable, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableHeaderRow(table, columnWidths, []string{wearable.EquipmentCategory.String(), ""}, res)
	styled.NewTableRow(table, columnWidths, []string{consts.DefenseLabel, fmt.Sprintf("%+d", wearable.Defense)}, specTableAligns, nil, res)

	if wearable.InsulationCold != 0 {
		styled.NewTableRow(table, columnWidths, []string{"耐寒", fmt.Sprintf("%+d", wearable.InsulationCold)}, specTableAligns, nil, res)
	}
	if wearable.InsulationHeat != 0 {
		styled.NewTableRow(table, columnWidths, []string{"耐熱", fmt.Sprintf("%+d", wearable.InsulationHeat)}, specTableAligns, nil, res)
	}

	addEquipBonusToTable(table, columnWidths, wearable.EquipBonus, res)
	targetContainer.AddChild(table)
}

// addWeaponInfo はWeaponコンポーネントの情報を追加する
func addWeaponInfo(targetContainer *widget.Container, weapon *gc.Weapon, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableRow(table, columnWidths, []string{"コスト", strconv.Itoa(weapon.Cost)}, specTableAligns, nil, res)
	targetContainer.AddChild(table)
}

// addValueInfo はValueコンポーネントの情報を追加する
func addValueInfo(targetContainer *widget.Container, value *gc.Value, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableRow(table, columnWidths, []string{"価値", worldhelper.FormatCurrency(value.Value)}, specTableAligns, nil, res)
	targetContainer.AddChild(table)
}

// addHealingInfo はProvidesHealingコンポーネントの情報を追加する
func addHealingInfo(targetContainer *widget.Container, healing *gc.ProvidesHealing, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	var healValue string
	switch amt := healing.Amount.(type) {
	case gc.NumeralAmount:
		healValue = strconv.Itoa(amt.Numeral)
	case gc.RatioAmount:
		healValue = fmt.Sprintf("%.0f%%", amt.Ratio*100)
	default:
		healValue = "-"
	}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableRow(table, columnWidths, []string{"体力", healValue}, specTableAligns, nil, res)
	targetContainer.AddChild(table)
}

// addNutritionInfo はProvidesNutritionコンポーネントの情報を追加する
func addNutritionInfo(targetContainer *widget.Container, nutrition *gc.ProvidesNutrition, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableRow(table, columnWidths, []string{"栄養", strconv.Itoa(nutrition.Amount)}, specTableAligns, nil, res)
	targetContainer.AddChild(table)
}

// addWeightInfo はWeightコンポーネントの情報を追加する
func addWeightInfo(targetContainer *widget.Container, weight *gc.Weight, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	table := styled.NewTableContainer(columnWidths, res)
	styled.NewTableRow(table, columnWidths, []string{"重量", fmt.Sprintf("%.2f%s", weight.Kg, consts.IconKg)}, specTableAligns, nil, res)
	targetContainer.AddChild(table)
}

// addBookInfo はBookコンポーネントの情報を追加する
func addBookInfo(targetContainer *widget.Container, book *gc.Book, world w.World) {
	res := world.Resources.UIResources
	columnWidths := []int{70, 80}

	table := styled.NewTableContainer(columnWidths, res)

	styled.NewTableHeaderRow(table, columnWidths, []string{"本", ""}, res)

	if book.Skill != nil {
		skillName := gc.SkillName(book.Skill.TargetSkill)
		styled.NewTableRow(table, columnWidths, []string{"スキル", skillName}, specTableAligns, nil, res)

		lvRange := fmt.Sprintf("%d %s %d", book.Skill.RequiredLevel, consts.IconArrowRight, book.Skill.MaxLevel)
		styled.NewTableRow(table, columnWidths, []string{"Lv", lvRange}, specTableAligns, nil, res)
	}

	if book.Effort.Current > 0 && book.Effort.Max > 0 {
		pct := book.Effort.Current * 100 / book.Effort.Max
		styled.NewTableRow(table, columnWidths, []string{"進捗", fmt.Sprintf("%d%%", pct)}, specTableAligns, nil, res)
	}

	targetContainer.AddChild(table)
}

// addEquipBonusToTable は装備ボーナスをテーブルに追加する
func addEquipBonusToTable(table *widget.Container, columnWidths []int, equipBonus gc.EquipBonus, res *resources.UIResources) {
	if equipBonus.Vitality != 0 {
		styled.NewTableRow(table, columnWidths, []string{consts.VitalityLabel, fmt.Sprintf("%+d", equipBonus.Vitality)}, specTableAligns, nil, res)
	}
	if equipBonus.Strength != 0 {
		styled.NewTableRow(table, columnWidths, []string{consts.StrengthLabel, fmt.Sprintf("%+d", equipBonus.Strength)}, specTableAligns, nil, res)
	}
	if equipBonus.Sensation != 0 {
		styled.NewTableRow(table, columnWidths, []string{consts.SensationLabel, fmt.Sprintf("%+d", equipBonus.Sensation)}, specTableAligns, nil, res)
	}
	if equipBonus.Dexterity != 0 {
		styled.NewTableRow(table, columnWidths, []string{consts.DexterityLabel, fmt.Sprintf("%+d", equipBonus.Dexterity)}, specTableAligns, nil, res)
	}
	if equipBonus.Agility != 0 {
		styled.NewTableRow(table, columnWidths, []string{consts.AgilityLabel, fmt.Sprintf("%+d", equipBonus.Agility)}, specTableAligns, nil, res)
	}
}
