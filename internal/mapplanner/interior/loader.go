package interior

import (
	"fmt"
	"sync"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// rawsPath は内装レシピを含む raw データの資材パス。activeContents が一度だけ読む。
const rawsPath = "metadata/entities/raw/raw.toml"

var (
	pkgContents     *ContentSet
	pkgContentsOnce sync.Once
)

// activeContents は現在の内装レシピ一式を返す。raw.toml から一度だけ組んで package に保持する。
// レシピは施設生成の共有データなので、呼び出しごとに渡さず package で一元管理して呼び出しを簡潔に保つ。
// raw は起動時に検証済みなので、ここでの読込失敗は不変条件違反として panic で露見させる。
func activeContents() *ContentSet {
	pkgContentsOnce.Do(func() {
		master, err := raw.LoadFromFile(rawsPath)
		if err != nil {
			panic(fmt.Sprintf("interior: load raw for contents: %v", err))
		}
		cs, err := LoadContents(master)
		if err != nil {
			panic(fmt.Sprintf("interior: build contents: %v", err))
		}
		pkgContents = cs
	})
	return pkgContents
}

// ContentSet はロードした内装レシピ一式。id から Content を、施設種別から主室変種と奥室カタログを引く。
// content_catalog.go と facility.go が Go 定数で持っていたレシピと写像を raw.toml のデータへ移す受け皿。
type ContentSet struct {
	byID          map[string]Content
	facilityMain  map[FacilityKind][]string
	facilityRooms map[FacilityKind]roomSet
}

// roomSet は1施設の奥室カタログ。役割名から content id を引き、カタログに無い役割は fallback へ落とす。
type roomSet struct {
	rooms    map[roleName]string
	fallback string
}

// LoadContents は oapi.Raws の内装レシピ定義から ContentSet を組む。ダイス表記のパース失敗だけを error に
// し、参照の実在は raw の ValidateReferences 側に委ねる。
func LoadContents(raws oapi.Raws) (*ContentSet, error) {
	cs := &ContentSet{
		byID:          make(map[string]Content),
		facilityMain:  make(map[FacilityKind][]string),
		facilityRooms: make(map[FacilityKind]roomSet),
	}
	for _, ic := range raw.PtrSlice(raws.InteriorContents) {
		c, err := toContent(ic)
		if err != nil {
			return nil, err
		}
		cs.byID[ic.Id] = c
	}
	for _, fc := range raw.PtrSlice(raws.FacilityContents) {
		cs.facilityMain[FacilityKind(fc.Facility)] = append([]string(nil), fc.Variants...)
	}
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		rooms := raw.PtrSlice(fr.Rooms)
		rc := roomSet{rooms: make(map[roleName]string, len(rooms)), fallback: fr.Fallback}
		for _, r := range rooms {
			rc.rooms[roleName(r.Role)] = r.Content
		}
		cs.facilityRooms[FacilityKind(fr.Facility)] = rc
	}
	return cs, nil
}

// facilityContent は施設種別の主室 content を seed で1変種引く。未割り当ての施設は generic へ落とす。
// 同じ施設でも複数の変種を持ち、seed で引くことで同じ店が薬局にも食料品店にもなる。
func (cs *ContentSet) facilityContent(facility FacilityKind, seed uint64) Content {
	variants := cs.facilityMain[facility]
	if len(variants) == 0 {
		variants = []string{"generic"}
	}
	id := variants[int(childSeed(seed, 9_000_000)%uint64(len(variants)))]
	return cs.byID[id].clone()
}

// roomContent は施設の役割別 content を引く。役割が奥室カタログに無ければ ok=false。
func (cs *ContentSet) roomContent(facility FacilityKind, role roleName) (Content, bool) {
	id, ok := cs.facilityRooms[facility].rooms[role]
	if !ok {
		return Content{}, false
	}
	return cs.byID[id].clone(), true
}

// backRoomContent は施設の奥室フォールバック content を引く。カタログに無い役割はここへ落とす。
func (cs *ContentSet) backRoomContent(facility FacilityKind) Content {
	return cs.byID[cs.facilityRooms[facility].fallback].clone()
}

// clone は Content をディープコピーする。cs.byID は共有レシピを1つずつ保持するので、返り値を
// applyDensity などが in-place で書き換えても共有元を壊さないよう、Groups/Items/Satellites/Offsets まで
// 複製する。
func (c Content) clone() Content {
	if c.Groups == nil {
		return c
	}
	groups := make([]Group, len(c.Groups))
	for gi, g := range c.Groups {
		items := make([]Stuff, len(g.Items))
		for ii, it := range g.Items {
			if it.Satellites != nil {
				sats := make([]Satellite, len(it.Satellites))
				for si, s := range it.Satellites {
					if s.Offsets != nil {
						s.Offsets = append([]Vec(nil), s.Offsets...)
					}
					sats[si] = s
				}
				it.Satellites = sats
			}
			items[ii] = it
		}
		g.Items = items
		groups[gi] = g
	}
	c.Groups = groups
	return c
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
