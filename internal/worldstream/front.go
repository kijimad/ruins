package worldstream

import "github.com/kijimaD/ruins/internal/consts"

// Front は寒波前線、すなわち移動する極低温ゾーンを表す。
//
// East は極低温ゾーンの東端の絶対タイル X。そこから西へ ColdWidth 分が極低温ゾーンで、
// 生存不能な極寒。ゾーンの西端 East-ColdWidth が破棄と進入不可ラインを兼ねる。
// 実ターン経過でゆるやかに東進し、居座るとプレイヤーを飲む。これが一方向の空間的強制になる。
// 破棄機構はこの1本のラインに統合され、シフト破棄と前線が別々に存在しない。
type Front struct {
	// East は極低温ゾーン東端の絶対 X
	East AbsTileX
	// ColdWidth は極低温ゾーンの幅。タイル単位
	ColdWidth consts.Tile
}

// ColdZoneWest は極低温ゾーンの西端＝破棄/進入不可ラインの絶対 X。
func (f Front) ColdZoneWest() AbsTileX {
	return f.East - AbsTileX(f.ColdWidth)
}

// InColdZone は絶対 X abs が極低温ゾーン (ColdZoneWest, East] 内かを返す。
// 西端は含まない。進入不可ラインの東側から極寒になる。東端は含む。
func (f Front) InColdZone(abs AbsTileX) bool {
	return abs > f.ColdZoneWest() && abs <= f.East
}

// IsWestOfFront は絶対 X abs が破棄と進入不可ライン、すなわち極低温ゾーン西端以西かを返す。
func (f Front) IsWestOfFront(abs AbsTileX) bool {
	return abs <= f.ColdZoneWest()
}

// Advance は前線を dx タイル東進させた新しい Front を返す。値は不変でコピーを返す。
func (f Front) Advance(dx consts.Tile) Front {
	f.East += AbsTileX(dx)
	return f
}

// FrontConfig は寒波前線の前進パラメータ。ラン開始時に決めて永続化する。
//
// 前線の現在位置は総経過ターン数から決定的に導出する。位置そのものは保存せず、
// この config と永続の総ターン数だけを保存すれば復元できる。ドリフトが起きない。
type FrontConfig struct {
	// StartEast はラン開始時の極低温ゾーン東端の絶対 X。プレイヤーの西に置いて背後から迫らせる
	StartEast AbsTileX
	// ColdWidth は極低温ゾーンの幅。タイル単位
	ColdWidth consts.Tile
	// AdvanceTurns はこの経過ターンごとに Step タイル東進する。0 以下なら前進しない
	AdvanceTurns int
	// Step は1回の前進量。タイル単位
	Step consts.Tile
}

// FrontAt は総経過ターン数 totalTurns 時点の Front を返す純関数。
// AdvanceTurns ごとに Step 前進する階段状の前進。負のターンは前進0として扱う。
func FrontAt(cfg FrontConfig, totalTurns int) Front {
	var advanced consts.Tile
	if cfg.AdvanceTurns > 0 && totalTurns > 0 {
		advanced = consts.Tile(totalTurns/cfg.AdvanceTurns) * cfg.Step
	}
	return Front{East: cfg.StartEast + AbsTileX(advanced), ColdWidth: cfg.ColdWidth}
}
