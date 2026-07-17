package worldstream

import "github.com/kijimaD/ruins/internal/consts"

// Front は寒波前線＝移動する極低温ゾーンを表す（設計 docs/design/20260717_60.md §3）。
//
// East は極低温ゾーンの東端の絶対タイル X。そこから西へ ColdWidth 分が極低温ゾーンで、
// 生存不能な極寒。ゾーンの西端 (East-ColdWidth) が「破棄＝進入不可ライン」を兼ねる。
// 実ターン経過でゆるやかに東進し、居座るとプレイヤーを飲む（一方向の空間的強制）。
// 破棄機構はこの1本のラインに統合される（シフト破棄と前線が別々に存在しない）。
type Front struct {
	// East は極低温ゾーン東端の絶対 X
	East AbsTileX
	// ColdWidth は極低温ゾーンの幅（タイル）
	ColdWidth consts.Tile
}

// ColdZoneWest は極低温ゾーンの西端＝破棄/進入不可ラインの絶対 X。
func (f Front) ColdZoneWest() AbsTileX {
	return f.East - AbsTileX(f.ColdWidth)
}

// InColdZone は絶対 X abs が極低温ゾーン (ColdZoneWest, East] 内かを返す。
// 西端は含まず（進入不可ラインの東側から極寒）、東端は含む。
func (f Front) InColdZone(abs AbsTileX) bool {
	return abs > f.ColdZoneWest() && abs <= f.East
}

// IsWestOfFront は絶対 X abs が破棄/進入不可ライン（極低温ゾーン西端）以西かを返す。
func (f Front) IsWestOfFront(abs AbsTileX) bool {
	return abs <= f.ColdZoneWest()
}

// Advance は前線を dx タイル東進させた新しい Front を返す（値は不変、コピーを返す）。
func (f Front) Advance(dx consts.Tile) Front {
	f.East += AbsTileX(dx)
	return f
}
