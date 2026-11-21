package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// LightChargeSystem は照らしている灯りの残量を毎ターン1減らす。退避中のステージの灯りは減らない
type LightChargeSystem struct{}

// String はシステム名を返す
func (sys *LightChargeSystem) String() string {
	return "LightChargeSystem"
}

// Update は照らしている灯りを集めてから、残量を減らす。尽きた灯りを消すので走査の外で減らす
func (sys *LightChargeSystem) Update(world w.World) error {
	// プレイヤーがいなくても床の灯りは減らす
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		player = ecs.Entity{}
	}
	var lights []ecs.Entity
	lightQuery := query.ActiveFilter2[gc.LightSource, gc.Charge](world).Query()
	for lightQuery.Next() {
		if e := lightQuery.Entity(); query.LightLit(world, player, e) {
			lights = append(lights, e)
		}
	}
	for _, light := range lights {
		gameaction.ConsumeCharge(world, light, 1)
	}
	return nil
}
