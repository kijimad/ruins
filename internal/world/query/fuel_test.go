package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaterialDefs_HeatOf_材質のkgあたり熱量へ重量を掛ける(t *testing.T) {
	t.Parallel()
	defs := testutil.InitTestWorld(t).Resources.MaterialDefs

	tests := []struct {
		name     string
		material rawschema.Material
		weight   consts.Milligram
		expected consts.Heat
	}{
		{"WOOD 3kg で 600", rawschema.WOOD, 3 * consts.MilligramPerKg, 600},
		{"COAL 1kg で 800", rawschema.COAL, consts.MilligramPerKg, 800},
		{"不燃の金属は0", rawschema.METAL, consts.MilligramPerKg, 0},
		{"軽すぎると切り捨てで0", rawschema.BONE, consts.MilligramPerGram, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, defs.HeatOf(tt.material, tt.weight))
		})
	}
}

func TestIsCombustible(t *testing.T) {
	t.Parallel()

	t.Run("燃焼熱量が正なら可燃", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Material.Add(e, &gc.Material{Kind: rawschema.WOOD})
		world.Components.Weight.Add(e, &gc.Weight{Milligram: consts.MilligramPerKg})

		assert.True(t, query.IsCombustible(world, e))
	})

	t.Run("不燃の材質は不可燃", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Material.Add(e, &gc.Material{Kind: rawschema.METAL})
		world.Components.Weight.Add(e, &gc.Weight{Milligram: consts.MilligramPerKg})

		assert.False(t, query.IsCombustible(world, e))
	})

	t.Run("MaterialとWeightのどちらも無ければ不可燃", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()

		assert.False(t, query.IsCombustible(world, e))
	})
}

func TestHeatContent_燃料の行の物は残量の割合を掛ける(t *testing.T) {
	t.Parallel()
	// 灯油は OIL 1000/kg × 500g で熱 500、残量の上限は 1500。松明は WOOD 200/kg × 500g で熱 100
	tests := []struct {
		name    string
		rawID   string
		current int
		want    consts.Heat
	}{
		{name: "満タンの灯油は材質と重さの熱", rawID: "lamp_oil", current: 1500, want: 500},
		{name: "半分の灯油は半分の熱", rawID: "lamp_oil", current: 750, want: 250},
		{name: "残量1の灯油も燃やせる物のまま", rawID: "lamp_oil", current: 1, want: 1},
		{name: "燃料でない松明は残量で変わらない", rawID: "torch", current: 1, want: 100},
		{name: "残量0の灯油は燃えない", rawID: "lamp_oil", current: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			item, err := lifecycle.SpawnFieldItem(world, tt.rawID, 3, 4, 1)
			require.NoError(t, err)
			if world.Components.Charge.Has(item) {
				world.Components.Charge.Get(item).Current = tt.current
			}

			assert.Equal(t, tt.want, query.HeatContent(world, item))
		})
	}
}
