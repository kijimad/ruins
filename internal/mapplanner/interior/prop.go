package interior

import (
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// interiorRef は raw の interiorRefs から Ref の行を引く。内装の Ref は数十件規模なので線形走査で足りる
func interiorRef(raws rawschema.Raws, ref string) (rawschema.InteriorRef, bool) {
	for _, r := range raw.PtrSlice(raws.InteriorRefs) {
		if r.Id == ref {
			return r, true
		}
	}
	return rawschema.InteriorRef{}, false
}

// PropRawName は Ref に対応する raw prop 名と、それが in-game に spawn されるかを返す。overworld の spawn と
// VRT の描画がこの1関数を共有し、片方だけが描く/置くという乖離が構造的に起きないようにする。
// interiorRefs に prop を持たない Ref は spawn されず描かれない。
func PropRawName(raws rawschema.Raws, ref string) (string, bool) {
	r, ok := interiorRef(raws, ref)
	if !ok || r.Prop == nil {
		return "", false
	}
	return *r.Prop, true
}

// LootGroupName は loot Ref に対応する raw item group 名と、spawn されるかを返す。overworld の床 loot spawn と
// VRT の描画がこの1関数を共有し、片方だけが置く、描くという乖離が構造的に起きないようにする。
func LootGroupName(raws rawschema.Raws, ref string) (string, bool) {
	r, ok := interiorRef(raws, ref)
	if !ok || r.ItemGroup == nil {
		return "", false
	}
	return *r.ItemGroup, true
}

// defaultPlacement は Ref の既定の置き方を返す。interiorRefs に無ければ PlaceDefault を返す
func defaultPlacement(raws rawschema.Raws, ref string) rawschema.Placement {
	r, ok := interiorRef(raws, ref)
	if !ok || r.Placement == nil {
		return PlaceDefault
	}
	return *r.Placement
}
