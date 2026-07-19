package geometry

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestChebyshevDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		x0, y0   int
		x1, y1   int
		expected int
	}{
		{"同一座標", 0, 0, 0, 0, 0},
		{"水平のみ", 0, 0, 5, 0, 5},
		{"垂直のみ", 0, 0, 0, 3, 3},
		{"対角線(dx=dy)", 0, 0, 4, 4, 4},
		{"dx>dy", 0, 0, 5, 3, 5},
		{"dx<dy", 0, 0, 2, 7, 7},
		{"負の座標", -3, -2, 1, 4, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := consts.Coord[int]{X: tt.x0, Y: tt.y0}
			b := consts.Coord[int]{X: tt.x1, Y: tt.y1}
			assert.Equal(t, tt.expected, ChebyshevDistance(a, b))
		})
	}
}

// BresenhamLine は始点・終点を除く中間点のみを返す
func TestBresenhamLine_Vertical(t *testing.T) {
	t.Parallel()

	// (0,0)→(0,3): 中間点は (0,1) と (0,2) の2点
	points := BresenhamLine(consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 0, Y: 3})
	assert.Len(t, points, 2)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 0, Y: 1}, points[0])
	assert.Equal(t, consts.Coord[consts.Tile]{X: 0, Y: 2}, points[1])
}

func TestBresenhamLine_Diagonal(t *testing.T) {
	t.Parallel()

	// (0,0)→(3,3): 中間点は (1,1) と (2,2) の2点
	points := BresenhamLine(consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.Tile]{X: 3, Y: 3})
	assert.Len(t, points, 2)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 1, Y: 1}, points[0])
	assert.Equal(t, consts.Coord[consts.Tile]{X: 2, Y: 2}, points[1])
}

func TestBresenhamLine_Reverse(t *testing.T) {
	t.Parallel()

	// (3,0)→(0,0): 中間点は (2,0) と (1,0) の2点
	points := BresenhamLine(consts.Coord[consts.Tile]{X: 3, Y: 0}, consts.Coord[consts.Tile]{X: 0, Y: 0})
	assert.Len(t, points, 2)
	assert.Equal(t, consts.Tile(2), points[0].X)
	assert.Equal(t, consts.Tile(1), points[1].X)
}
