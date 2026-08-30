package lifecycle

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// AddFuel は燃料アイテムを火へくべる。燃料の熱量を効率で割り引いた燃焼時間を Burning.Remaining へ足し、
// 燃料アイテムを消費する。火は燃料を貯め込まず、くべた瞬間に残ターン数へ畳み込む。効率は場所から引く。
// 呼び出し側は fire に Burning を付けてから呼ぶ。
func AddFuel(world w.World, fire ecs.Entity, fuel ecs.Entity) {
	eff := query.BurnEfficiency(world, fire)
	burning := world.Components.Burning.Get(fire)
	burning.Remaining += world.Components.Fuel.Get(fuel).HeatContent.BurnTurns(eff)
	world.ECS.RemoveEntity(fuel)
}
