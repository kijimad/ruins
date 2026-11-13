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

func TestMaterialDefs_DisassemblyOf(t *testing.T) {
	t.Parallel()

	defs := MaterialDefs{
		rawschema.METAL: {Disassembly: &MaterialDisassembly{Item: "scrap_iron", PerWeight: consts.MilligramPerKg}},
		rawschema.CLOTH: {},
	}

	type want struct {
		id    string
		count int
		ok    bool
	}
	tests := []struct {
		name     string
		itemID   string
		material rawschema.Material
		weight   consts.Milligram
		want     want
	}{
		{"重さを素材1個の重さで割った数だけ戻る", "sword", rawschema.METAL, 3 * consts.MilligramPerKg, want{"scrap_iron", 3, true}},
		{"端数は切り捨てる", "knife", rawschema.METAL, consts.MilligramPerKg * 3 / 2, want{"scrap_iron", 1, true}},
		{"1個に満たない軽い物は分解できない", "nail", rawschema.METAL, consts.MilligramPerKg / 2, want{"", 0, false}},
		{"戻し先の素材そのものは分解できない", "scrap_iron", rawschema.METAL, consts.MilligramPerKg, want{"", 0, false}},
		{"戻し先の無い材質は分解できない", "shirt", rawschema.CLOTH, consts.MilligramPerKg, want{"", 0, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, count, ok := defs.DisassemblyOf(tt.itemID, tt.material, tt.weight)
			assert.Equal(t, tt.want, want{id, count, ok})
		})
	}
}
