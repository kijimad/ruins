package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// WhileOnChargeSystem はオンの間に減る物のうち、いまオンの物の残量を毎ターン1減らす。退避中のステージの物は減らない
type WhileOnChargeSystem struct{}

// String はシステム名を返す
func (sys *WhileOnChargeSystem) String() string {
	return "WhileOnChargeSystem"
}

// Update はオンの物を集めてから、残量を減らす。尽きた物を消すので走査の外で減らす
func (sys *WhileOnChargeSystem) Update(world w.World) error {
	// プレイヤーがいなくても床の物は減らす
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		player = gc.InvalidEntity
	}
	var on []ecs.Entity
	q := query.ActiveFilter1[gc.Charge](world).Query()
	for q.Next() {
		e := q.Entity()
		if query.ChargeTriggerOf(world, e) == query.ChargeTriggerWhileOn && query.IsOn(world, player, e) {
			on = append(on, e)
		}
	}
	for _, item := range on {
		gameaction.ConsumeCharge(world, item, 1)
	}
	return nil
}
