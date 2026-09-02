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
	Indent bool // 見出しの下にぶら下がる子行。描画で1段インデントしてグループを読みやすくする
	Color  *color.RGBA
}

// markChildren は見出し以外の行を子行としてインデント指定する。
// 見出しを持つビルダの戻り値を包み、見出しの下の行を1段下げてグループ化する
func markChildren(rows []SpecRow) []SpecRow {
	for i := range rows {
		if !rows[i].Header {
			rows[i].Indent = true
		}
	}
	return rows
}

// specPart は性能表示の1要素。実体と raw spec の2つのデータ源それぞれから行を作る。
// fromSpec が nil の要素は raw spec 表示には出ない。生成後にしか定まらない鮮度や競売などが該当する。
// 要素はこの specParts スライスが単一の真実で、SpecRows と SpecRowsFromSpec が同じ一覧を回す。
// component を足すときはここへ1要素足すだけで両ビューに反映され、片方への入れ忘れが起きない
type specPart struct {
	fromEntity func(world w.World, entity ecs.Entity) []SpecRow
	fromSpec   func(world w.World, spec gc.EntitySpec) []SpecRow
}

var specParts = []specPart{
	{ // 能力値。実体のみ
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Abilities.Has(e) {
				return nil
			}
			return abilityRows(world, world.Components.Abilities.Get(e))
		},
	},
	{ // 近接
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Melee.Has(e) {
				return nil
			}
			return attackerRows(world, world.Components.Melee.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Melee == nil {
				return nil
			}
			return attackerRows(world, s.Melee)
		},
	},
	{ // 射撃
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Fire.Has(e) {
				return nil
			}
			fire := world.Components.Fire.Get(e)
			return append(attackerRows(world, fire), fireAmmoRows(world, fire)...)
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Fire == nil {
				return nil
			}
			return append(attackerRows(world, s.Fire), fireAmmoRows(world, s.Fire)...)
		},
	},
	{ // 材質
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Material.Has(e) {
				return nil
			}
			return []SpecRow{materialRow(world, world.Components.Material.Get(e))}
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Material == nil {
				return nil
			}
			return []SpecRow{materialRow(world, s.Material)}
		},
	},
	{ // 燃料。熱量は保持せず材質と重量から導くので、実体と spec で算出元が違う
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if heat := query.HeatContent(world, e); heat > 0 {
				return []SpecRow{fuelRow(world, heat)}
			}
			return nil
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Material == nil || s.Weight == nil {
				return nil
			}
			if heat := query.HeatOf(s.Material.Kind, s.Weight.Milligram); heat > 0 {
				return []SpecRow{fuelRow(world, heat)}
			}
			return nil
		},
	},
	{ // 防具
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Wearable.Has(e) {
				return nil
			}
			return wearableRows(world, world.Components.Wearable.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Wearable == nil {
				return nil
			}
			return wearableRows(world, s.Wearable)
		},
	},
	{ // 回復
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.ProvidesHealing.Has(e) {
				return nil
			}
			return healingRows(world, world.Components.ProvidesHealing.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.ProvidesHealing == nil {
				return nil
			}
			return healingRows(world, s.ProvidesHealing)
		},
	},
	{ // 栄養
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.ProvidesNutrition.Has(e) {
				return nil
			}
			return nutritionRows(world, world.Components.ProvidesNutrition.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.ProvidesNutrition == nil {
				return nil
			}
			return nutritionRows(world, s.ProvidesNutrition)
		},
	},
	{ // 鮮度。生成時の刻印が要るので実体のみ
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Perishable.Has(e) {
				return nil
			}
			return []SpecRow{freshnessRow(world, e)}
		},
	},
	{ // 本
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Book.Has(e) {
				return nil
			}
			return bookRows(world, world.Components.Book.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Book == nil {
				return nil
			}
			return bookRows(world, s.Book)
		},
	},
	{ // 価値
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Value.Has(e) {
				return nil
			}
			return valueRows(world, world.Components.Value.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Value == nil {
				return nil
			}
			return valueRows(world, s.Value)
		},
	},
	{ // 重量
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Weight.Has(e) {
				return nil
			}
			return weightRows(world, world.Components.Weight.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Weight == nil {
				return nil
			}
			return weightRows(world, s.Weight)
		},
	},
	{ // 治療。価値や重量など多くのアイテムに共通の項目の後に置く
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.Remedy.Has(e) {
				return nil
			}
			return remedyRows(world, world.Components.Remedy.Get(e))
		},
		fromSpec: func(world w.World, s gc.EntitySpec) []SpecRow {
			if s.Remedy == nil {
				return nil
			}
			return remedyRows(world, s.Remedy)
		},
	},
	{ // 出品中。実体のみ
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.AuctionListing.Has(e) {
				return nil
			}
			return auctionListingRows(world, world.Components.AuctionListing.Get(e))
		},
	},
	{ // 落札済み。実体のみ
		fromEntity: func(world w.World, e ecs.Entity) []SpecRow {
			if !world.Components.AuctionSold.Has(e) {
				return nil
			}
			return auctionSoldRows(world, world.Components.AuctionSold.Get(e))
		},
	},
}

// SpecRows はエンティティの性能表示を行の並びとして返す。
// specParts を順に回し、実体が持つ要素だけを含める
func SpecRows(world w.World, entity ecs.Entity) []SpecRow {
	var rows []SpecRow
	for _, p := range specParts {
		if p.fromEntity != nil {
			rows = append(rows, p.fromEntity(world, entity)...)
		}
	}
	return rows
}

// auctionListingRows は出品中の品の番号と現在値を返す。先頭は見出し
func auctionListingRows(world w.World, l *gc.AuctionListing) []SpecRow {
	return markChildren([]SpecRow{
		{Label: query.T(world, "Auction"), Header: true},
		{Label: query.T(world, "Number"), Value: "#" + strconv.Itoa(l.Number)},
		{Label: query.T(world, "Status"), Value: query.T(world, "Bidding")},
		{Label: query.T(world, "Current bid"), Value: l.CurrentBid.String()},
	})
}

// auctionSoldRows は落札済みの品の番号と落札額、出荷期限を返す。先頭は見出し
func auctionSoldRows(world w.World, s *gc.AuctionSold) []SpecRow {
	return markChildren([]SpecRow{
		{Label: query.T(world, "Auction"), Header: true},
		{Label: query.T(world, "Number"), Value: "#" + strconv.Itoa(s.Number)},
		{Label: query.T(world, "Status"), Value: query.T(world, "Won")},
		{Label: query.T(world, "Bid"), Value: s.Bid.String()},
		{Label: query.T(world, "Ship by turn"), Value: strconv.Itoa(s.DueTurn)},
	})
}

// SpecRowsFromSpec は EntitySpec の性能表示を行の並びとして返す。
// エンティティを生成せず raw 定義から詳細を出す商店などで使う。
// specParts を順に回し、fromSpec を持つ要素のうち spec が持つものだけを含める。
// 鮮度など fromSpec が nil の要素は、生成後にしか定まらないのでここには出ない
func SpecRowsFromSpec(world w.World, spec gc.EntitySpec) []SpecRow {
	var rows []SpecRow
	for _, p := range specParts {
		if p.fromSpec != nil {
			rows = append(rows, p.fromSpec(world, spec)...)
		}
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
	return markChildren(rows)
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
	return markChildren(rows)
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
	return markChildren(rows)
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

// remedyRows は治療できる不調と効力の行を返す。先頭は見出し。何を治すか分からないと選べないので示す
func remedyRows(world w.World, remedy *gc.Remedy) []SpecRow {
	rows := make([]SpecRow, 0, 2+len(remedy.Treats))
	rows = append(rows, SpecRow{Label: query.T(world, "Treatment"), Header: true})
	for _, ct := range remedy.Treats {
		rows = append(rows, SpecRow{Label: query.T(world, "Target"), Value: query.T(world, gc.ConditionTypeDisplayName(ct))})
	}
	rows = append(rows, SpecRow{Label: query.T(world, "Potency"), Value: fmt.Sprintf("%d%%", int(remedy.Potency))})
	return markChildren(rows)
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
	return markChildren(rows)
}
