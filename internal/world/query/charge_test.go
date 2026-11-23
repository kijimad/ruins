package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/rawschema"
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

func TestFuelKindMatches(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	noRaw := world.ECS.NewEntity()
	_, ok := AcceptedFuelOf(world, noRaw)
	assert.False(t, ok, "raw の行を持たない物は受ける種別を持たない")
	assert.False(t, FuelKindMatches(world, rawschema.FuelKindId("oil"), noRaw), "raw の行を持たない物は燃料でない")

	oil := world.ECS.NewEntity()
	world.Components.RawID.Add(oil, &gc.RawID{ID: "lamp_oil"})
	assert.True(t, FuelKindMatches(world, rawschema.FuelKindId("oil"), oil))
	assert.False(t, FuelKindMatches(world, rawschema.FuelKindId("battery"), oil))
}

func TestChargeKindOf(t *testing.T) {
	t.Parallel()
	tests := []struct {
		rawID  string
		want   rawschema.FuelKindId
		wantOK bool
	}{
		{rawID: "battery", want: "battery", wantOK: true},
		{rawID: "flashlight", want: "battery", wantOK: true},
		{rawID: "lamp_oil", want: "oil", wantOK: true},
		{rawID: "torch"},
		{rawID: "iron_sword"},
	}
	for _, tt := range tests {
		t.Run(tt.rawID, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			item := world.ECS.NewEntity()
			world.Components.RawID.Add(item, &gc.RawID{ID: tt.rawID})

			got, ok := ChargeKindOf(world, item)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTransferableCharge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		from int
		to   int
		want int
	}{
		{name: "受け側の空きのほうが少ない", from: 800, to: 700, want: 300},
		{name: "出し側の残りのほうが少ない", from: 200, to: 100, want: 200},
		{name: "受け側が満タン", from: 800, to: 1000, want: 0},
		{name: "出し側が空", from: 0, to: 100, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			from := world.ECS.NewEntity()
			world.Components.Charge.Add(from, &gc.Charge{Current: tt.from, Max: 1000})
			to := world.ECS.NewEntity()
			world.Components.Charge.Add(to, &gc.Charge{Current: tt.to, Max: 1000})

			assert.Equal(t, tt.want, TransferableCharge(world, from, to))
		})
	}
}

func TestChargeMarker(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	battery := world.ECS.NewEntity()
	world.Components.Charge.Add(battery, &gc.Charge{Current: 750, Max: 1000})
	empty := world.ECS.NewEntity()
	world.Components.Charge.Add(empty, &gc.Charge{Current: 0, Max: 1000})
	stone := world.ECS.NewEntity()

	assert.Equal(t, []string{"750", "0", ""}, []string{ChargeMarker(world, battery), ChargeMarker(world, empty), ChargeMarker(world, stone)})
}
