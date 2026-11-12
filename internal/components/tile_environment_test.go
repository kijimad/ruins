package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

func TestTileEnvDefs_TempModifier(t *testing.T) {
	t.Parallel()

	defs := TileEnvDefs{
		Water:   map[rawschema.WaterExposure]int{rawschema.Nearby: -5, rawschema.Submerged: -10},
		Foliage: map[rawschema.Foliage]int{rawschema.FoliageGrass: -1, rawschema.FoliageForest: -3},
	}
	tests := []struct {
		name string
		env  TileEnvironment
		want int
	}{
		{"全要素がゼロ値", TileEnvironment{}, 0},
		{"囲われは緩和度なので合計に含まない", TileEnvironment{Shelter: rawschema.Indoor}, 0},
		{"水辺で森", TileEnvironment{Water: rawschema.Nearby, Foliage: rawschema.FoliageForest}, -8},
		{"水中", TileEnvironment{Water: rawschema.Submerged}, -10},
		{"全要素の組み合わせ", TileEnvironment{Shelter: rawschema.Partial, Water: rawschema.Nearby, Foliage: rawschema.FoliageGrass}, -6},
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

	defs := TileEnvDefs{Shelters: map[rawschema.Shelter]ShelterDef{
		rawschema.Outdoor: {AnchorTemp: 10, InfluencePercent: 100},
		rawschema.Partial: {AnchorTemp: 10, InfluencePercent: 75},
		rawschema.Indoor:  {AnchorTemp: 10, InfluencePercent: 50},
	}}
	worldTemps := []int{-31, -10, -3, 10, 13, 25}
	got := map[rawschema.Shelter][]int{}
	for _, s := range []rawschema.Shelter{rawschema.Outdoor, rawschema.Partial, rawschema.Indoor, ""} {
		for _, w := range worldTemps {
			got[s] = append(got[s], defs.ShelteredWorldTemp(s, w))
		}
	}
	assert.Equal(t, map[rawschema.Shelter][]int{
		rawschema.Outdoor: {-31, -10, -3, 10, 13, 25},
		rawschema.Partial: {-20, -5, 1, 10, 12, 21},
		rawschema.Indoor:  {-10, 0, 4, 10, 11, 17},
		"":                {-31, -10, -3, 10, 13, 25},
	}, got, "屋内は 10 + (w-10)/2、半屋外は 10 + (w-10)*3/4、表に無い囲われは屋外と同じ")
}
