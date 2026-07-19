package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
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
		r := &Rect{Min: consts.Coord[consts.Tile]{X: 0, Y: 0}, Max: consts.Coord[consts.Tile]{X: 10, Y: 10}}
		x, y := r.Center()
		assert.Equal(t, 5, int(x))
		assert.Equal(t, 5, int(y))
	})

	t.Run("奇数サイズ", func(t *testing.T) {
		t.Parallel()
		r := &Rect{Min: consts.Coord[consts.Tile]{X: 1, Y: 2}, Max: consts.Coord[consts.Tile]{X: 4, Y: 7}}
		x, y := r.Center()
		assert.Equal(t, 2, int(x))
		assert.Equal(t, 4, int(y))
	})

	t.Run("ゼロサイズ", func(t *testing.T) {
		t.Parallel()
		r := &Rect{Min: consts.Coord[consts.Tile]{X: 5, Y: 3}, Max: consts.Coord[consts.Tile]{X: 5, Y: 3}}
		x, y := r.Center()
		assert.Equal(t, 5, int(x))
		assert.Equal(t, 3, int(y))
	})
}
