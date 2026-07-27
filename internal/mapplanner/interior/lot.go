package interior

// lot pass。建物を裸で地面に置かず、敷地を塀で囲い門で出入りさせ、前庭に外構を置く。自然な街並みに
// 区画分割アルゴリズムは要らず、本質は「建物が裸で地面に置かれない」こと。街路側の footprint 縁へ
// 塀を並べ、入口の軸に門の隙間を空け、前庭へ外構 prop を1つ置く。FacadePass と同じく壁でなく地面タイルの
// 上に載る prop なので、既存の spawn 経路にそのまま乗る。塀は仮画像。

// lotElements は敷地の外構 prop を返す。街路側の footprint 縁に塀を並べ、入口の軸を門として開ける。前庭には
// 店なら自販機、民家なら観葉を1つ置く。
func lotElements(s Site, facility FacilityKind) []Placed {
	fside := frontSide(s)
	lo, hi, edge := lotEdgeSpan(s.Footprint, fside)
	doorAxis := edge.along(s.Door)

	var out []Placed
	// 塀で敷地を囲い、入口の軸を門として開ける。商店街は街路に面して開くので塀を張らない
	if s.Type != siteShopfront {
		for a := lo; a <= hi; a++ {
			if abs(a-doorAxis) <= 1 { // 門の隙間。街路から前庭へ入る
				continue
			}
			out = append(out, Placed{Kind: KindDecor, Ref: "fence", Pos: edge.at(a)})
		}
	}
	// 前庭の外構。店は入口脇に自販機、民家は観葉。前庭のタイルにだけ置く
	if spot, ok := yardSpot(s, fside, doorAxis); ok {
		ref := "plant"
		if isShop(facility) {
			ref = "vending"
		}
		out = append(out, Placed{Kind: KindDecor, Ref: ref, Pos: spot})
	}
	return out
}

// lotEdgeSpan は footprint の街路側の縁を、縁に沿う軸の範囲[lo,hi]と縁のラインで返す。塀は角も含めて敷地を
// ぐるりと閉じるので corners を除かない。
func lotEdgeSpan(f Rect, fside side) (lo, hi int, edge tileLine) {
	switch fside {
	case sideNorth:
		return f.X, f.X + f.W - 1, tileLine{cross: f.Y, horiz: true}
	case sideSouth:
		return f.X, f.X + f.W - 1, tileLine{cross: f.Y + f.H - 1, horiz: true}
	case sideWest:
		return f.Y, f.Y + f.H - 1, tileLine{cross: f.X, horiz: false}
	default: // sideEast
		return f.Y, f.Y + f.H - 1, tileLine{cross: f.X + f.W - 1, horiz: false}
	}
}

// yardSpot は前庭の外構を置くタイルを返す。門の脇で塀の1マス内側の前庭タイルを試す。前庭でなければ置かない。
func yardSpot(s Site, fside side, doorAxis int) (Vec, bool) {
	f := s.Footprint
	var p Vec
	switch fside {
	case sideNorth:
		p = Vec{X: doorAxis + 2, Y: f.Y + 1}
	case sideSouth:
		p = Vec{X: doorAxis + 2, Y: f.Y + f.H - 2}
	case sideWest:
		p = Vec{X: f.X + 1, Y: doorAxis + 2}
	default: // sideEast
		p = Vec{X: f.X + f.W - 2, Y: doorAxis + 2}
	}
	if s.Garden[p] {
		return p, true
	}
	return Vec{}, false
}
