package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// groundBurnEfficiency は地面直の火の燃焼効率をパーセント。100 が等倍で、地面直は低効率で薪をすぐ食う。
// 効率は熱の強さでなく燃焼時間に効く。良い火の見返りは暖かさでなく薪の節約になる
const groundBurnEfficiency = 50

// BurnEfficiency は火が燃えている場所の燃焼効率をパーセントで返す。
// 今は地面直だけなので定数。将来かまどを足すときは Hearth の効率をここで分ける。
// 燃料を燃やし始めるたびにこの値を引くので、後からかまどを足しても同じ機構に載る
func BurnEfficiency(_ w.World, _ ecs.Entity) int {
	return groundBurnEfficiency
}

// EstimateBurnTurns は火があと何ターン燃えるかの見積もりを返す。
// 今燃えている残量に、収納の各燃料を効率で割り引いた燃焼時間を足す。
// fireBurnPerTurn は1なので残量がそのままターン数になる。燃えていなければ0
func EstimateBurnTurns(world w.World, fire ecs.Entity) int {
	if !world.Components.Burning.Has(fire) {
		return 0
	}
	total := world.Components.Burning.Get(fire).Remaining
	eff := BurnEfficiency(world, fire)
	for _, item := range GetStorageItems(world, fire) {
		if world.Components.Fuel.Has(item) {
			total += world.Components.Fuel.Get(item).HeatContent * eff / 100
		}
	}
	return total
}
