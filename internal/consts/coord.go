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
