package geometry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		x0, y0   float64
		x1, y1   float64
		expected float64
	}{
		{"同じ座標", 0, 0, 0, 0, 0},
		{"水平", 0, 0, 3, 0, 3},
		{"垂直", 0, 0, 0, 4, 4},
		{"3-4-5三角形", 0, 0, 3, 4, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Distance(tt.x0, tt.y0, tt.x1, tt.y1)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestIsAdjacent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		x0, y0   int
		x1, y1   int
		expected bool
	}{
		{"同一座標", 5, 5, 5, 5, false},
		{"右隣", 5, 5, 6, 5, true},
		{"左隣", 5, 5, 4, 5, true},
		{"斜め", 5, 5, 6, 6, true},
		{"距離2", 5, 5, 7, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, IsAdjacent(tt.x0, tt.y0, tt.x1, tt.y1))
		})
	}
}

func TestBresenhamLine(t *testing.T) {
	t.Parallel()

	t.Run("隣接は空", func(t *testing.T) {
		t.Parallel()
		points := BresenhamLine(0, 0, 1, 0)
		assert.Empty(t, points)
	})

	t.Run("水平線は始点終点を含まない", func(t *testing.T) {
		t.Parallel()
		points := BresenhamLine(0, 0, 3, 0)
		assert.Len(t, points, 2)
		assert.Equal(t, 1, points[0].X)
		assert.Equal(t, 2, points[1].X)
	})

	t.Run("同じ座標なら空", func(t *testing.T) {
		t.Parallel()
		points := BresenhamLine(5, 5, 5, 5)
		assert.Empty(t, points)
	})
}

func TestAbs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5, Abs(5))
	assert.Equal(t, 5, Abs(-5))
	assert.Equal(t, 0, Abs(0))
}
