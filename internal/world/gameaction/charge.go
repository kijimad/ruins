package gameaction

import (
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ConsumeCharge は Charge を持つ item の残量を amount 減らす。0になったら物を消し、視界を計算し直させる。
// プレイヤーが持つ物なら、灯りは燃え尽きた、ほかは壊れたとログに出す。減っていく途中はゲージで見せるのでログを出さない
func ConsumeCharge(world w.World, item ecs.Entity, amount int) {
	charge := world.Components.Charge.Get(item)
	charge.Current = max(charge.Current-amount, 0)
	if charge.Current > 0 {
		return
	}
	if heldByPlayer(world, item) {
		name := gamelog.Tag("item", query.GetEntityName(item, world))
		text := query.T(world, "%s broke.", name)
		if world.Components.LightSource.Has(item) {
			text = query.T(world, "%s burned out.", name)
		}
		gamelog.New(query.GetGameLog(world)).Markup(text).Log()
	}
	lifecycle.RemoveItem(world, item)
	query.GetVisionState(world).RequestUpdate()
}

// heldByPlayer は item がプレイヤーの手持ち欄かバックパックにあるかを返す
func heldByPlayer(world w.World, item ecs.Entity) bool {
	switch {
	case world.Components.LocationEquipped.Has(item):
		return world.Components.Player.Has(world.Components.LocationEquipped.Get(item).Owner)
	case world.Components.LocationInBackpack.Has(item):
		return world.Components.Player.Has(world.Components.LocationInBackpack.Get(item).Owner)
	}
	return false
}
