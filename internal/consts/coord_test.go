package consts_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestCoord_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    consts.Coord[int]
		want string
	}{
		{"正の座標", consts.Coord[int]{X: 1, Y: 2}, "(1,2)"},
		{"ゼロ", consts.Coord[int]{X: 0, Y: 0}, "(0,0)"},
		{"負の座標", consts.Coord[int]{X: -3, Y: -4}, "(-3,-4)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.c.String())
		})
	}
}

func TestCoord_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    consts.Coord[int]
		b    consts.Coord[int]
		want consts.Coord[int]
	}{
		{"正同士", consts.Coord[int]{X: 1, Y: 2}, consts.Coord[int]{X: 3, Y: 4}, consts.Coord[int]{X: 4, Y: 6}},
		{"ゼロを足す", consts.Coord[int]{X: 5, Y: 5}, consts.Coord[int]{X: 0, Y: 0}, consts.Coord[int]{X: 5, Y: 5}},
		{"負を足すと減る", consts.Coord[int]{X: 5, Y: 5}, consts.Coord[int]{X: -2, Y: -3}, consts.Coord[int]{X: 3, Y: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.a.Add(tt.b))
		})
	}
}

func TestCoord_Sub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    consts.Coord[int]
		b    consts.Coord[int]
		want consts.Coord[int]
	}{
		{"正同士", consts.Coord[int]{X: 5, Y: 6}, consts.Coord[int]{X: 3, Y: 4}, consts.Coord[int]{X: 2, Y: 2}},
		{"ゼロを引く", consts.Coord[int]{X: 5, Y: 5}, consts.Coord[int]{X: 0, Y: 0}, consts.Coord[int]{X: 5, Y: 5}},
		{"引いた結果が負になる", consts.Coord[int]{X: 1, Y: 1}, consts.Coord[int]{X: 3, Y: 4}, consts.Coord[int]{X: -2, Y: -3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.a.Sub(tt.b))
		})
	}
}

func TestTileCenterToWorld(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		grid consts.Coord[consts.Tile]
		want consts.Coord[consts.WorldPixel]
	}{
		// タイル中心へ半タイル分ずらした位置になる
		{"原点タイル", consts.Coord[consts.Tile]{X: 0, Y: 0}, consts.Coord[consts.WorldPixel]{X: 16, Y: 16}},
		{"1マス目", consts.Coord[consts.Tile]{X: 1, Y: 1}, consts.Coord[consts.WorldPixel]{X: 48, Y: 48}},
		{"XとYが異なる", consts.Coord[consts.Tile]{X: 2, Y: 0}, consts.Coord[consts.WorldPixel]{X: 80, Y: 16}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, consts.TileCenterToWorld(tt.grid))
		})
	}
}

func TestWorldToScreen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		world     consts.Coord[consts.WorldPixel]
		cameraPos consts.Coord[consts.WorldPixel]
		scale     float64
		screenW   int
		screenH   int
		want      consts.Coord[consts.ScreenPixel]
	}{
		{
			name:      "カメラ原点で等倍なら画面中央基準",
			world:     consts.Coord[consts.WorldPixel]{X: 0, Y: 0},
			cameraPos: consts.Coord[consts.WorldPixel]{X: 0, Y: 0},
			scale:     1,
			screenW:   960,
			screenH:   720,
			want:      consts.Coord[consts.ScreenPixel]{X: 480, Y: 360},
		},
		{
			name:      "カメラがずれると相対位置がずれる",
			world:     consts.Coord[consts.WorldPixel]{X: 100, Y: 0},
			cameraPos: consts.Coord[consts.WorldPixel]{X: 50, Y: 0},
			scale:     1,
			screenW:   960,
			screenH:   720,
			want:      consts.Coord[consts.ScreenPixel]{X: 530, Y: 360},
		},
		{
			name:      "スケール2倍で差分が拡大する",
			world:     consts.Coord[consts.WorldPixel]{X: 100, Y: 0},
			cameraPos: consts.Coord[consts.WorldPixel]{X: 0, Y: 0},
			scale:     2,
			screenW:   960,
			screenH:   720,
			want:      consts.Coord[consts.ScreenPixel]{X: 680, Y: 360},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := consts.WorldToScreen(tt.world, tt.cameraPos, tt.scale, tt.screenW, tt.screenH)
			assert.Equal(t, tt.want, got)
		})
	}
}
