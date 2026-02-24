package components

// ShelterType は遮蔽状態を表す（室内/屋外）
// 値は気温修正値（°C）を直接表す
type ShelterType int

// 遮蔽による気温修正値
const (
	ShelterNone    ShelterType = 0  // 屋外（露出）
	ShelterPartial ShelterType = 5  // 半屋外
	ShelterFull    ShelterType = 10 // 屋内（完全遮蔽）
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

// Total は気温修正値の合計を返す
func (tt *TileTemperature) Total() int {
	return int(tt.Shelter) + int(tt.Water) + int(tt.Foliage)
}
