package interior

import "github.com/kijimaD/ruins/internal/consts"

// testContent は id からロード済みレシピを引くテスト補助。旧 content_catalog の関数の代わりに、raw.toml から
// 組んだ activeContents を引く。返り値は clone で、テストが in-place で書き換えても共有元を壊さない。
func testContent(id string) Content {
	return activeContents().byID[id].clone()
}

// testRoomContents は施設の役割別 content をテスト用に map で返す。旧 houseRoomContents 等の代わり。
func testRoomContents(fac FacilityKind) map[roleName]Content {
	cs := activeContents()
	rs := cs.facilityRooms[fac]
	out := make(map[roleName]Content, len(rs.rooms))
	for role, id := range rs.rooms {
		out[role] = cs.byID[id].clone()
	}
	return out
}

// diningTableStuff は椅子を四辺へ束ねた食卓の Stuff。衛星配置のテスト専用フィクスチャで、本番レシピは
// raw.toml が持つ。旧 fixtures.go の diningTable をテストへ移したもの。
func diningTableStuff(placement Placement) Stuff {
	chair := func(offs ...Vec) Satellite {
		return Satellite{Kind: KindFurniture, Ref: "chair", Offsets: offs}
	}
	return Stuff{
		Kind: KindFurniture, Ref: "table", Placement: placement, Amount: consts.Dice{Base: 1, Sides: 1},
		Satellites: []Satellite{
			chair(Vec{X: 0, Y: -1}, Vec{X: -1, Y: -1}, Vec{X: 1, Y: -1}),
			chair(Vec{X: 0, Y: 1}, Vec{X: -1, Y: 1}, Vec{X: 1, Y: 1}),
			chair(Vec{X: -1, Y: 0}, Vec{X: -1, Y: -1}, Vec{X: -1, Y: 1}),
			chair(Vec{X: 1, Y: 0}, Vec{X: 1, Y: -1}, Vec{X: 1, Y: 1}),
		},
	}
}
