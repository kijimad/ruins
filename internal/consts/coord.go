package consts

import "fmt"

// Numeric は int または float64 を基底型とする数値型を表す汎用制約
type Numeric interface {
	~int | ~float64
}

// Coord は2次元座標を表すジェネリック型
type Coord[T Numeric] struct {
	X T
	Y T
}

// String は (x,y) 形式の文字列を返す。ログや座標表示の整形を一箇所に集約する。
func (c Coord[T]) String() string {
	return fmt.Sprintf("(%v,%v)", c.X, c.Y)
}

// Add は2つの座標を成分ごとに加算した座標を返す。
func (c Coord[T]) Add(o Coord[T]) Coord[T] {
	return Coord[T]{X: c.X + o.X, Y: c.Y + o.Y}
}

// Sub は成分ごとに減算した座標を返す。
func (c Coord[T]) Sub(o Coord[T]) Coord[T] {
	return Coord[T]{X: c.X - o.X, Y: c.Y - o.Y}
}

// TileCenterToWorld はタイル座標を、そのタイル中心のワールドピクセル座標へ変換する。
// スプライトはタイル中心に合わせて配置するため中心へ半タイルぶんずらす。
func TileCenterToWorld(grid Coord[Tile]) Coord[WorldPixel] {
	half := TileSize / 2
	return Coord[WorldPixel]{X: WorldPixel(grid.X)*TileSize + half, Y: WorldPixel(grid.Y)*TileSize + half}
}

// AbsTileY は南北の絶対タイル Y 座標。
//
// 北へ進むほど無限に減る絶対軸で、帯ローカルの GridElement.Y とは別物。GridElement.Y は
// 常に 0..rows*chunkH の有界。絶対と局所の取り違えを Go の型で弾くための別名型。
// 帯原点はこの絶対軸で扱う。北が上すなわち -Y なので、奥へ進むほど絶対 Y は負へ伸びる。
// 東西はストリーミングせず幅固定の帯なので、絶対軸は Y のみで足りる。
//
// worldstream の帯ドライバも components の永続状態 SeamlessBand も同じ絶対軸を扱うため、
// 双方から import できる leaf の consts に置く。これで境界のキャストを無くせる。
//
// 基底は int。奥行き計算は int(AbsTileY) で受けて演算するので、幅を int より狭い型へ変えると
// 無言の切り詰めが起きる。狭める必要が出たら int(...) の各キャストを見直すこと。
type AbsTileY int
