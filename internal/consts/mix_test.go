package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMix64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		x    uint64
		want uint64
	}{
		{"入力0は0を返す", 0, 0},
		{"入力1の確定値", 1, 6238072747940578789},
		{"入力42の確定値", 42, 12058926934050108962},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Mix64(tt.x))
		})
	}
}

func TestMix2D(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		seed uint64
		x, y int
		want uint64
	}{
		{"原点は0を返す", 0, 0, 0, 0},
		{"座標(1,2)の確定値", 0, 1, 2, 10111300219251959524},
		{"座標(2,1)の確定値", 0, 2, 1, 17937901347196855910},
		{"負の座標でも動く", 0, -1, 0, 3703370420611038912},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Mix2D(tt.seed, tt.x, tt.y))
		})
	}
}

func TestMix2D_座標を転置すると異なる値になる(t *testing.T) {
	t.Parallel()

	// x と y に別の定数を掛けるので、転置した座標は同じ値にならない
	assert.NotEqual(t, Mix2D(0, 1, 2), Mix2D(0, 2, 1))
}

func TestMix2D_異なるseedは異なる値を返す(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, Mix2D(0, 2, 0), Mix2D(1, 2, 0))
}

func TestMix2D_隣接する座標同士は衝突しない(t *testing.T) {
	t.Parallel()

	const seed = uint64(7)
	seen := map[uint64]bool{}
	for x := -2; x <= 2; x++ {
		for y := -2; y <= 2; y++ {
			v := Mix2D(seed, x, y)
			assert.False(t, seen[v], "座標(%d,%d)が別座標と衝突した", x, y)
			seen[v] = true
		}
	}
}
