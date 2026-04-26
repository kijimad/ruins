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
func IsAdjacent(x0, y0, x1, y1 int) bool {
	dx := Abs(x0 - x1)
	dy := Abs(y0 - y1)
	if dx == 0 && dy == 0 {
		return false
	}
	return dx <= 1 && dy <= 1
}

// BresenhamLine はBresenhamアルゴリズムで2点間の座標列を返す。始点と終点は含まない
func BresenhamLine(x0, y0, x1, y1 int) []consts.Coord[int] {
	var points []consts.Coord[int]

	dx := Abs(x1 - x0)
	dy := Abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	x, y := x0, y0
	for {
		if (x != x0 || y != y0) && (x != x1 || y != y1) {
			points = append(points, consts.Coord[int]{X: x, Y: y})
		}
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}

	return points
}

// Abs は整数の絶対値を返す
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
