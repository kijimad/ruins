package entityspec

import (
	"fmt"
	"image/color"
	"strconv"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"

	"github.com/kijimaD/ruins/internal/world/query"
)

// SpecRow は性能表示の1行。Header が真ならカテゴリ見出し行、偽なら ラベル/値 のデータ行。
// 詳細モーダルはこの行の並びを単位としてページ分割する。行の高さは一定なので行数が高さの目安になる。
// Color が非 nil のデータ行はその色で描く。クラフトの必要素材を条件可否で色分けする用途に使う
type SpecRow struct {
	Label  string
	Value  string
	Header bool
	Color  *color.RGBA
}

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
	if world.Components.Material.Has(entity) {
		rows = append(rows, materialRow(world, world.Components.Material.Get(entity)))
	}
	if heat := query.HeatContent(world, entity); heat > 0 {
		rows = append(rows, fuelRow(world, heat))
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
	if world.Components.AuctionListing.Has(entity) {
		rows = append(rows, auctionListingRows(world, world.Components.AuctionListing.Get(entity))...)
	}
	if world.Components.AuctionSold.Has(entity) {
		rows = append(rows, auctionSoldRows(world, world.Components.AuctionSold.Get(entity))...)
	}
	return rows
}

// auctionListingRows は出品中の品の番号と現在値を返す。先頭は見出し
func auctionListingRows(world w.World, l *gc.AuctionListing) []SpecRow {
	return []SpecRow{
		{Label: query.T(world, "Auction"), Header: true},
		{Label: query.T(world, "Number"), Value: "#" + strconv.Itoa(l.Number)},
		{Label: query.T(world, "Status"), Value: query.T(world, "Bidding")},
		{Label: query.T(world, "Current bid"), Value: l.CurrentBid.String()},
	}
}

// auctionSoldRows は落札済みの品の番号と落札額、出荷期限を返す。先頭は見出し
func auctionSoldRows(world w.World, s *gc.AuctionSold) []SpecRow {
	return []SpecRow{
		{Label: query.T(world, "Auction"), Header: true},
		{Label: query.T(world, "Number"), Value: "#" + strconv.Itoa(s.Number)},
		{Label: query.T(world, "Status"), Value: query.T(world, "Won")},
		{Label: query.T(world, "Bid"), Value: s.Bid.String()},
		{Label: query.T(world, "Ship by turn"), Value: strconv.Itoa(s.DueTurn)},
	}
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
	if spec.Material != nil {
		rows = append(rows, materialRow(world, spec.Material))
	}
	if spec.Material != nil && spec.Weight != nil {
		if heat := query.HeatOf(spec.Material.Kind, spec.Weight.Milligram); heat > 0 {
			rows = append(rows, fuelRow(world, heat))
		}
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
	// 鮮度は生成時の刻印 RotUpdatedTurn が要る。spec 段階では未刻印なので出さない
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

// materialRow は材質の1行を返す。可燃性と燃焼熱量の根拠で、不燃の材質も見せて燃える/燃えないを読み取れる
func materialRow(world w.World, material *gc.Material) SpecRow {
	return SpecRow{Label: query.T(world, "Material"), Value: query.T(world, materialDisplayName(material.Kind))}
}

// materialDisplayName は raw の Material enum 値を表示名へ写す。表示名は query.T で各言語へ訳す。
// default に enum 値をそのまま返し、未知の材質でも空欄にしない
func materialDisplayName(kind oapi.Material) string {
	switch kind {
	case oapi.WOOD:
		return "Wood"
	case oapi.PAPER:
		return "Paper"
	case oapi.CLOTH:
		return "Cloth"
	case oapi.LEATHER:
		return "Leather"
	case oapi.PLANT:
		return "Plant fiber"
	case oapi.FOOD:
		return "Food"
	case oapi.BONE:
		return "Bone"
	case oapi.OIL:
		return "Oil"
	case oapi.COAL:
		return "Coal"
	case oapi.PLASTIC:
		return "Plastic"
	case oapi.METAL:
		return "Metal"
	case oapi.STONE:
		return "Stone"
	case oapi.GLASS:
		return "Glass"
	case oapi.CRYSTAL:
		return "Crystal"
	case oapi.CERAMIC:
		return "Ceramic"
	case oapi.LIQUID:
		return "Liquid"
	default:
		return string(kind)
	}
}

// fuelRow は燃料の熱量の1行を返す。燃やしたとき火へ移す熱量で、燃焼時間の目安になる。
// 値の無い見出し行は置かず、材質の行と並べて読ませる。熱量は保持せず材質と重量から導く
func fuelRow(world w.World, heat consts.Heat) SpecRow {
	// 熱量は炎アイコンで見せる。満burn時の燃焼ターン数に等しく、地面直の火では効率で減る
	return SpecRow{Label: query.T(world, "Fuel"), Value: heat.String()}
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
	return SpecRow{Label: query.T(world, "Freshness"), Value: query.T(world, stage.Label())}
}

// valueRows は価値の行を返す
func valueRows(world w.World, value *gc.Value) []SpecRow {
	return []SpecRow{{Label: query.T(world, "Value"), Value: consts.Currency(value.Value).String()}}
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
