package rawcheck

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComplete_必須のキーが列挙の値と一致する(t *testing.T) {
	t.Parallel()
	v, err := unify(rawschema.Raws{})
	require.NoError(t, err)

	elements := []string{}
	for _, e := range rawschema.ElementValues() {
		if e != rawschema.ElementNONE {
			elements = append(elements, string(e))
		}
	}
	// データを重ねずに引く。データの行は欄として現れ、必須のキーの欠けを覆い隠す
	tests := []struct {
		path string
		want []string
	}{
		{"freshnessStages_complete", valueStrings(rawschema.FreshnessStageIdValues())},
		{"timesOfDay_complete", valueStrings(rawschema.TimeOfDayIdValues())},
		{"seasons_complete", valueStrings(rawschema.SeasonIdValues())},
		{"pointFeatures_complete", valueStrings(rawschema.PointFeatureKindValues())},
		{"weathers_complete", valueStrings(rawschema.WeatherIdValues())},
		{"shelterEffects_complete", valueStrings(rawschema.ShelterValues())},
		{"waterEffects_complete", valueStrings(rawschema.WaterExposureValues())},
		{"hungerStages_complete", valueStrings(rawschema.HungerStageIdValues())},
		{"fatigueStages_complete", valueStrings(rawschema.FatigueStageIdValues())},
		{"proficiencies_complete", valueStrings(rawschema.ProficiencyIdValues())},
		{"attackTypes_complete", valueStrings(rawschema.AttackCategoryValues())},
		{"conditionOnsets_complete", valueStrings(rawschema.ConditionOnsetValues())},
		{"mapGlyphChunks_complete", valueStrings(rawschema.GlyphChunkValues())},
		{"resists_complete", elements},
	}

	// #Complete に検査を足してこの表に行を足し忘れると突き合わされないので、欄の集合も突き合わせる。
	// 天気の行ごとの季節は seasons_complete と同じ #SeasonKeys を使うので表に持たない
	paths := make([]string, 0, 1+len(tests))
	paths = append(paths, "weatherSeasons_complete")
	for _, tt := range tests {
		paths = append(paths, tt.path)
	}
	fields, err := v.LookupPath(cue.ParsePath(completePath)).Fields()
	require.NoError(t, err)
	var declared []string
	for fields.Next() {
		declared = append(declared, fields.Selector().Unquoted())
	}
	assert.ElementsMatch(t, paths, declared)

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			it, err := v.LookupPath(cue.ParsePath(completePath + "." + tt.path)).Fields(cue.Optional(true))
			require.NoError(t, err)
			var got []string
			for it.Next() {
				got = append(got, it.Selector().Unquoted())
			}
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func valueStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}
