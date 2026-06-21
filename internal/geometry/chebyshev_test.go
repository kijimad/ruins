package geometry

import (
	"testing"

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
			assert.Equal(t, tt.expected, ChebyshevDistance(tt.x0, tt.y0, tt.x1, tt.y1))
		})
	}
}

// BresenhamLine は始点・終点を除く中間点のみを返す
func TestBresenhamLine_Vertical(t *testing.T) {
	t.Parallel()

	// (0,0)→(0,3): 中間点は (0,1) と (0,2) の2点
	points := BresenhamLine(0, 0, 0, 3)
	assert.Len(t, points, 2)
	assert.Equal(t, 0, points[0].X)
	assert.Equal(t, 1, points[0].Y)
	assert.Equal(t, 0, points[1].X)
	assert.Equal(t, 2, points[1].Y)
}

func TestBresenhamLine_Diagonal(t *testing.T) {
	t.Parallel()

	// (0,0)→(3,3): 中間点は (1,1) と (2,2) の2点
	points := BresenhamLine(0, 0, 3, 3)
	assert.Len(t, points, 2)
	assert.Equal(t, 1, points[0].X)
	assert.Equal(t, 1, points[0].Y)
	assert.Equal(t, 2, points[1].X)
	assert.Equal(t, 2, points[1].Y)
}

func TestBresenhamLine_Reverse(t *testing.T) {
	t.Parallel()

	// (3,0)→(0,0): 中間点は (2,0) と (1,0) の2点
	points := BresenhamLine(3, 0, 0, 0)
	assert.Len(t, points, 2)
	assert.Equal(t, 2, points[0].X)
	assert.Equal(t, 1, points[1].X)
}
