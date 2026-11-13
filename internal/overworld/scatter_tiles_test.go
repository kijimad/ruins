package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
)

func TestScatterTiles(t *testing.T) {
	t.Parallel()
	all := func(consts.Coord[consts.Tile]) bool { return true }

	t.Run("同じseedは同じタイルを同じ順で返す", func(t *testing.T) {
		t.Parallel()
		want := scatterTiles(12, 12, all, 42, 6)
		assert.Equal(t, want, scatterTiles(12, 12, all, 42, 6))
	})
	t.Run("count枚で頭打ちになり0以下は空", func(t *testing.T) {
		t.Parallel()
		assert.Len(t, scatterTiles(12, 12, all, 1, 3), 3)
		assert.Empty(t, scatterTiles(12, 12, all, 1, 0))
	})
	t.Run("acceptを満たす外周より内側のタイルだけを全て返す", func(t *testing.T) {
		t.Parallel()
		got := scatterTiles(5, 4, func(v consts.Coord[consts.Tile]) bool { return v.X != 2 }, 7, 100)
		assert.ElementsMatch(t, []consts.Coord[consts.Tile]{{X: 1, Y: 1}, {X: 3, Y: 1}, {X: 1, Y: 2}, {X: 3, Y: 2}}, got)
	})
}
