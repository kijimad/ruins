package lifecycle

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// AddFuel は燃料アイテムを燃えている火へくべる。燃料の熱量を効率で割り引いた燃焼時間を
// Burning.Remaining へ足し、燃料アイテムを消費する。火は燃料を貯め込まず、くべた瞬間に残ターン数へ
// 畳み込む。効率は場所から引く。
// 火が燃えていなければ何もしない。燃料も消費しない。事前条件を呼び出し側に委ねず内部で確かめるので、
// 着火直後でも、メニュー表示中に火が燃え尽きた後でも、同じように安全に呼べる。
func AddFuel(world w.World, fire ecs.Entity, fuel ecs.Entity) {
	if !world.Components.Burning.Has(fire) {
		return
	}
	eff := query.BurnEfficiency(world, fire)
	burning := world.Components.Burning.Get(fire)
	burning.Remaining += query.HeatContent(world, fuel).BurnTurns(eff)
	world.ECS.RemoveEntity(fuel)
}
