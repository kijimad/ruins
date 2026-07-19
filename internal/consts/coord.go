package consts

// Numeric は座標で使用可能な数値型を定義する
type Numeric interface {
	~int | ~float64
}

// Coord は2次元座標を表すジェネリック型
type Coord[T Numeric] struct {
	X T
	Y T
}

// AbsTileX は東西の絶対タイル X 座標。
//
// 東へ進むほど無限に増える絶対軸で、帯ローカルの GridElement.X とは別物。GridElement.X は
// 常に 0..K*chunkW の有界。絶対と局所の取り違えを Go の型で弾くための別名型。
// 寒波前線の東端・帯原点・到達距離スコアはこの絶対軸で扱う。
// 南北はストリーミングせず高さ固定の帯なので、絶対軸は X のみで足りる。
//
// worldstream の帯ドライバも components の永続状態 SeamlessBand も同じ絶対軸を扱うため、
// 双方から import できる leaf の consts に置く。これで境界のキャストを無くせる。
type AbsTileX int
