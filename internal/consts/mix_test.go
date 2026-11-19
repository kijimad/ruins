package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMix2D_同じ入力なら同じ値を返す(t *testing.T) {
	t.Parallel()

	first := Mix2D(42, 5, 2)
	again := Mix2D(42, 5, 2)
	assert.Equal(t, first, again)
}

func TestMix2D_座標を転置すると異なる値になる(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, Mix2D(0, 1, 2), Mix2D(0, 2, 1))
}

func TestMix2D_異なるseedは異なる値を返す(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, Mix2D(0, 2, 0), Mix2D(1, 2, 0))
}

func TestMix2D_隣接する座標同士は衝突しない(t *testing.T) {
	t.Parallel()

	const seed = uint64(7)
	seen := map[uint64]Coord[int]{}
	for x := -8; x <= 8; x++ {
		for y := -8; y <= 8; y++ {
			v := Mix2D(seed, x, y)
			prev, dup := seen[v]
			assert.False(t, dup, "座標(%d,%d)が(%d,%d)と衝突した", x, y, prev.X, prev.Y)
			seen[v] = Coord[int]{X: x, Y: y}
		}
	}
}
