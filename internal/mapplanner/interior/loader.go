package interior

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// contentByID は id の内装レシピを raws から探して都度 interior.Content へ変換する。内装 content は数十件規模
// なので線形走査で足り、索引を持たず毎回新規に組む。返り値を applyDensity が in-place で書き換えても共有元が
// 無く clone が要らない。他ドメインの NewItemSpec と同じ「その場で引いて変換」の形。参照もダイス表記も raw の
// ValidateReferences がロード時に検証済みなので、ここでの未定義・解析失敗は起きえない。不変条件違反として
// panic で露見させる backstop に留める。flavor はコード直引きで validate 対象外なので、欠ければここで panic する。
func contentByID(raws oapi.Raws, id string) Content {
	for _, ic := range raw.PtrSlice(raws.InteriorContents) {
		if ic.Id == id {
			c, err := toContent(ic)
			if err != nil {
				panic(fmt.Sprintf("interior: content %q: %v", id, err))
			}
			return c
		}
	}
	panic(fmt.Sprintf("interior: content %q not found", id))
}

// facilityContent は施設種別の主室 content を seed で1変種引く。同じ施設でも複数の変種を持ち、seed で引く
// ことで同じ店が薬局にも食料品店にもなる。facility は overworld の閉じた enum で全種別が facilityContents に
// 登録済みなので、未登録は不変条件違反として panic で露見させる。
func facilityContent(raws oapi.Raws, facility FacilityKind, seed uint64) Content {
	variants := facilityVariants(raws, facility)
	if len(variants) == 0 {
		panic(fmt.Sprintf("interior: facility %q not registered in facilityContents", facility))
	}
	id := variants[int(childSeed(seed, 9_000_000)%uint64(len(variants)))]
	return contentByID(raws, id)
}

// facilityVariants は施設種別の主室変種 id 列を raws から引く。未割り当ての施設は nil。
func facilityVariants(raws oapi.Raws, facility FacilityKind) []string {
	for _, fc := range raw.PtrSlice(raws.FacilityContents) {
		if FacilityKind(fc.Facility) == facility {
			return fc.Variants
		}
	}
	return nil
}

// roomContent は施設の役割別 content を引く。役割が奥室カタログに無ければ ok=false。
func roomContent(raws oapi.Raws, facility FacilityKind, role roleName) (Content, bool) {
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		if FacilityKind(fr.Facility) != facility {
			continue
		}
		for _, r := range raw.PtrSlice(fr.Rooms) {
			if roleName(r.Role) == role {
				return contentByID(raws, r.Content), true
			}
		}
		return Content{}, false
	}
	return Content{}, false
}

// backRoomContent は施設の奥室フォールバック content を引く。カタログに無い役割はここへ落とす。facility は
// 全種別が facilityRooms に登録済みなので、未登録は不変条件違反として panic で露見させる。
func backRoomContent(raws oapi.Raws, facility FacilityKind) Content {
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		if FacilityKind(fr.Facility) == facility {
			return contentByID(raws, fr.Fallback)
		}
	}
	panic(fmt.Sprintf("interior: facility %q not registered in facilityRooms", facility))
}

// toContent は oapi の内装レシピを interior.Content へ変換する。抽選順に効く Groups と Items の並びは配列の
// 記述順をそのまま保つ。
func toContent(ic oapi.InteriorContent) (Content, error) {
	c := Content{ID: ic.Id}
	for _, g := range raw.PtrSlice(ic.Groups) {
		grp := Group{Style: GroupStyle(g.Style), Pick: int(deref(g.Pick))}
		for _, s := range g.Items {
			amount, err := consts.ParseDice(s.Amount)
			if err != nil {
				return Content{}, fmt.Errorf("interior content %q ref %q amount: %w", ic.Id, s.Ref, err)
			}
			grp.Items = append(grp.Items, Stuff{
				Kind:       StuffKind(s.Kind),
				Ref:        s.Ref,
				Weight:     int(deref(s.Weight)),
				Chance:     int(deref(s.Chance)),
				Amount:     amount,
				Placement:  Placement(deref(s.Placement)),
				Satellites: toSatellites(s.Satellites),
			})
		}
		c.Groups = append(c.Groups, grp)
	}
	return c, nil
}

// deref は optional なポインタを値へ落とす。未設定は型のゼロ値で、消費側が既定へ倒す規約に従う。呼び出し側が
// interior の型へ変換する。optional array の PtrSlice と対になるスカラー版。
func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// toSatellites は oapi の衛星束を interior.Satellite へ変換する。束が無ければ nil を返し、束のない Stuff の
// Go レシピと等価になるようにする。offsets の consts.Tile は負値も取り、anchor から上/左方向を表す。
func toSatellites(in *[]oapi.ContentSatellite) []Satellite {
	items := raw.PtrSlice(in)
	if len(items) == 0 {
		return nil
	}
	out := make([]Satellite, 0, len(items))
	for _, s := range items {
		sat := Satellite{Kind: StuffKind(s.Kind), Ref: s.Ref}
		for _, o := range s.Offsets {
			sat.Offsets = append(sat.Offsets, Vec{X: consts.Tile(o.X), Y: consts.Tile(o.Y)})
		}
		out = append(out, sat)
	}
	return out
}
