package interior

import (
	"errors"
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// 内装レシピ引きの sentinel エラー。呼び出し側とテストが errors.Is で同定できるようにする。
var (
	errContentNotFound       = errors.New("interior content not found")
	errFacilityNotRegistered = errors.New("interior facility not registered")
	errFlavorContentMissing  = errors.New("interior flavor content missing")
)

// flavorContent は全施設共通のフレーバー装飾レシピを返す。施設レシピと直交する専用フィールドから引くので、
// 施設種別を取らない。raw に未設定なら生成側が既定へ倒せるよう error で返す。
func flavorContent(raws oapi.Raws) (Content, error) {
	if raws.FlavorContent == nil {
		return Content{}, errFlavorContentMissing
	}
	return toContent(*raws.FlavorContent)
}

// contentByID は id の内装レシピを raws から探して都度 interior.Content へ変換する。内装 content は数十件規模
// なので線形走査で足り、索引を持たず毎回新規に組む。返り値を applyDensity が in-place で書き換えても共有元が
// 無く clone が要らない。参照とダイスは raw の ValidateReferences がロード時に検証するので通常は成功するが、
// 壊れた raw で生成を落とさないよう未定義・解析失敗は error で返す。
func contentByID(raws oapi.Raws, id string) (Content, error) {
	for _, ic := range raw.PtrSlice(raws.InteriorContents) {
		if ic.Id == id {
			c, err := toContent(ic)
			if err != nil {
				return Content{}, fmt.Errorf("interior content %q: %w", id, err)
			}
			return c, nil
		}
	}
	return Content{}, fmt.Errorf("%q: %w", id, errContentNotFound)
}

// facilityContent は施設の主室 content を seed で1変種引く。同じ施設でも複数の変種を持つ。id は
// facilities 行の閉じた集合で全施設が登録済みなので通常は成功し、未登録は error で返す。
func facilityContent(raws oapi.Raws, id string, seed uint64) (Content, error) {
	variants := facilityVariants(raws, id)
	if len(variants) == 0 {
		return Content{}, fmt.Errorf("%q in facilityContents: %w", id, errFacilityNotRegistered)
	}
	// 9_000_000 は変種抽選専用の child index。同じ親 seed から引く他ストリームと無相関にする帯オフセット
	vid := variants[int(childSeed(seed, 9_000_000)%uint64(len(variants)))]
	return contentByID(raws, vid)
}

// facilityVariants は施設の主室変種 id 列を raws から引く。未割り当ての施設は nil。
func facilityVariants(raws oapi.Raws, id string) []string {
	for _, fc := range raw.PtrSlice(raws.FacilityContents) {
		if fc.Facility == id {
			return fc.Variants
		}
	}
	return nil
}

// roomContent は施設の役割別 content を引く。ok は役割がカタログに在ったかを表し err と直交する。役割が無ければ
// ok=false・err=nil、役割は在るが content 参照が壊れていれば ok=true・err!=nil。呼び出し側は ok||err で「引けたか
// 壊れているか」をまとめて次段へ委ね、どちらでもないときだけフォールバックへ進む。
func roomContent(raws oapi.Raws, id string, role roleName) (Content, bool, error) {
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		if fr.Facility != id {
			continue
		}
		for _, r := range raw.PtrSlice(fr.Rooms) {
			if roleName(r.Role) == role {
				c, err := contentByID(raws, r.Content)
				return c, true, err
			}
		}
		return Content{}, false, nil
	}
	return Content{}, false, nil
}

// backRoomContent は施設の奥室フォールバック content を引く。カタログに無い役割はここへ落とす。facility は
// 全種別が facilityRooms に登録済みなので通常は成功する。未登録は error で返す。
func backRoomContent(raws oapi.Raws, id string) (Content, error) {
	for _, fr := range raw.PtrSlice(raws.FacilityRooms) {
		if fr.Facility == id {
			return contentByID(raws, fr.Fallback)
		}
	}
	return Content{}, fmt.Errorf("%q in facilityRooms: %w", id, errFacilityNotRegistered)
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

// toSatellites は oapi の衛星束を interior.Satellite へ変換する。束が無ければ nil を返す。offsets の consts.Tile
// は負値も取り、anchor から上/左方向を表す。
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
