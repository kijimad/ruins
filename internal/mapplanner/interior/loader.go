package interior

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// genericContentID と flavorContentID はデータの参照でなくコードが直接引く content id。generic は未知施設や
// 未割り当ての主室・奥室のフォールバック、flavor は flavor machine が引く。データ間参照を見る
// ValidateReferences では守れないので、欠けていれば contentByID がロードでなく生成時に panic で露見させる。
const (
	genericContentID = "generic"
	flavorContentID  = "flavor"
)

// contentByID は id の内装レシピを raws から探して都度 interior.Content へ変換する。内装 content は数十件規模
// なので線形走査で足り、索引を持たず毎回新規に組む。返り値を applyDensity が in-place で書き換えても共有元が
// 無く clone が要らない。他ドメインの NewItemSpec と同じ「その場で引いて変換」の形。参照は raw の
// ValidateReferences で検証済みなので、ここでの未定義とダイス解析失敗は不変条件違反として panic で露見させる。
// 生成はテスト golden で走るので CI で捕まる。
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

// facilityContent は施設種別の主室 content を seed で1変種引く。未割り当ての施設は generic へ落とす。
// 同じ施設でも複数の変種を持ち、seed で引くことで同じ店が薬局にも食料品店にもなる。
func facilityContent(raws oapi.Raws, facility FacilityKind, seed uint64) Content {
	variants := facilityVariants(raws, facility)
	if len(variants) == 0 {
		variants = []string{genericContentID}
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

// backRoomContent は施設の奥室フォールバック content を引く。カタログに無い役割はここへ落とす。
// facilityRooms に無い未知施設は fallback が空になるので、facilityContent と同じく generic へ落として
// 空部屋の silent 生成を防ぐ。
func backRoomContent(raws oapi.Raws, facility FacilityKind) Content {
	id := genericContentID
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		if FacilityKind(fr.Facility) == facility {
			if fr.Fallback != "" {
				id = fr.Fallback
			}
			break
		}
	}
	return contentByID(raws, id)
}

// toContent は oapi の内装レシピを interior.Content へ変換する。抽選順に効く Groups と Items の並びは配列の
// 記述順をそのまま保つ。
func toContent(ic oapi.InteriorContent) (Content, error) {
	c := Content{ID: ic.Id}
	for _, g := range raw.PtrSlice(ic.Groups) {
		grp := Group{Style: GroupStyle(g.Style), Pick: derefInt32(g.Pick)}
		for _, s := range g.Items {
			amount, err := consts.ParseDice(s.Amount)
			if err != nil {
				return Content{}, fmt.Errorf("interior content %q ref %q amount: %w", ic.Id, s.Ref, err)
			}
			grp.Items = append(grp.Items, Stuff{
				Kind:       StuffKind(s.Kind),
				Ref:        s.Ref,
				Weight:     derefWeight(s.Weight),
				Chance:     derefInt32(s.Chance),
				Amount:     amount,
				Placement:  derefPlacement(s.Placement),
				Satellites: toSatellites(s.Satellites),
			})
		}
		c.Groups = append(c.Groups, grp)
	}
	return c, nil
}

// derefInt32 は optional な int32 を int へ。未設定は 0 で、消費側が 0 を既定へ倒す規約に従う。
func derefInt32(p *int32) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

// derefWeight は optional な重みを int へ。未設定は 0 で、消費側が 0 を 1 とみなす規約に従う。
func derefWeight(p *oapi.EntryWeight) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

// derefPlacement は optional な配置を Placement へ。未設定は空文字で、archetype の既定へ落ちる。
func derefPlacement(p *oapi.Placement) Placement {
	if p == nil {
		return ""
	}
	return Placement(*p)
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
