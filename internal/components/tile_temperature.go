package components

// ShelterType はタイルの囲われの度合いを表す。屋内外の判定に使い、温度には
// 世界温度の受け方を決める緩和度として効く。値は識別子で、そのまま℃にはならない
type ShelterType int

// 囲われの度合い
const (
	ShelterNone    ShelterType = 0  // 屋外
	ShelterPartial ShelterType = 5  // 半屋外。世界温度を中間の強さで受ける
	ShelterFull    ShelterType = 10 // 屋内。世界温度を緩和して受ける
)

// WaterType は水の種類を表す
// 値は気温修正値（°C）を直接表す
type WaterType int

// 水による気温修正値
const (
	WaterNone      WaterType = 0   // なし
	WaterNearby    WaterType = -5  // 水辺
	WaterSubmerged WaterType = -10 // 水中
)

// FoliageType は植生の種類を表す
// 値は気温修正値（°C）を直接表す
type FoliageType int

// 植生による気温修正値
const (
	FoliageNone   FoliageType = 0  // なし
	FoliageGrass  FoliageType = -1 // 草原
	FoliageForest FoliageType = -3 // 森
)

// TileTemperature はタイルの気温修正値を持つコンポーネント
// 各要因を個別に保持し、ホバー時に内訳を表示できるようにする
type TileTemperature struct {
	Shelter ShelterType
	Water   WaterType
	Foliage FoliageType
}

// Total は加算℃の気温修正の合計を返す。Shelter は加算℃でなく緩和度なので含まない
func (tt *TileTemperature) Total() int {
	return int(tt.Water) + int(tt.Foliage)
}
