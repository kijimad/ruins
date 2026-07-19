package geometry

import (
	"math"

	"github.com/kijimaD/ruins/internal/consts"
)

// Distance は2点間のユークリッド距離を返す
func Distance(x0, y0, x1, y1 float64) float64 {
	dx := x0 - x1
	dy := y0 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

// IsAdjacent は2点が隣接しているかを判定する。チェビシェフ距離が1以下で同一座標でない場合にtrueを返す
func IsAdjacent[T consts.Numeric](a, b consts.Coord[T]) bool {
	dx := Abs(a.X - b.X)
	dy := Abs(a.Y - b.Y)
	if dx == 0 && dy == 0 {
		return false
	}
	return dx <= 1 && dy <= 1
}

// BresenhamLine はBresenhamアルゴリズムで2点間の座標列を返す。始点と終点は含まない。
// 純粋な整数グリッドのアルゴリズムなので、ドメイン単位でなく Coord[int] で扱う
func BresenhamLine(from, to consts.Coord[int]) []consts.Coord[int] {
	var points []consts.Coord[int]

	dx := Abs(to.X - from.X)
	dy := Abs(to.Y - from.Y)
	sx := 1
	if from.X > to.X {
		sx = -1
	}
	sy := 1
	if from.Y > to.Y {
		sy = -1
	}
	err := dx - dy

	cur := from
	for {
		if cur != from && cur != to {
			points = append(points, cur)
		}
		if cur == to {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			cur.X += sx
		}
		if e2 < dx {
			err += dx
			cur.Y += sy
		}
	}

	return points
}

// ChebyshevDistance は2点間のチェビシェフ距離を返す。距離はタイル数などの int
func ChebyshevDistance[T consts.Numeric](a, b consts.Coord[T]) int {
	dx := Abs(a.X - b.X)
	dy := Abs(a.Y - b.Y)
	if dx > dy {
		return int(dx)
	}
	return int(dy)
}

// Abs は絶対値を返す。int だけでなく consts.Tile/WorldPixel などの単位型でも使える
func Abs[T consts.Numeric](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
