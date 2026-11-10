package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
)

func TestTileEnvDefs_TempModifier(t *testing.T) {
	t.Parallel()

	defs := TileEnvDefs{
		Water:   map[oapi.WaterExposure]int{oapi.Nearby: -5, oapi.Submerged: -10},
		Foliage: map[oapi.Foliage]int{oapi.Grass: -1, oapi.Forest: -3},
	}
	tests := []struct {
		name string
		env  TileEnvironment
		want int
	}{
		{"全要素がゼロ値", TileEnvironment{}, 0},
		{"囲われは緩和度なので合計に含まない", TileEnvironment{Shelter: ShelterFull}, 0},
		{"水辺で森", TileEnvironment{Water: oapi.Nearby, Foliage: oapi.Forest}, -8},
		{"水中", TileEnvironment{Water: oapi.Submerged}, -10},
		{"全要素の組み合わせ", TileEnvironment{Shelter: ShelterPartial, Water: oapi.Nearby, Foliage: oapi.Grass}, -6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, defs.TempModifier(tt.env))
		})
	}
}
