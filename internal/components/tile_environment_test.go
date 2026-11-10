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
		Foliage: map[oapi.Foliage]int{oapi.FoliageGrass: -1, oapi.FoliageForest: -3},
	}
	tests := []struct {
		name string
		env  TileEnvironment
		want int
	}{
		{"全要素がゼロ値", TileEnvironment{}, 0},
		{"囲われは緩和度なので合計に含まない", TileEnvironment{Shelter: oapi.Indoor}, 0},
		{"水辺で森", TileEnvironment{Water: oapi.Nearby, Foliage: oapi.FoliageForest}, -8},
		{"水中", TileEnvironment{Water: oapi.Submerged}, -10},
		{"全要素の組み合わせ", TileEnvironment{Shelter: oapi.Partial, Water: oapi.Nearby, Foliage: oapi.FoliageGrass}, -6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, defs.TempModifier(tt.env))
		})
	}
}

func TestTileEnvDefs_ShelteredWorldTemp(t *testing.T) {
	t.Parallel()

	defs := TileEnvDefs{Shelters: map[oapi.Shelter]ShelterDef{
		oapi.Outdoor: {AnchorTemp: 10, InfluencePercent: 100},
		oapi.Partial: {AnchorTemp: 10, InfluencePercent: 75},
		oapi.Indoor:  {AnchorTemp: 10, InfluencePercent: 50},
	}}
	worldTemps := []int{-31, -10, -3, 10, 13, 25}
	got := map[oapi.Shelter][]int{}
	for _, s := range []oapi.Shelter{oapi.Outdoor, oapi.Partial, oapi.Indoor, ""} {
		for _, w := range worldTemps {
			got[s] = append(got[s], defs.ShelteredWorldTemp(s, w))
		}
	}
	assert.Equal(t, map[oapi.Shelter][]int{
		oapi.Outdoor: {-31, -10, -3, 10, 13, 25},
		oapi.Partial: {-20, -5, 1, 10, 12, 21},
		oapi.Indoor:  {-10, 0, 4, 10, 11, 17},
		"":           {-31, -10, -3, 10, 13, 25},
	}, got, "屋内は 10 + (w-10)/2、半屋外は 10 + (w-10)*3/4、表に無い囲われは屋外と同じ")
}
