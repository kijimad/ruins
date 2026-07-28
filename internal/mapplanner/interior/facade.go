package interior

import (
	"strconv"

	"github.com/kijimaD/ruins/internal/consts"
)

// 外皮 FacadePass。分割文法の後、街路側の前壁へ窓・シャッター・看板を付ける。外から見た自然さの多くは「窓が
// 通りに並ぶ・店が正面に看板」という単層平面で完結する性質が占める。前壁の外側へ prop を載せるだけで、閉じた
// 箱だった建物が「正面のある建物」に見える。窓と
// シャッターは仮画像で、実スプライトは今後差し替える。

// facadeElements は街路側の前壁に付ける外皮 prop を返す。前庭ぶん内寄せした建物の街路側の壁へ、入口と角を
// 避けて窓を等間隔に並べる。廃業した店はシャッターを下ろす。店は入口脇に看板を出す。prop は壁タイルの上に
// 載り、VRT と overworld が同じ表で描き spawn するので乖離しない。
func facadeElements(s Site, facility FacilityKind, dmg damageLevel) []Placed {
	fside := frontSide(s)
	lo, hi, wall := frontWallSpan(s.Building, fside)
	doorAxis := wall.along(s.Door)
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
		// 窓とシャッターは壁付きの設備なので前壁のタイルに載せる
		out = append(out, Placed{Kind: KindDecor, Ref: winRef, Pos: wall.at(a)})
	}
	// 店は入口脇に看板を出す。ポスアポ日本の商店街の一番安い説得力。看板は独立物なので壁に埋めると不自然。
	// 前壁の1マス外の前庭タイルへ立て、店の正面に置く。前庭が無い建物では立てない
	if isShop(facility) {
		in := porchStep(fside)
		outward := Vec{X: -in.X, Y: -in.Y} // 建物の外向き
		for _, a := range [2]consts.Tile{doorAxis + 2, doorAxis - 2} {
			if a < lo || a > hi {
				continue
			}
			w := wall.at(a)
			yard := Vec{X: w.X + outward.X, Y: w.Y + outward.Y}
			if s.Garden[yard] {
				out = append(out, Placed{Kind: KindDecor, Ref: "sign", Pos: yard})
				break
			}
		}
	}
	return out
}

// isShop は看板とシャッターを付ける店かを返す。骨董品店も店に含める。
func isShop(facility FacilityKind) bool { return facility == facStore || facility == facAntique }

// tileLine は1本の軸に沿ったタイル列。cross は列の固定座標、horiz は列が横方向すなわち X に沿うか。壁や敷地縁の
// ように、固定座標と向きが常にセットで決まるものを1つの値にまとめる。along を渡すと列上の1タイルを返すので、
// 呼び出し側は列に沿う位置だけを渡せばよく、along と cross の取り違えが型で起きなくなる。
type tileLine struct {
	cross consts.Tile
	horiz bool
}

// at は列に沿う位置 along のタイル座標を返す。
func (l tileLine) at(along consts.Tile) Vec {
	if l.horiz {
		return Vec{X: along, Y: l.cross}
	}
	return Vec{X: l.cross, Y: along}
}

// along は v の列に沿う成分を取り出す。入口の軸座標を得るのに使う。
func (l tileLine) along(v Vec) consts.Tile {
	if l.horiz {
		return v.X
	}
	return v.Y
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
	// 内寄せが無い建物は辺がずれない。入口の辺を前面にするが、ポーチで入口を内側へ下げた後は s.Door が辺上に
	// 無いため doorSide では panic する。最も近い辺へ丸めて前面を選ぶ。
	return nearestSide(b, s.Door)
}

// frontWallSpan は前壁のタイル列を、壁に沿う軸の範囲[lo,hi](角を除く)と壁のラインで返す。
func frontWallSpan(b Rect, fside side) (lo, hi consts.Tile, wall tileLine) {
	switch fside {
	case sideNorth:
		return b.X + 1, b.X + b.W - 2, tileLine{cross: b.Y, horiz: true}
	case sideSouth:
		return b.X + 1, b.X + b.W - 2, tileLine{cross: b.Y + b.H - 1, horiz: true}
	case sideWest:
		return b.Y + 1, b.Y + b.H - 2, tileLine{cross: b.X, horiz: false}
	case sideEast:
		return b.Y + 1, b.Y + b.H - 2, tileLine{cross: b.X + b.W - 1, horiz: false}
	}
	panic("未知の side: " + strconv.Itoa(int(fside)))
}
