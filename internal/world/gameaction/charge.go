package gameaction

import (
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ConsumeCharge は Charge を持つ item の残量を amount 減らす。0になったら物を消し、視界を計算し直させる。
// 持ち主がプレイヤーなら尽きたことをログに出す
func ConsumeCharge(world w.World, item ecs.Entity, amount int) {
	charge := world.Components.Charge.Get(item)
	charge.Current = max(charge.Current-amount, 0)
	if charge.Current > 0 {
		return
	}
	if world.Components.LocationEquipped.Has(item) && world.Components.Player.Has(world.Components.LocationEquipped.Get(item).Owner) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s burned out.", gamelog.Tag("item", query.GetEntityName(item, world)))).
			Log()
	}
	lifecycle.RemoveItem(world, item)
	query.GetVisionState(world).RequestUpdate()
}
