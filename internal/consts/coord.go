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
// 北が上すなわち -Y で、奥へ進むほど負へ無限に伸びる絶対軸。帯ローカルの GridElement.Y とは別物で、
// GridElement.Y は常に 0..rows*chunkH の有界。絶対と局所の取り違えを Go の型で弾くための別名型。
// 帯原点はこの絶対軸で扱う。東西はストリーミングせず幅固定なので、絶対軸は Y のみで足りる。
//
// worldstream の帯ドライバも components の永続状態 SeamlessBand も同じ絶対軸を扱うため、
// 双方から import できる leaf の consts に置く。これで境界のキャストを無くせる。
type AbsTileY int

// BandOriginY は北進チャンク数 northIndex と帯高 chunkH から帯の絶対原点 Y を返す。
// 帯ローカル Y=0 すなわち北端が絶対軸で指す位置。北は -Y なので northIndex ぶん負へ伸びる。
// worldstream と components の双方が同じ式を要るので、両者が import する leaf の consts を単一出典にする。
func BandOriginY(northIndex Chunk, chunkH Tile) AbsTileY {
	return AbsTileY(-int(northIndex.Tiles(chunkH)))
}

// FloorDiv は負の被除数でも床方向へ丸める整数除算。Go の / はゼロ方向へ丸めるため、
// 負側で境界が二重にならないよう床方向へ丸める。絶対 Y のチャンク割りとリージョン割りが
// 北側の負座標で連続するよう、query と overworld の双方がこの1関数を共有する。
func FloorDiv[T ~int](a, b T) T {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}
