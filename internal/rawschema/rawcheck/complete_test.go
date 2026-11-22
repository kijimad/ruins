package rawcheck_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/rawschema/rawcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateComplete_部分のデータはValidateだけを通る(t *testing.T) {
	t.Parallel()
	raws := makeRefRaws(func(*rawschema.Raws) {})
	require.NoError(t, rawcheck.Validate(raws))
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.ValidateComplete(raws), &ce)
	assert.Contains(t, ce.Paths(), "#Complete.freshnessStages_complete.fresh")
}

func TestValidateComplete_揃っていないと位置で返す(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*rawschema.Raws)
		want   []string
	}{
		{
			name: "行が欠ける",
			mutate: func(r *rawschema.Raws) {
				*r.FreshnessStages = (*r.FreshnessStages)[:2]
			},
			want: []string{"#Complete.freshnessStages_complete.rotten"},
		},
		{
			// 重なると struct が衝突で止まり、同じ struct の欠けは併せて報告されない
			name: "idが重なる",
			mutate: func(r *rawschema.Raws) {
				(*r.FreshnessStages)[1].Id = rawschema.Fresh
			},
			want: []string{"#Complete.freshnessStages_complete.fresh"},
		},
		{
			name: "表が無い",
			mutate: func(r *rawschema.Raws) {
				r.Seasons = nil
			},
			want: []string{
				"#Complete.seasons_complete.autumn",
				"#Complete.seasons_complete.spring",
				"#Complete.seasons_complete.summer",
				"#Complete.seasons_complete.winter",
			},
		},
		{
			name: "天気の行の季節が欠ける",
			mutate: func(r *rawschema.Raws) {
				w := &(*r.Weathers)[1]
				w.Seasons = w.Seasons[:len(w.Seasons)-1]
			},
			want: []string{`#Complete.weatherSeasons_complete."1".winter`},
		},
		{
			name: "発症の欄が重なる",
			mutate: func(r *rawschema.Raws) {
				conds := *r.Conditions
				for i := range conds {
					if conds[i].Onset != nil && *conds[i].Onset == rawschema.Hunger {
						conds[i].Onset = new(rawschema.Cold)
					}
				}
			},
			want: []string{"#Complete.conditionOnsets_complete.cold"},
		},
		{
			name: "別の技能の熟練度が同じ属性を軽減する",
			mutate: func(r *rawschema.Raws) {
				for i := range *r.Skills {
					for j, p := range raw.PtrSlice((*r.Skills)[i].Proficiencies) {
						if p.Resists != nil && *p.Resists == rawschema.ElementTHUNDER {
							(*(*r.Skills)[i].Proficiencies)[j].Resists = new(rawschema.ElementFIRE)
						}
					}
				}
			},
			want: []string{"#Complete.resists_complete.FIRE"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raws, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
			require.NoError(t, err)
			tt.mutate(&raws)
			require.NoError(t, rawcheck.Validate(raws))
			var ce *rawcheck.Error
			require.ErrorAs(t, rawcheck.ValidateComplete(raws), &ce)
			assert.Equal(t, tt.want, ce.Paths())
		})
	}
}

func TestValidate_無属性を軽減する熟練度は位置で返す(t *testing.T) {
	t.Parallel()
	raws, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	(*(*raws.Skills)[0].Proficiencies)[0].Resists = new(rawschema.ElementNONE)
	var ce *rawcheck.Error
	require.ErrorAs(t, rawcheck.Validate(raws), &ce)
	assert.Equal(t, []string{"skills.0.proficiencies.0.resists"}, ce.Paths())
}
