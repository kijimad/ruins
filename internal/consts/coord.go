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

// TileCenterToWorld はタイル座標を、そのタイル中心のワールドピクセル座標へ変換する。
// スプライトはタイル中心に合わせて配置するため中心へ半タイルぶんずらす。
func TileCenterToWorld(grid Coord[Tile]) Coord[Pixel] {
	half := TileSize / 2
	return Coord[Pixel]{X: Pixel(grid.X)*TileSize + half, Y: Pixel(grid.Y)*TileSize + half}
}

// WorldToScreen はワールドピクセル座標をカメラ変換して画面のスクリーンピクセル座標へ変換する。
// cameraPos はカメラ中心のワールド位置、scale はズーム率、screen は画面サイズ。
func WorldToScreen(world Coord[Pixel], cameraPos Coord[Pixel], scale float64, screenW, screenH int) Coord[ScreenPixel] {
	return Coord[ScreenPixel]{
		X: ScreenPixel(float64(world.X-cameraPos.X)*scale + float64(screenW)/2),
		Y: ScreenPixel(float64(world.Y-cameraPos.Y)*scale + float64(screenH)/2),
	}
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
