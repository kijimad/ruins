package raw

import (
	"image/color"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeWeatherRaws は天候と空腹・疲労の段階を持つ検証を通る raws を作り、mutate で崩す
func makeWeatherRaws(mutate func(*rawschema.Raws)) rawschema.Raws {
	seasons := func() []rawschema.WeatherSeasonWeight {
		return []rawschema.WeatherSeasonWeight{{Id: rawschema.Spring, Weight: 1}, {Id: rawschema.Summer, Weight: 1}, {Id: rawschema.Autumn, Weight: 1}, {Id: rawschema.Winter, Weight: 1}}
	}
	ids := []rawschema.WeatherId{rawschema.Clear, rawschema.Cloudy, rawschema.Rain, rawschema.Snow, rawschema.Blizzard, rawschema.ColdSnap}
	weathers := make([]rawschema.Weather, 0, len(ids))
	for _, id := range ids {
		weathers = append(weathers, rawschema.Weather{Id: id, SpellTurnsMin: 100, SpellTurnsMax: 200, Seasons: seasons()})
	}
	raws := rawschema.Raws{
		Weathers: &weathers,
		HungerStages: &[]rawschema.HungerStage{
			{Id: rawschema.HungerStageIdSatiated, From: 0.9},
			{Id: rawschema.HungerStageIdNormal, From: 0.6},
			{Id: rawschema.HungerStageIdHungry, From: 0.3},
			{Id: rawschema.HungerStageIdStarving, From: 0},
		},
		FatigueStages: &[]rawschema.FatigueStage{
			{Id: rawschema.FatigueStageIdRested, From: 0},
			{Id: rawschema.FatigueStageIdNormal, From: 0.3},
			{Id: rawschema.FatigueStageIdTired, From: 0.5},
			{Id: rawschema.FatigueStageIdExhausted, From: 0.8},
		},
		FreshnessStages: &[]rawschema.FreshnessStage{
			{Id: rawschema.Fresh, From: 0},
			{Id: rawschema.Stale, From: 1},
			{Id: rawschema.Rotten, From: 2},
		},
	}
	mutate(&raws)
	return raws
}

func TestValidateWeathersAndNeedStages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rawschema.Raws)
		wantErr error
	}{
		{
			name:   "enum の全値を持てば通る",
			mutate: func(*rawschema.Raws) {},
		},
		{
			name: "天候の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.Weathers = (*r.Weathers)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "天候の季節の重みが欠ける",
			mutate: func(r *rawschema.Raws) {
				(*r.Weathers)[0].Seasons = (*r.Weathers)[0].Seasons[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "天候の季節の重みが重複する",
			mutate: func(r *rawschema.Raws) {
				(*r.Weathers)[0].Seasons = append((*r.Weathers)[0].Seasons, rawschema.WeatherSeasonWeight{Id: rawschema.Winter, Weight: 1})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "スペルの下限が上限を超える",
			mutate: func(r *rawschema.Raws) {
				(*r.Weathers)[0].SpellTurnsMin = 300
			},
			wantErr: errSpellTurnsReversed,
		},
		{
			name: "空腹の段階が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.HungerStages = (*r.HungerStages)[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "疲労の段階に割合0が無い",
			mutate: func(r *rawschema.Raws) {
				(*r.FatigueStages)[0].From = 0.1
			},
			wantErr: errNeedStageNoZero,
		},
		{
			name: "鮮度の段階に0から始まるものが無い",
			mutate: func(r *rawschema.Raws) {
				(*r.FreshnessStages)[0].From = 1
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
			{Level: rawschema.HungerStageIdSatiated, Name: "Full", From: 0.95, ReachedLog: "%s is now full.", Badge: &color.RGBA{100, 200, 100, 255}},
			{Level: rawschema.HungerStageIdNormal, Name: "Normal", From: 0.66},
			{Level: rawschema.HungerStageIdHungry, Name: "Hungry", From: 0.33, Condition: gc.SeverityMinor, Badge: &color.RGBA{255, 200, 0, 255}},
			{Level: rawschema.HungerStageIdStarving, Name: "Starving", From: 0, Condition: gc.SeverityMedium, WakesSleeper: true, Badge: &color.RGBA{255, 50, 50, 255}},
		},
		Fatigue: []gc.FatigueStage{
			{Level: rawschema.FatigueStageIdRested, Name: "Rested", From: 0},
			{Level: rawschema.FatigueStageIdNormal, Name: "Normal", From: 0.3, CanSleep: true},
			{Level: rawschema.FatigueStageIdTired, Name: "Tired", From: 0.5, Condition: gc.SeverityMinor, CanSleep: true, Badge: &color.RGBA{255, 200, 0, 255}},
			{Level: rawschema.FatigueStageIdExhausted, Name: "Exhausted", From: 0.8, Condition: gc.SeverityMedium, CanSleep: true, Badge: &color.RGBA{255, 50, 50, 255}},
		},
		Freshness: []gc.FreshnessDef{
			{Level: rawschema.Fresh, From: 0, Name: "Fresh", NutritionPercent: 100, EatLog: "%s ate fresh %s."},
			{Level: rawschema.Stale, From: 1, Name: "Stale", NutritionPercent: 50, EatLog: "%s ate old %s.", Marker: true},
			{Level: rawschema.Rotten, From: 2, Name: "Rotten", NutritionPercent: 30, EatLog: "%s ate spoiled %s.", Marker: true, IllnessChancePercent: 25},
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
	defs, err := NewMaterialDefs(raws)
	require.NoError(t, err)
	disassembly := func(item string, per consts.Milligram) *gc.MaterialDisassembly {
		return &gc.MaterialDisassembly{Item: item, PerWeight: per}
	}
	assert.Equal(t, gc.MaterialDefs{
		rawschema.WOOD:    {Name: "Wood", HeatPerKg: 200, Disassembly: disassembly("branch", 300*consts.MilligramPerGram)},
		rawschema.PAPER:   {Name: "Paper", HeatPerKg: 300},
		rawschema.CLOTH:   {Name: "Cloth", HeatPerKg: 160, Disassembly: disassembly("rag", 100*consts.MilligramPerGram)},
		rawschema.LEATHER: {Name: "Leather", HeatPerKg: 120, Disassembly: disassembly("leather_scrap", 200*consts.MilligramPerGram)},
		rawschema.PLANT:   {Name: "Plant fiber", HeatPerKg: 120},
		rawschema.FOOD:    {Name: "Food", HeatPerKg: 60},
		rawschema.BONE:    {Name: "Bone", HeatPerKg: 40, Disassembly: disassembly("ancient_bone", 200*consts.MilligramPerGram)},
		rawschema.OIL:     {Name: "Oil", HeatPerKg: 1000},
		rawschema.COAL:    {Name: "Coal", HeatPerKg: 800},
		rawschema.PLASTIC: {Name: "Plastic", HeatPerKg: 240},
		rawschema.METAL:   {Name: "Metal", HeatPerKg: 0, Disassembly: disassembly("scrap_iron", consts.MilligramPerKg)},
		rawschema.STONE:   {Name: "Stone", HeatPerKg: 0, Disassembly: disassembly("stone", consts.MilligramPerKg)},
		rawschema.GLASS:   {Name: "Glass", HeatPerKg: 0},
		rawschema.CRYSTAL: {Name: "Crystal", HeatPerKg: 0, Disassembly: disassembly("crystal_powder", 100*consts.MilligramPerGram)},
		rawschema.CERAMIC: {Name: "Ceramic", HeatPerKg: 0},
		rawschema.LIQUID:  {Name: "Liquid", HeatPerKg: 0},
	}, defs)
}

func TestNewSkillDefs_道具の分類が伸ばすスキルの実データ(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	defs, err := NewSkillDefs(raws)
	require.NoError(t, err)
	assert.Equal(t, map[rawschema.ToolCategory]rawschema.SkillId{
		rawschema.Prying:    rawschema.SkillIdMechanic,
		rawschema.Precision: rawschema.SkillIdMechanic,
		rawschema.Cutting:   rawschema.SkillIdMechanic,
	}, defs.ToolSkills)
}

func TestValidateTileEffects(t *testing.T) {
	t.Parallel()

	full := func() rawschema.Raws {
		return rawschema.Raws{
			ShelterEffects: &[]rawschema.ShelterEffect{{Id: rawschema.Outdoor}, {Id: rawschema.Partial}, {Id: rawschema.Indoor}},
			WaterEffects:   &[]rawschema.WaterEffect{{Id: rawschema.Nearby, TempModifier: -5}, {Id: rawschema.Submerged, TempModifier: -10}},
			FoliageEffects: &[]rawschema.FoliageEffect{{Id: rawschema.FoliageGrass, TempModifier: -1}, {Id: rawschema.FoliageForest, TempModifier: -3}},
		}
	}
	tests := []struct {
		name    string
		mutate  func(*rawschema.Raws)
		wantErr error
	}{
		{name: "全値の行があれば通る", mutate: func(*rawschema.Raws) {}},
		{
			name:    "囲われの行が欠ける",
			mutate:  func(r *rawschema.Raws) { *r.ShelterEffects = (*r.ShelterEffects)[:2] },
			wantErr: errEnumRowMissing,
		},
		{
			name:    "水の影響の行が欠ける",
			mutate:  func(r *rawschema.Raws) { *r.WaterEffects = (*r.WaterEffects)[:1] },
			wantErr: errEnumRowMissing,
		},
		{
			name: "植生の行が重複する",
			mutate: func(r *rawschema.Raws) {
				*r.FoliageEffects = append(*r.FoliageEffects, rawschema.FoliageEffect{Id: rawschema.FoliageGrass})
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
		Shelters: map[rawschema.Shelter]gc.ShelterDef{
			rawschema.Outdoor: {Name: "Outdoor", AnchorTemp: 10, InfluencePercent: 100},
			rawschema.Partial: {Name: "Semi-outdoor", AnchorTemp: 10, InfluencePercent: 75, Covered: true},
			rawschema.Indoor:  {Name: "Indoor", AnchorTemp: 10, InfluencePercent: 50, Covered: true},
		},
		Water:   map[rawschema.WaterExposure]int{rawschema.Nearby: -5, rawschema.Submerged: -10},
		Foliage: map[rawschema.Foliage]int{rawschema.FoliageGrass: -1, rawschema.FoliageForest: -3},
	}, NewTileEnvDefs(raws))
}
