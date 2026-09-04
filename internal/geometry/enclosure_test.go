package geometry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnclosedRegion(t *testing.T) {
	t.Parallel()

	// 5x5 グリッドの外周を壁にし、中央 3x3 を空けた囲い
	walledBox := func(x, y int) bool {
		return x == 0 || x == 4 || y == 0 || y == 4
	}

	t.Run("壁に囲まれた領域は外周に達しない", func(t *testing.T) {
		t.Parallel()
		cells, touchesEdge := EnclosedRegion(5, 5, walledBox, 2, 2)
		assert.False(t, touchesEdge)
		assert.Len(t, cells, 9, "中央の 3x3 がすべて繋がる")
	})

	t.Run("壁の無いグリッドは外周に達する", func(t *testing.T) {
		t.Parallel()
		open := func(_, _ int) bool { return false }
		cells, touchesEdge := EnclosedRegion(5, 5, open, 2, 2)
		assert.True(t, touchesEdge)
		assert.Len(t, cells, 25)
	})

	t.Run("壁に開いた通路で外周と繋がると外周に達する", func(t *testing.T) {
		t.Parallel()
		// 囲いの壁 (2,0) に穴を開ける
		holed := func(x, y int) bool {
			if x == 2 && y == 0 {
				return false
			}
			return walledBox(x, y)
		}
		cells, touchesEdge := EnclosedRegion(5, 5, holed, 2, 2)
		assert.True(t, touchesEdge)
		assert.Len(t, cells, 10, "中央 3x3 と穴の 1 セル")
	})

	t.Run("seedが塞がっていると空集合を返す", func(t *testing.T) {
		t.Parallel()
		cells, touchesEdge := EnclosedRegion(5, 5, walledBox, 0, 0)
		assert.False(t, touchesEdge)
		assert.Empty(t, cells)
	})

	t.Run("seedが範囲外だと空集合を返す", func(t *testing.T) {
		t.Parallel()
		cells, touchesEdge := EnclosedRegion(5, 5, walledBox, -1, 2)
		assert.False(t, touchesEdge)
		assert.Empty(t, cells)
	})
}
