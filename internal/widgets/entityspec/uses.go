package entityspec

import (
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// UseHints は実体の性質・使い道を表す短い文言を表示順に返す。
// 判定は item_action_state の動詞タブと同じコンポーネント条件を使い、
// 「その動詞が使える」と「その性質が出る」を一致させる。装備や武器は
// 専用画面で扱うため動詞タブには無いが、用途としてここに並べる。
// 死んだ実体には nil を返し、ゼロ実体への Get で落ちるのを防ぐ。
func UseHints(world w.World, e ecs.Entity) []string {
	if !world.ECS.Alive(e) {
		return nil
	}
	c := world.Components
	consumable := c.Consumable.Has(e)
	nutrition := c.ProvidesNutrition.Has(e)
	healing := c.ProvidesHealing.Has(e)

	var hints []string
	// 栄養か回復を持つ消費物は食べられる。acceptConsumeFood と同条件
	if consumable && (nutrition || healing) {
		hints = append(hints, "Edible")
	}
	if c.Book.Has(e) {
		hints = append(hints, "Readable")
	}
	// 栄養も回復も持たない消費物は道具として使う。acceptUseTool と同条件
	if consumable && !nutrition && !healing {
		hints = append(hints, "Usable")
	}
	if c.Wearable.Has(e) {
		hints = append(hints, "Wearable")
	}
	if c.Melee.Has(e) || c.Fire.Has(e) {
		hints = append(hints, "Weapon")
	}
	// 分解工具は専用コンポーネントを持たず raw 定義の有無で判定する
	if _, ok := raw.FindDisassemblyTool(world.Resources.RawMaster, query.GetEntityID(e, world)); ok {
		hints = append(hints, "Can disassemble items")
	}
	return hints
}
