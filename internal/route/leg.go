package route

// Weight は積載重量。
type Weight int

// Supply は1レグで消費する供給量。食料・燃料は束ねず独立に扱う（緩さ4原則）。
type Supply struct {
	Food int
	Fuel int
}

// LegResult は1レグ（辺）の踏破結果。純関数 ResolveLeg が返す。
// システム側はこれを適用するだけ（供給減算・体温変動・寒波前進・遭遇判定）。
type LegResult struct {
	Cost            Supply // 消費する供給（食料・燃料）
	TempDelta       int    // 体温変化（負＝冷える）
	EncounterChance int    // 遭遇（襲撃）確率（%）。ダイスは呼び出し側が振る
	FrontAdvance    int    // 寒波前線の前進量（＝面数）。累積面数で位置と比較する
}

// 供給消費の係数。
const (
	FoodPerFaceBase   = 2  // 1面あたりの基本食料消費
	WeightPerFoodUnit = 20 // この重量ごとに1面の食料消費が +1（運搬役が積荷を食う）
	FuelPerFace       = 1  // 1面あたりの基本燃料消費
)

// ResolveLeg は辺の踏破結果を算出する純関数。積載が重いほど1面の食料消費が増える
// （§14「運搬役が積荷を食う＝物量で頂点」）。ECS/Ebiten 非依存でテストできる心臓部。
func ResolveLeg(edge Edge, load Weight) LegResult {
	foodPerFace := FoodPerFaceBase + int(load)/WeightPerFoodUnit
	return LegResult{
		Cost: Supply{
			Food: edge.Faces * foodPerFace,
			Fuel: edge.Faces * FuelPerFace,
		},
		TempDelta:       edge.Type.tempDelta(),
		EncounterChance: edge.Type.encounterChance(),
		FrontAdvance:    edge.Faces,
	}
}
