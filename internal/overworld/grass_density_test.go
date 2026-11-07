package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestGrassPatchFactor_平均はおよそ0点9で茂った所と疎らな所に分かれる(t *testing.T) {
	t.Parallel()
	const size consts.Tile = 256
	sum := 0.0
	sparse, dense := 0, 0
	for y := range size {
		for x := range size {
			f := grassPatchFactor(1, consts.Coord[consts.Tile]{X: x, Y: y})
			sum += f
			switch {
			case f < 0.3:
				sparse++
			case f > 2:
				dense++
			}
		}
	}
	total := float64(size * size)

	assert.InDelta(t, 0.9, sum/total, 0.1)
	assert.Greater(t, float64(sparse)/total, 0.1)
	assert.Greater(t, float64(dense)/total, 0.05)
}

func TestGrassPatchFactor_隣り合うタイルで値が大きく跳ばない(t *testing.T) {
	t.Parallel()
	for x := consts.Tile(-50); x < 50; x++ {
		a := grassPatchFactor(1, consts.Coord[consts.Tile]{X: x, Y: 3})
		b := grassPatchFactor(1, consts.Coord[consts.Tile]{X: x + 1, Y: 3})
		assert.InDelta(t, a, b, 0.6, "x=%d", x)
	}
}
