package raw

import (
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/require"
)

// makeInteriorRaws は内装の Ref と生活感の抽選表を持つ検証を通る raws を作り、mutate で崩す
func makeInteriorRaws(mutate func(*rawschema.Raws)) rawschema.Raws {
	raws := rawschema.Raws{
		Props:      &[]rawschema.Prop{{Id: "bed"}},
		ItemGroups: &[]rawschema.ItemGroup{{Id: "food"}},
		InteriorRefs: &[]rawschema.InteriorRef{
			{Id: "bed", Prop: new("bed"), Placement: new(rawschema.FarFromDoor)},
			{Id: "snacks", ItemGroup: new("food")},
			{Id: "litter", Placement: new(rawschema.FullArea)},
		},
		InteriorProfile: &rawschema.InteriorProfile{
			Damages:   []rawschema.DamageChoice{{Id: rawschema.BuildingDamageIntact, Weight: 1}, {Id: rawschema.BuildingDamageMinor, Weight: 1}, {Id: rawschema.BuildingDamageMajor, Weight: 1}},
			Clutters:  []rawschema.ClutterChoice{{Id: rawschema.Tidy, Weight: 1}, {Id: rawschema.Messy, Weight: 1}, {Id: rawschema.Filthy, Weight: 1}},
			Variants:  []rawschema.VariantChoice{{Id: rawschema.Standard, Weight: 1}, {Id: rawschema.Abandoned, Weight: 1}, {Id: rawschema.UnderConstruction, Weight: 1}, {Id: rawschema.Survivalist, Weight: 1}, {Id: rawschema.Hoarder, Weight: 1}},
			Densities: []rawschema.DensityChoice{{Factor: 10, Weight: 1}},
			Heroes:    []rawschema.HeroChoice{{Id: "reward", Weight: 1, Refs: []string{"bed"}}},
			Sites: []rawschema.SiteChoice{
				{Id: rawschema.Detached, HomeWeight: 1},
				{Id: rawschema.Shopfront, ShopWeight: 1},
				{Id: rawschema.Roadside, ShopWeight: 1},
			},
			Facade: rawschema.FacadeRefs{Window: "bed", Shutter: "bed", Sign: "bed"},
			Lot:    rawschema.LotRefs{Fence: "bed", Plant: "bed", Vending: "bed"},
		},
	}
	mutate(&raws)
	return raws
}

func TestValidateInteriorRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rawschema.Raws)
		wantErr error
	}{
		{
			name:   "参照が実在し抽選表が enum の全値を持てば通る",
			mutate: func(*rawschema.Raws) {},
		},
		{
			name: "Ref の id 重複",
			mutate: func(r *rawschema.Raws) {
				*r.InteriorRefs = append(*r.InteriorRefs, rawschema.InteriorRef{Id: "bed"})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "Ref の prop が実在しない",
			mutate: func(r *rawschema.Raws) {
				(*r.InteriorRefs)[0].Prop = new("sofa")
			},
			wantErr: errInteriorRefPropUndefined,
		},
		{
			name: "Ref の item group が実在しない",
			mutate: func(r *rawschema.Raws) {
				(*r.InteriorRefs)[1].ItemGroup = new("drinks")
			},
			wantErr: errInteriorRefGroupUndefined,
		},
		{
			name: "損傷段階の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				r.InteriorProfile.Damages = r.InteriorProfile.Damages[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "変種の行が重複する",
			mutate: func(r *rawschema.Raws) {
				r.InteriorProfile.Variants = append(r.InteriorProfile.Variants, rawschema.VariantChoice{Id: rawschema.Hoarder, Weight: 1})
			},
			wantErr: errDuplicateID,
		},
		{
			name: "敷地の類型の行が欠ける",
			mutate: func(r *rawschema.Raws) {
				r.InteriorProfile.Sites = r.InteriorProfile.Sites[1:]
			},
			wantErr: errEnumRowMissing,
		},
		{
			name: "店の敷地の重みが無い",
			mutate: func(r *rawschema.Raws) {
				for i := range r.InteriorProfile.Sites {
					r.InteriorProfile.Sites[i].ShopWeight = 0
				}
			},
			wantErr: errSiteWeightZero,
		},
		{
			name: "外構の Ref が prop を持たない",
			mutate: func(r *rawschema.Raws) {
				r.InteriorProfile.Lot.Vending = "litter"
			},
			wantErr: errInteriorProfileRefUndefined,
		},
		{
			name: "外皮の Ref が実在しない",
			mutate: func(r *rawschema.Raws) {
				r.InteriorProfile.Facade.Sign = "billboard"
			},
			wantErr: errInteriorProfileRefUndefined,
		},
		{
			name: "hero の Ref が prop を持たない",
			mutate: func(r *rawschema.Raws) {
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
