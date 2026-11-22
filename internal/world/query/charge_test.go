package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

func TestChargeRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		charge *gc.Charge
		want   float64
		wantOK bool
	}{
		{name: "上限に対する割合を返す", charge: &gc.Charge{Max: 600, Current: 150}, want: 0.25, wantOK: true},
		{name: "上限を超えても1に収める", charge: &gc.Charge{Max: 600, Current: 700}, want: 1, wantOK: true},
		{name: "残量を持たなければ ok は偽"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			e := world.ECS.NewEntity()
			if tt.charge != nil {
				world.Components.Charge.Add(e, tt.charge)
			}
			got, ok := ChargeRatio(world, e)
			assert.Equal(t, tt.wantOK, ok)
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}

func TestChargeTriggerOf(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func(world w.World, e ecs.Entity)
		want  ChargeTrigger
	}{
		{name: "光源を持つ物はオンの間に減る", build: func(world w.World, e ecs.Entity) {
			world.Components.Charge.Add(e, &gc.Charge{Max: 600, Current: 600})
			world.Components.LightSource.Add(e, &gc.LightSource{Radius: 6, Enabled: true})
		}, want: ChargeTriggerWhileOn},
		{name: "ほかは使うたびに減る", build: func(world w.World, e ecs.Entity) {
			world.Components.Charge.Add(e, &gc.Charge{Max: 150, Current: 150})
		}, want: ChargeTriggerOnUse},
		{name: "残量を持たない物は減らない", build: func(world w.World, e ecs.Entity) {
			world.Components.LightSource.Add(e, &gc.LightSource{Radius: 6, Enabled: true})
		}, want: ChargeTriggerNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			e := world.ECS.NewEntity()
			tt.build(world, e)
			assert.Equal(t, tt.want, ChargeTriggerOf(world, e))
		})
	}
	t.Run("武器の実体が無ければ減らない", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, ChargeTriggerNone, ChargeTriggerOf(testutil.InitTestWorld(t), gc.InvalidEntity))
	})
}

func TestIsOn(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		build func(world w.World, e ecs.Entity)
		want  bool
	}{
		{name: "床で点いた灯りはオン", build: func(world w.World, e ecs.Entity) {
			world.Components.Charge.Add(e, &gc.Charge{Max: 600, Current: 600})
			world.Components.LightSource.Add(e, &gc.LightSource{Radius: 6, Enabled: true})
			world.Components.GridElement.Add(e, &gc.GridElement{})
		}, want: true},
		{name: "ほかの者のバックパックの灯りはオフ", build: func(world w.World, e ecs.Entity) {
			world.Components.Charge.Add(e, &gc.Charge{Max: 600, Current: 600})
			world.Components.LightSource.Add(e, &gc.LightSource{Radius: 6, Enabled: true})
			world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: world.ECS.NewEntity()})
		}, want: false},
		{name: "光源を持たない物はオフ", build: func(world w.World, e ecs.Entity) {
			world.Components.Charge.Add(e, &gc.Charge{Max: 150, Current: 150})
			world.Components.GridElement.Add(e, &gc.GridElement{})
		}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			e := world.ECS.NewEntity()
			tt.build(world, e)
			assert.Equal(t, tt.want, IsOn(world, gc.InvalidEntity, e))
		})
	}
}
