package entityspec

import (
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// UseHints は実体の性質・使い道を「用途」見出しでまとめた行の並びを返す。
// 見出し1行に続けて性質を1段下げて並べ、性能行と同じ体裁でグルーピングして描く。
// 文言は query.T で現在言語へ訳す。各判定は item_action_state の動詞タブと同じ
// コンポーネント条件を使い、「その動詞が使える」と「その性質が出る」を一致させる。
// 装備や武器は専用画面で扱うため動詞タブには無いが、用途としてここに並べる。
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
	// 栄養か回復を持つ消費物は食べられる。acceptConsumeFood と同条件
	if consumable && (nutrition || healing) {
		uses = append(uses, query.T(world, "Edible"))
	}
	if c.Book.Has(e) {
		uses = append(uses, query.T(world, "Readable"))
	}
	// 栄養も回復も持たない消費物は道具として使う。acceptUseTool と同条件
	if consumable && !nutrition && !healing {
		uses = append(uses, query.T(world, "Usable"))
	}
	// 防具も武器も装備スロットに付けるので同じ「装備できる」でまとめる
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
