package interior

// 外皮 FacadePass。分割文法の後、街路側の前壁へ窓・シャッター・看板を付ける。外から見た自然さの多くは「窓が
// 通りに並ぶ・店が正面に看板」という単層平面で完結する性質が占める。前壁の外側へ prop を載せるだけで、閉じた
// 箱だった建物が「正面のある建物」に見える。docs/design/20260725_70.md 追記その4 収穫2・追記その13。窓と
// シャッターは仮画像で、実スプライトは今後差し替える。

// facadeElements は街路側の前壁に付ける外皮 prop を返す。前庭ぶん内寄せした建物の街路側の壁へ、入口と角を
// 避けて窓を等間隔に並べる。廃業した店はシャッターを下ろす。店は入口脇に看板を出す。prop は壁タイルの上に
// 載り、VRT と overworld が同じ表で描き spawn するので乖離しない。
func facadeElements(s Site, facility string, dmg damageLevel) []Placed {
	lo, hi, fixed, horiz := frontWallSpan(s.Building, frontSide(s))
	doorAxis := s.Door.X
	if !horiz {
		doorAxis = s.Door.Y
	}
	winRef := "window"
	if isShop(facility) && dmg == dmgMajor {
		winRef = "shutter" // 廃業した店はシャッターを下ろす
	}

	var out []Placed
	for a := lo; a <= hi; a++ {
		if abs(a-doorAxis) <= 1 { // 入口の近くは開けない
			continue
		}
		if (a-lo)%3 != 1 { // 角から1つ内で始め、3タイルおきに1枚
			continue
		}
		out = append(out, facadeAt(a, fixed, horiz, winRef))
	}
	// 店は入口脇に看板を出す。ポスアポ日本の商店街の一番安い説得力
	if isShop(facility) {
		if a := doorAxis + 2; a <= hi {
			out = append(out, facadeAt(a, fixed, horiz, "sign"))
		} else if a := doorAxis - 2; a >= lo {
			out = append(out, facadeAt(a, fixed, horiz, "sign"))
		}
	}
	return out
}

// isShop は看板とシャッターを付ける店かを返す。骨董品店も店に含める。
func isShop(facility string) bool { return facility == facStore || facility == facAntique }

// facadeAt は壁に沿う軸の座標 a と固定座標 fixed から外皮 prop を1つ作る。horiz は前壁が横方向(北/南)か。
func facadeAt(a, fixed int, horiz bool, ref string) Placed {
	if horiz {
		return Placed{Kind: KindDecor, Ref: ref, Pos: Vec{X: a, Y: fixed}}
	}
	return Placed{Kind: KindDecor, Ref: ref, Pos: Vec{X: fixed, Y: a}}
}

// frontSide は建物が footprint から内寄せされた辺、すなわち街路に面する前面の辺を返す。insetBuilding は入口側の
// 一辺だけ内寄せするので、footprint と縁がずれた辺がちょうど1つある。内寄せが無い建物は辺がずれないので、
// 入口のある辺を前面にする。既定の一辺へ倒すと入口と逆の壁に窓や塀を付けてしまうため、doorSide で補う。
func frontSide(s Site) side {
	b, f := s.Building, s.Footprint
	switch {
	case b.Y > f.Y:
		return sideNorth
	case b.Y+b.H < f.Y+f.H:
		return sideSouth
	case b.X > f.X:
		return sideWest
	case b.X+b.W < f.X+f.W:
		return sideEast
	}
	return doorSide(b, s.Door)
}

// frontWallSpan は前壁のタイル列を、壁に沿う軸の範囲[lo,hi](角を除く)・固定座標 fixed・横方向かで返す。
func frontWallSpan(b Rect, fside side) (lo, hi, fixed int, horiz bool) {
	switch fside {
	case sideNorth:
		return b.X + 1, b.X + b.W - 2, b.Y, true
	case sideSouth:
		return b.X + 1, b.X + b.W - 2, b.Y + b.H - 1, true
	case sideWest:
		return b.Y + 1, b.Y + b.H - 2, b.X, false
	default: // sideEast
		return b.Y + 1, b.Y + b.H - 2, b.X + b.W - 1, false
	}
}
