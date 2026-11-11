package raw

import (
	"image/color"
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
		got[gc.WeatherKind(k)] = defs.Kinds[gc.WeatherKind(k)]
	}
	assert.Equal(t, map[gc.WeatherKind]gc.WeatherDef{
		gc.WeatherClear:    {Name: "Clear", Effect: gc.WeatherEffect{TempModifier: 2, VisionDelta: 0}, Severity: 0, SpellTurnsMin: 2250, SpellTurnsMax: 4500, SeasonWeights: [gc.NumSeason]float64{1.4, 1.8, 1.1, 0.6}},
		gc.WeatherCloudy:   {Name: "Cloudy", Effect: gc.WeatherEffect{TempModifier: 0, VisionDelta: 0}, Severity: 0, SpellTurnsMin: 1500, SpellTurnsMax: 3000, SeasonWeights: [gc.NumSeason]float64{1.2, 1.0, 1.2, 0.9}},
		gc.WeatherRain:     {Name: "Rain", Effect: gc.WeatherEffect{TempModifier: -2, VisionDelta: -6}, Severity: 0, SpellTurnsMin: 750, SpellTurnsMax: 2250, SeasonWeights: [gc.NumSeason]float64{0.9, 1.1, 0.8, 0.2}},
		gc.WeatherSnow:     {Name: "Snow", Effect: gc.WeatherEffect{TempModifier: -3, VisionDelta: -10}, Severity: 1, SpellTurnsMin: 1200, SpellTurnsMax: 2250, SeasonWeights: [gc.NumSeason]float64{0.8, 0.3, 1.0, 1.3}},
		gc.WeatherBlizzard: {Name: "Blizzard", Effect: gc.WeatherEffect{TempModifier: -8, VisionDelta: -28}, Severity: 2, SpellTurnsMin: 450, SpellTurnsMax: 900, SeasonWeights: [gc.NumSeason]float64{0.3, 0.1, 0.5, 1.2}},
		gc.WeatherColdSnap: {Name: "Cold snap", Effect: gc.WeatherEffect{TempModifier: -15, VisionDelta: 0}, Severity: 2, SpellTurnsMin: 1500, SpellTurnsMax: 3000, SeasonWeights: [gc.NumSeason]float64{0.1, 0.0, 0.2, 0.8}},
	}, got)
}

func TestNewNeedDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs := NewNeedDefs(raws)

	assert.Equal(t, gc.NeedDefs{
		Hunger: []gc.HungerStage{
			{Level: oapi.HungerStageIdSatiated, Name: "Full", From: 0.95, ReachedLog: "%s is now full.", Badge: &color.RGBA{100, 200, 100, 255}},
			{Level: oapi.HungerStageIdNormal, Name: "Normal", From: 0.66},
			{Level: oapi.HungerStageIdHungry, Name: "Hungry", From: 0.33, Condition: gc.SeverityMinor, Badge: &color.RGBA{255, 200, 0, 255}},
			{Level: oapi.HungerStageIdStarving, Name: "Starving", From: 0, Condition: gc.SeverityMedium, WakesSleeper: true, Badge: &color.RGBA{255, 50, 50, 255}},
		},
		Fatigue: []gc.FatigueStage{
			{Level: oapi.FatigueStageIdRested, Name: "Rested", From: 0},
			{Level: oapi.FatigueStageIdNormal, Name: "Normal", From: 0.3, CanSleep: true},
			{Level: oapi.FatigueStageIdTired, Name: "Tired", From: 0.5, Condition: gc.SeverityMinor, CanSleep: true, Badge: &color.RGBA{255, 200, 0, 255}},
			{Level: oapi.FatigueStageIdExhausted, Name: "Exhausted", From: 0.8, Condition: gc.SeverityMedium, CanSleep: true, Badge: &color.RGBA{255, 50, 50, 255}},
		},
		Freshness: map[oapi.FreshnessStageId]gc.FreshnessDef{
			oapi.Fresh:  {Rank: 1, Name: "Fresh", NutritionPercent: 100, EatLog: "%s ate fresh %s."},
			oapi.Stale:  {Rank: 2, Name: "Stale", NutritionPercent: 50, EatLog: "%s ate old %s.", Marker: true},
			oapi.Rotten: {Rank: 3, Name: "Rotten", NutritionPercent: 30, EatLog: "%s ate spoiled %s.", Marker: true, IllnessChancePercent: 25},
		},
	}, defs)
}

func TestNewCalendarDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	assert.Equal(t, gc.CalendarDefs{
		TimesOfDay: [gc.NumTimeOfDay]gc.TimeOfDayDef{
			gc.TimeDawn:     {Name: "Dawn", TempModifier: 0, Daylight: 0.65, AmbientColor: [3]float64{1.0, 0.80, 0.72}},
			gc.TimeMorning:  {Name: "Morning", TempModifier: 5, Daylight: 0.72, AmbientColor: [3]float64{1.0, 0.96, 0.90}},
			gc.TimeDay:      {Name: "Noon", TempModifier: 10, Daylight: 0.95, AmbientColor: [3]float64{1.0, 1.0, 1.0}},
			gc.TimeEvening:  {Name: "Evening", TempModifier: 5, Daylight: 0.55, AmbientColor: [3]float64{1.0, 0.72, 0.52}},
			gc.TimeNight:    {Name: "Night", TempModifier: -5, Daylight: 0.14, AmbientColor: [3]float64{0.55, 0.60, 0.85}},
			gc.TimeMidnight: {Name: "Midnight", TempModifier: -10, Daylight: 0.06, AmbientColor: [3]float64{0.42, 0.48, 0.78}},
		},
		Seasons: [gc.NumSeason]gc.SeasonDef{
			gc.SeasonSpring: {Name: "Spring", Temp: 5},
			gc.SeasonSummer: {Name: "Summer", Temp: 22},
			gc.SeasonAutumn: {Name: "Autumn", Temp: 0},
			gc.SeasonWinter: {Name: "Winter", Temp: -30},
		},
	}, NewCalendarDefs(raws))
}

func TestNewMaterialDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	assert.Equal(t, gc.MaterialDefs{
		oapi.WOOD:    {Name: "Wood", HeatPerKg: 200},
		oapi.PAPER:   {Name: "Paper", HeatPerKg: 300},
		oapi.CLOTH:   {Name: "Cloth", HeatPerKg: 160},
		oapi.LEATHER: {Name: "Leather", HeatPerKg: 120},
		oapi.PLANT:   {Name: "Plant fiber", HeatPerKg: 120},
		oapi.FOOD:    {Name: "Food", HeatPerKg: 60},
		oapi.BONE:    {Name: "Bone", HeatPerKg: 40},
		oapi.OIL:     {Name: "Oil", HeatPerKg: 1000},
		oapi.COAL:    {Name: "Coal", HeatPerKg: 800},
		oapi.PLASTIC: {Name: "Plastic", HeatPerKg: 240},
		oapi.METAL:   {Name: "Metal", HeatPerKg: 0},
		oapi.STONE:   {Name: "Stone", HeatPerKg: 0},
		oapi.GLASS:   {Name: "Glass", HeatPerKg: 0},
		oapi.CRYSTAL: {Name: "Crystal", HeatPerKg: 0},
		oapi.CERAMIC: {Name: "Ceramic", HeatPerKg: 0},
		oapi.LIQUID:  {Name: "Liquid", HeatPerKg: 0},
	}, NewMaterialDefs(raws))
}

func TestNewSkillDefs_道具の分類が伸ばすスキルの実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs, err := NewSkillDefs(raws)
	require.NoError(t, err)
	assert.Equal(t, map[oapi.ToolCategory]oapi.SkillId{
		oapi.Prying:    oapi.SkillIdMechanic,
		oapi.Precision: oapi.SkillIdMechanic,
		oapi.Cutting:   oapi.SkillIdMechanic,
	}, defs.ToolSkills)
}

func TestValidateTileEffects(t *testing.T) {
	t.Parallel()

	full := func() oapi.Raws {
		return oapi.Raws{
			ShelterEffects: &[]oapi.ShelterEffect{{Id: oapi.Outdoor}, {Id: oapi.Partial}, {Id: oapi.Indoor}},
			WaterEffects:   &[]oapi.WaterEffect{{Id: oapi.Nearby, TempModifier: -5}, {Id: oapi.Submerged, TempModifier: -10}},
			FoliageEffects: &[]oapi.FoliageEffect{{Id: oapi.FoliageGrass, TempModifier: -1}, {Id: oapi.FoliageForest, TempModifier: -3}},
		}
	}
	tests := []struct {
		name    string
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{name: "全値の行があれば通る", mutate: func(*oapi.Raws) {}},
		{
			name:    "囲われの行が欠ける",
			mutate:  func(r *oapi.Raws) { *r.ShelterEffects = (*r.ShelterEffects)[:2] },
			wantErr: errEnumRowMissing,
		},
		{
			name:    "水の影響の行が欠ける",
			mutate:  func(r *oapi.Raws) { *r.WaterEffects = (*r.WaterEffects)[:1] },
			wantErr: errEnumRowMissing,
		},
		{
			name: "植生の行が重複する",
			mutate: func(r *oapi.Raws) {
				*r.FoliageEffects = append(*r.FoliageEffects, oapi.FoliageEffect{Id: oapi.FoliageGrass})
			},
			wantErr: errDuplicateID,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws := full()
			tt.mutate(&raws)
			err := validateTileEffects(raws)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewTileEnvDefs_実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	assert.Equal(t, gc.TileEnvDefs{
		Shelters: map[oapi.Shelter]gc.ShelterDef{
			oapi.Outdoor: {Name: "Outdoor", AnchorTemp: 10, InfluencePercent: 100},
			oapi.Partial: {Name: "Semi-outdoor", AnchorTemp: 10, InfluencePercent: 75, Covered: true},
			oapi.Indoor:  {Name: "Indoor", AnchorTemp: 10, InfluencePercent: 50, Covered: true},
		},
		Water:   map[oapi.WaterExposure]int{oapi.Nearby: -5, oapi.Submerged: -10},
		Foliage: map[oapi.Foliage]int{oapi.FoliageGrass: -1, oapi.FoliageForest: -3},
	}, NewTileEnvDefs(raws))
}
