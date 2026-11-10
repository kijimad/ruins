package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/require"
)

// makeInteriorRaws は内装の Ref と生活感の抽選表を持つ検証を通る raws を作り、mutate で崩す
func makeInteriorRaws(mutate func(*oapi.Raws)) oapi.Raws {
	raws := oapi.Raws{
		Props:      &[]oapi.Prop{{Id: "bed"}},
		ItemGroups: &[]oapi.ItemGroup{{Id: "food"}},
		InteriorRefs: &[]oapi.InteriorRef{
			{Id: "bed", Prop: new("bed"), Placement: new(oapi.FarFromDoor)},
			{Id: "snacks", ItemGroup: new("food")},
			{Id: "litter", Placement: new(oapi.FullArea)},
		},
		InteriorProfile: &oapi.InteriorProfile{
			Damages:   []oapi.DamageChoice{{Id: oapi.BuildingDamageIntact, Weight: 1}, {Id: oapi.BuildingDamageMinor, Weight: 1}, {Id: oapi.BuildingDamageMajor, Weight: 1}},
			Clutters:  []oapi.ClutterChoice{{Id: oapi.Tidy, Weight: 1}, {Id: oapi.Messy, Weight: 1}, {Id: oapi.Filthy, Weight: 1}},
			Variants:  []oapi.VariantChoice{{Id: oapi.Standard, Weight: 1}, {Id: oapi.Abandoned, Weight: 1}, {Id: oapi.UnderConstruction, Weight: 1}, {Id: oapi.Survivalist, Weight: 1}, {Id: oapi.Hoarder, Weight: 1}},
			Densities: []oapi.DensityChoice{{Factor: 10, Weight: 1}},
			Heroes:    []oapi.HeroChoice{{Id: "reward", Weight: 1, Refs: []string{"bed"}}},
		},
	}
	mutate(&raws)
	return raws
}

func TestValidateInteriorRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*oapi.Raws)
		wantErr error
	}{
		{
			name:   "参照が実在し抽選表が enum の全値を持てば通る",
			mutate: func(*oapi.Raws) {},
		},
		{
			name: "Ref の id 重複",
			mutate: func(r *oapi.Raws) {
				*r.InteriorRefs = append(*r.InteriorRefs, oapi.InteriorRef{Id: "bed"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "Ref の prop が実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.InteriorRefs)[0].Prop = new("sofa")
			},
			wantErr: errInteriorRefPropUndefined,
		},
		{
			name: "Ref の item group が実在しない",
			mutate: func(r *oapi.Raws) {
				(*r.InteriorRefs)[1].ItemGroup = new("drinks")
			},
			wantErr: errInteriorRefGroupUndefined,
		},
		{
			name: "損傷段階の行が欠ける",
			mutate: func(r *oapi.Raws) {
				r.InteriorProfile.Damages = r.InteriorProfile.Damages[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "変種の行が重複する",
			mutate: func(r *oapi.Raws) {
				r.InteriorProfile.Variants = append(r.InteriorProfile.Variants, oapi.VariantChoice{Id: oapi.Hoarder, Weight: 1})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "hero の Ref が prop を持たない",
			mutate: func(r *oapi.Raws) {
				r.InteriorProfile.Heroes[0].Refs = []string{"litter"}
			},
			wantErr: errHeroRefUndefined,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateInteriorRefs(makeInteriorRaws(tt.mutate))
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}
