package query_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
)

func TestHeatOf_材質のkgあたり熱量へ重量を掛ける(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		material oapi.Material
		weight   consts.Milligram
		expected consts.Heat
	}{
		{"WOOD 3kg で 600", oapi.WOOD, 3 * consts.MilligramPerKg, 600},
		{"COAL 1kg で 800", oapi.COAL, consts.MilligramPerKg, 800},
		{"不燃の金属は0", oapi.METAL, consts.MilligramPerKg, 0},
		{"軽すぎると切り捨てで0", oapi.BONE, consts.MilligramPerGram, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, query.HeatOf(tt.material, tt.weight))
		})
	}
}
