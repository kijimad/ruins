package query_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

func TestMaterialDefs_HeatOf_材質のkgあたり熱量へ重量を掛ける(t *testing.T) {
	t.Parallel()
	defs := testutil.InitTestWorld(t).Resources.MaterialDefs

	tests := []struct {
		name     string
		material rawschema.MaterialId
		weight   consts.Milligram
		expected consts.Heat
	}{
		{"wood 3kg で 600", "wood", 3 * consts.MilligramPerKg, 600},
		{"COAL 1kg で 800", "coal", consts.MilligramPerKg, 800},
		{"不燃の金属は0", "metal", consts.MilligramPerKg, 0},
		{"軽すぎると切り捨てで0", "bone", consts.MilligramPerGram, 0},
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
		world.Components.Material.Add(e, &gc.Material{Kind: "wood"})
		world.Components.Weight.Add(e, &gc.Weight{Milligram: consts.MilligramPerKg})

		assert.True(t, query.IsCombustible(world, e))
	})

	t.Run("不燃の材質は不可燃", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Material.Add(e, &gc.Material{Kind: "metal"})
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
