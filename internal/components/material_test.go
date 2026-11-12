package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestMaterialDefs_HeatOf(t *testing.T) {
	t.Parallel()

	defs := MaterialDefs{
		rawschema.WOOD:  {HeatPerKg: 15000},
		rawschema.METAL: {HeatPerKg: 0},
	}

	tests := []struct {
		name     string
		material rawschema.Material
		weight   consts.Milligram
		want     consts.Heat
	}{
		{"1kgの可燃材質はkgあたり熱量そのまま", rawschema.WOOD, consts.MilligramPerKg, 15000},
		{"0.5kgは半分の熱量", rawschema.WOOD, consts.MilligramPerKg / 2, 7500},
		{"軽すぎる重量は切り捨てで0", rawschema.WOOD, 1, 0},
		{"不燃材質は0", rawschema.METAL, consts.MilligramPerKg, 0},
		{"未登録の材質は0", rawschema.Material("UNKNOWN"), consts.MilligramPerKg, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, defs.HeatOf(tt.material, tt.weight))
		})
	}
}
