package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirection_GetDelta(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		dir   Direction
		wantX int
		wantY int
	}{
		{"None", DirectionNone, 0, 0},
		{"Up", DirectionUp, 0, -1},
		{"Down", DirectionDown, 0, 1},
		{"Left", DirectionLeft, -1, 0},
		{"Right", DirectionRight, 1, 0},
		{"UpLeft", DirectionUpLeft, -1, -1},
		{"UpRight", DirectionUpRight, 1, -1},
		{"DownLeft", DirectionDownLeft, -1, 1},
		{"DownRight", DirectionDownRight, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			x, y := tt.dir.GetDelta()
			assert.Equal(t, tt.wantX, x)
			assert.Equal(t, tt.wantY, y)
		})
	}
}

func TestRect_Center(t *testing.T) {
	t.Parallel()

	t.Run("偶数サイズ", func(t *testing.T) {
		t.Parallel()
		r := &Rect{X1: 0, X2: 10, Y1: 0, Y2: 10}
		x, y := r.Center()
		assert.Equal(t, 5, int(x))
		assert.Equal(t, 5, int(y))
	})

	t.Run("奇数サイズ", func(t *testing.T) {
		t.Parallel()
		r := &Rect{X1: 1, X2: 4, Y1: 2, Y2: 7}
		x, y := r.Center()
		assert.Equal(t, 2, int(x))
		assert.Equal(t, 4, int(y))
	})

	t.Run("ゼロサイズ", func(t *testing.T) {
		t.Parallel()
		r := &Rect{X1: 5, X2: 5, Y1: 3, Y2: 3}
		x, y := r.Center()
		assert.Equal(t, 5, int(x))
		assert.Equal(t, 3, int(y))
	})
}
