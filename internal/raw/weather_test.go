package raw

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeWeatherRaws は天候と空腹・疲労の段階を持つ検証を通る raws を作り、mutate で崩す
func makeWeatherRaws(mutate func(*oapi.Raws)) oapi.Raws {
	seasons := func() []oapi.WeatherSeasonWeight {
		return []oapi.WeatherSeasonWeight{{Id: oapi.Spring, Weight: 1}, {Id: oapi.Summer, Weight: 1}, {Id: oapi.Autumn, Weight: 1}, {Id: oapi.Winter, Weight: 1}}
	}
	ids := []oapi.WeatherId{oapi.Clear, oapi.Cloudy, oapi.Rain, oapi.Snow, oapi.Blizzard, oapi.ColdSnap}
	weathers := make([]oapi.Weather, 0, len(ids))
	for _, id := range ids {
		weathers = append(weathers, oapi.Weather{Id: id, SpellTurnsMin: 100, SpellTurnsMax: 200, Seasons: seasons()})
	}
	raws := oapi.Raws{
		Weathers: &weathers,
		HungerStages: &[]oapi.HungerStage{
			{Id: oapi.HungerStageIdSatiated, From: 0.9},
			{Id: oapi.HungerStageIdNormal, From: 0.6},
			{Id: oapi.HungerStageIdHungry, From: 0.3},
			{Id: oapi.HungerStageIdStarving, From: 0},
		},
		FatigueStages: &[]oapi.FatigueStage{
			{Id: oapi.FatigueStageIdRested, From: 0},
			{Id: oapi.FatigueStageIdNormal, From: 0.3},
			{Id: oapi.FatigueStageIdTired, From: 0.5},
			{Id: oapi.FatigueStageIdExhausted, From: 0.8},
		},
	}
	mutate(&raws)
	return raws
}

func TestValidateWeathersAndNeedStages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{
			name:   "enum の全値を持てば通る",
			mutate: func(*oapi.Raws) {},
		},
		{
			name: "天候の行が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.Weathers = (*r.Weathers)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "天候の季節の重みが欠ける",
			mutate: func(r *oapi.Raws) {
				(*r.Weathers)[0].Seasons = (*r.Weathers)[0].Seasons[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "天候の季節の重みが重複する",
			mutate: func(r *oapi.Raws) {
				(*r.Weathers)[0].Seasons = append((*r.Weathers)[0].Seasons, oapi.WeatherSeasonWeight{Id: oapi.Winter, Weight: 1})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "スペルの下限が上限を超える",
			mutate: func(r *oapi.Raws) {
				(*r.Weathers)[0].SpellTurnsMin = 300
			},
			wantErr: errSpellTurnsReversed,
		},
		{
			name: "空腹の段階が欠ける",
			mutate: func(r *oapi.Raws) {
				*r.HungerStages = (*r.HungerStages)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "疲労の段階に割合0が無い",
			mutate: func(r *oapi.Raws) {
				(*r.FatigueStages)[0].From = 0.1
			},
			wantErr: errNeedStageNoZero,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := makeWeatherRaws(tt.mutate)
			err := validateWeathers(raws)
			if err == nil {
				err = validateNeedStages(raws)
			}
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewWeatherDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs := NewWeatherDefs(raws)

	got := map[gc.WeatherKind]gc.WeatherDef{}
	for k := range gc.NumWeatherKind {
		got[gc.WeatherKind(k)] = defs.Def(gc.WeatherKind(k))
	}
	assert.Equal(t, map[gc.WeatherKind]gc.WeatherDef{
		gc.WeatherClear:    {Effect: gc.WeatherEffect{TempModifier: 2, VisionDelta: 0}, Severity: 0, SpellTurnsMin: 2250, SpellTurnsMax: 4500, SeasonWeights: [4]float64{1.4, 1.8, 1.1, 0.6}},
		gc.WeatherCloudy:   {Effect: gc.WeatherEffect{TempModifier: 0, VisionDelta: 0}, Severity: 0, SpellTurnsMin: 1500, SpellTurnsMax: 3000, SeasonWeights: [4]float64{1.2, 1.0, 1.2, 0.9}},
		gc.WeatherRain:     {Effect: gc.WeatherEffect{TempModifier: -2, VisionDelta: -6}, Severity: 0, SpellTurnsMin: 750, SpellTurnsMax: 2250, SeasonWeights: [4]float64{0.9, 1.1, 0.8, 0.2}},
		gc.WeatherSnow:     {Effect: gc.WeatherEffect{TempModifier: -3, VisionDelta: -10}, Severity: 1, SpellTurnsMin: 1200, SpellTurnsMax: 2250, SeasonWeights: [4]float64{0.8, 0.3, 1.0, 1.3}},
		gc.WeatherBlizzard: {Effect: gc.WeatherEffect{TempModifier: -8, VisionDelta: -28}, Severity: 2, SpellTurnsMin: 450, SpellTurnsMax: 900, SeasonWeights: [4]float64{0.3, 0.1, 0.5, 1.2}},
		gc.WeatherColdSnap: {Effect: gc.WeatherEffect{TempModifier: -15, VisionDelta: 0}, Severity: 2, SpellTurnsMin: 1500, SpellTurnsMax: 3000, SeasonWeights: [4]float64{0.1, 0.0, 0.2, 0.8}},
	}, got)
}

func TestNewNeedDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs := NewNeedDefs(raws)

	assert.Equal(t, gc.NewNeedDefs(
		[]gc.HungerStage{
			{Level: gc.HungerSatiated, From: 0.95},
			{Level: gc.HungerNormal, From: 0.66},
			{Level: gc.HungerHungry, From: 0.33, Condition: gc.SeverityMinor},
			{Level: gc.HungerStarving, From: 0, Condition: gc.SeverityMedium},
		},
		[]gc.FatigueStage{
			{Level: gc.FatigueRested, From: 0},
			{Level: gc.FatigueNormal, From: 0.3},
			{Level: gc.FatigueTired, From: 0.5, Condition: gc.SeverityMinor},
			{Level: gc.FatigueExhausted, From: 0.8, Condition: gc.SeverityMedium},
		},
	), defs)
}
