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
		"wood":  {HeatPerKg: 15000},
		"metal": {HeatPerKg: 0},
	}

	tests := []struct {
		name     string
		material rawschema.MaterialId
		weight   consts.Milligram
		want     consts.Heat
	}{
		{"1kgの可燃材質はkgあたり熱量そのまま", "wood", consts.MilligramPerKg, 15000},
		{"0.5kgは半分の熱量", "wood", consts.MilligramPerKg / 2, 7500},
		{"軽すぎる重量は切り捨てで0", "wood", 1, 0},
		{"不燃材質は0", "metal", consts.MilligramPerKg, 0},
		{"未登録の材質は0", rawschema.MaterialId("UNKNOWN"), consts.MilligramPerKg, 0},
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
		"metal": {Disassembly: &MaterialDisassembly{Item: "scrap_iron", PerWeight: consts.MilligramPerKg}},
		"cloth": {},
	}

	type want struct {
		id    string
		count int
		ok    bool
	}
	tests := []struct {
		name     string
		itemID   string
		material rawschema.MaterialId
		weight   consts.Milligram
		want     want
	}{
		{"重さを素材1個の重さで割った数だけ戻る", "sword", "metal", 3 * consts.MilligramPerKg, want{"scrap_iron", 3, true}},
		{"端数は切り捨てる", "knife", "metal", consts.MilligramPerKg * 3 / 2, want{"scrap_iron", 1, true}},
		{"1個に満たない軽い物は分解できない", "nail", "metal", consts.MilligramPerKg / 2, want{"", 0, false}},
		{"戻し先の素材そのものは分解できない", "scrap_iron", "metal", consts.MilligramPerKg, want{"", 0, false}},
		{"戻し先の無い材質は分解できない", "shirt", "cloth", consts.MilligramPerKg, want{"", 0, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, count, ok := defs.DisassemblyOf(tt.itemID, tt.material, tt.weight)
			assert.Equal(t, tt.want, want{id, count, ok})
		})
	}
}
