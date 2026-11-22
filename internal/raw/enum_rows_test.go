package raw

import (
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 列挙のまま残す物は、値の集合を Go との約束にスキーマの列挙で持ち、値ごとのデータを raw の行に持つ。
// 列挙の値ごとに行や欄の値がちょうど1つあることを、生成された値の一覧と実データで突き合わせる
func TestLoadFromFile_列挙の値ごとに行がちょうど1つある(t *testing.T) {
	t.Parallel()
	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	var proficiencies []rawschema.ProficiencyId
	var resists []rawschema.Element
	for _, s := range PtrSlice(raws.Skills) {
		for _, p := range PtrSlice(s.Proficiencies) {
			proficiencies = append(proficiencies, p.Id)
			if p.Resists != nil {
				resists = append(resists, *p.Resists)
			}
		}
	}
	var onsets []rawschema.ConditionOnset
	for _, c := range PtrSlice(raws.Conditions) {
		if c.Onset != nil {
			onsets = append(onsets, *c.Onset)
		}
	}
	var chunks []rawschema.GlyphChunk
	for _, g := range PtrSlice(raws.MapGlyphs) {
		if g.Chunk != nil {
			chunks = append(chunks, *g.Chunk)
		}
	}
	// 無属性は軽減の対象外なので、無属性を除く属性を1つずつ軽減する
	elements := slices.DeleteFunc(rawschema.ElementValues(), func(e rawschema.Element) bool { return e == rawschema.ElementNONE })

	type rows struct {
		name      string
		want, got any
	}
	weathers := PtrSlice(raws.Weathers)
	weatherSeasons := make([]rows, 0, len(weathers))
	for _, w := range weathers {
		weatherSeasons = append(weatherSeasons, rows{"天気 " + string(w.Id) + " の季節", rawschema.SeasonIdValues(), idsOf(w.Seasons, func(r rawschema.WeatherSeasonWeight) rawschema.SeasonId { return r.Id })})
	}
	tests := append([]rows{
		{"鮮度の段階", rawschema.FreshnessStageIdValues(), idsOf(PtrSlice(raws.FreshnessStages), func(r rawschema.FreshnessStage) rawschema.FreshnessStageId { return r.Id })},
		{"時間帯", rawschema.TimeOfDayIdValues(), idsOf(PtrSlice(raws.TimesOfDay), func(r rawschema.TimeOfDay) rawschema.TimeOfDayId { return r.Id })},
		{"季節", rawschema.SeasonIdValues(), idsOf(PtrSlice(raws.Seasons), func(r rawschema.Season) rawschema.SeasonId { return r.Id })},
		{"地図の点", rawschema.PointFeatureKindValues(), idsOf(PtrSlice(raws.PointFeatures), func(r rawschema.PointFeature) rawschema.PointFeatureKind { return r.Id })},
		{"天気", rawschema.WeatherIdValues(), idsOf(weathers, func(r rawschema.Weather) rawschema.WeatherId { return r.Id })},
		{"囲い", rawschema.ShelterValues(), idsOf(PtrSlice(raws.ShelterEffects), func(r rawschema.ShelterEffect) rawschema.Shelter { return r.Id })},
		{"水", rawschema.WaterExposureValues(), idsOf(PtrSlice(raws.WaterEffects), func(r rawschema.WaterEffect) rawschema.WaterExposure { return r.Id })},
		{"空腹の段階", rawschema.HungerStageIdValues(), idsOf(PtrSlice(raws.HungerStages), func(r rawschema.HungerStage) rawschema.HungerStageId { return r.Id })},
		{"疲労の段階", rawschema.FatigueStageIdValues(), idsOf(PtrSlice(raws.FatigueStages), func(r rawschema.FatigueStage) rawschema.FatigueStageId { return r.Id })},
		{"攻撃の種類", rawschema.AttackCategoryValues(), idsOf(PtrSlice(raws.AttackTypes), func(r rawschema.AttackType) rawschema.AttackCategory { return r.Id })},
		{"熟練度", rawschema.ProficiencyIdValues(), proficiencies},
		{"耐性の属性", elements, resists},
		{"発症", rawschema.ConditionOnsetValues(), onsets},
		{"地図の記号のチャンク", rawschema.GlyphChunkValues(), chunks},
	}, weatherSeasons...)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.ElementsMatch(t, tt.want, tt.got)
		})
	}
}

func idsOf[R any, ID any](rows []R, id func(R) ID) []ID {
	out := make([]ID, 0, len(rows))
	for _, r := range rows {
		out = append(out, id(r))
	}
	return out
}
