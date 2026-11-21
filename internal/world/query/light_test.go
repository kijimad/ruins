package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestLightLit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		enabled bool
		place   func(light, player, other ecs.Entity, world w.World)
		want    bool
	}{
		{name: "点いた床の灯りは照らす", enabled: true, place: func(l, _, _ ecs.Entity, world w.World) {
			world.Components.GridElement.Add(l, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}})
		}, want: true},
		{name: "消えた床の灯りは照らさない", enabled: false, place: func(l, _, _ ecs.Entity, world w.World) {
			world.Components.GridElement.Add(l, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}})
		}, want: false},
		{name: "プレイヤーの手持ち欄の灯りは照らす", enabled: true, place: func(l, p, _ ecs.Entity, world w.World) {
			world.Components.LocationEquipped.Add(l, &gc.LocationEquipped{Owner: p, EquipmentSlot: gc.SlotHand3})
		}, want: true},
		{name: "ほかの者の手持ち欄の灯りは照らさない", enabled: true, place: func(l, _, o ecs.Entity, world w.World) {
			world.Components.LocationEquipped.Add(l, &gc.LocationEquipped{Owner: o, EquipmentSlot: gc.SlotHand1})
		}, want: false},
		{name: "プレイヤーの防具の欄の灯りは照らさない", enabled: true, place: func(l, p, _ ecs.Entity, world w.World) {
			world.Components.LocationEquipped.Add(l, &gc.LocationEquipped{Owner: p, EquipmentSlot: gc.SlotHead})
		}, want: false},
		{name: "背負った灯りは照らさない", enabled: true, place: func(l, p, _ ecs.Entity, world w.World) {
			world.Components.LocationInBackpack.Add(l, &gc.LocationInBackpack{Owner: p})
		}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, other, light := world.ECS.NewEntity(), world.ECS.NewEntity(), world.ECS.NewEntity()
			world.Components.LightSource.Add(light, &gc.LightSource{Radius: 4, Enabled: tt.enabled})
			tt.place(light, player, other, world)
			assert.Equal(t, tt.want, query.LightLit(world, player, light))
		})
	}
}
