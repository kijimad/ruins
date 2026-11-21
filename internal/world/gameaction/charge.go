package gameaction

import (
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/i18n"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ConsumeCharge は Charge を持つ item の残量を amount 減らす。0になったら物を消し、視界を計算し直させる。
// 持ち主がプレイヤーなら、段階が下がったときと尽きたときにログを出す
func ConsumeCharge(world w.World, item ecs.Entity, amount int) {
	before, _ := query.ChargeStage(world, item)
	charge := world.Components.Charge.Get(item)
	charge.Current = max(charge.Current-amount, 0)
	after, _ := query.ChargeStage(world, item)
	held := world.Components.LocationEquipped.Has(item) && world.Components.Player.Has(world.Components.LocationEquipped.Get(item).Owner)
	if charge.Current > 0 {
		if held && after < before {
			logItem(world, item, "%s is burning down.")
		}
		return
	}
	if held {
		logItem(world, item, "%s burned out.")
	}
	lifecycle.RemoveItem(world, item)
	query.GetVisionState(world).RequestUpdate()
}

// logItem は item の名前を差し込んだ文をゲームログに出す
func logItem(world w.World, item ecs.Entity, msgid i18n.Msgid) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, msgid, gamelog.Tag("item", query.GetEntityName(item, world)))).
		Log()
}
