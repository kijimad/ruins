package entityspec

import (
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// UseHints は実体の性質・使い道を「用途」見出しでまとめた行の並びを返す。
// 各判定は item_action_state の動詞タブの Accept と同条件にし、使える動詞と出る性質を一致させる。
// 死んだ実体や性質を持たない実体には nil を返す。
func UseHints(world w.World, e ecs.Entity) []SpecRow {
	if !world.ECS.Alive(e) {
		return nil
	}
	c := world.Components
	consumable := c.Consumable.Has(e)
	nutrition := c.ProvidesNutrition.Has(e)
	healing := c.ProvidesHealing.Has(e)

	var uses []string
	if consumable && (nutrition || healing) { // acceptConsumeFood
		uses = append(uses, query.T(world, "Edible"))
	}
	if c.Book.Has(e) {
		uses = append(uses, query.T(world, "Readable"))
	}
	if consumable && !nutrition && !healing { // acceptUseTool
		uses = append(uses, query.T(world, "Usable"))
	}
	// 武器も防具も装備スロットに付けるので同じ「装備できる」にまとめる
	if c.Wearable.Has(e) || c.Melee.Has(e) || c.Fire.Has(e) {
		uses = append(uses, query.T(world, "Wearable"))
	}
	// 分解工具は専用コンポーネントを持たず raw 定義の有無で判定する
	if _, ok := raw.FindDisassemblyTool(world.Resources.RawMaster, query.GetEntityID(e, world)); ok {
		uses = append(uses, query.T(world, "Can disassemble items"))
	}
	if len(uses) == 0 {
		return nil
	}

	rows := make([]SpecRow, 0, len(uses)+1)
	rows = append(rows, SpecRow{Label: query.T(world, "Uses"), Header: true})
	for _, u := range uses {
		rows = append(rows, SpecRow{Label: u, Indent: 1})
	}
	return rows
}
