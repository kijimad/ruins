package components

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistanceMap(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	wall := func(m *DistanceMap, cs ...consts.Coord[consts.Tile]) {
		for _, c := range cs {
			m.Cost[m.Index(c)] = 0
			m.Blocked[m.Index(c)] = true
		}
	}

	t.Run("空いた通路は1マスごとに1遠くなる", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(4, 1)
		m.Spread([]consts.Coord[consts.Tile]{at(0, 0)})
		got := make([]int32, 0, 4)
		for x := range 4 {
			d, ok := m.Dist(at(x, 0))
			require.True(t, ok)
			got = append(got, d)
		}
		assert.Equal(t, []int32{0, 1, 2, 3}, got)
	})

	t.Run("縦横の隣がどちらも塞がった角は斜めに抜けない", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(2, 2)
		wall(m, at(1, 0), at(0, 1))
		m.Spread([]consts.Coord[consts.Tile]{at(0, 0)})
		_, ok := m.Dist(at(1, 1))
		assert.False(t, ok)
	})

	t.Run("Steps は角の向こうの的へ斜めに抜けない", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(3, 3)
		wall(m, at(1, 0), at(0, 1))
		m.Spread([]consts.Coord[consts.Tile]{at(0, 0), at(2, 2)})
		assert.Equal(t, []consts.Coord[consts.Tile]{at(2, 2)}, m.Steps(at(1, 1), nil), "角の向こうの (0,0) でなく、斜めに抜けられる (2,2) へ進む")
	})

	t.Run("Steps は自分のマスの重さを数えずに比べ、遠ざかる隣を入れない", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(5, 3)
		m.Cost[m.Index(at(2, 1))] = DistanceMapObstacleCost
		m.Spread([]consts.Coord[consts.Tile]{at(0, 1)})
		assert.Equal(t, []consts.Coord[consts.Tile]{at(1, 1), at(1, 0), at(1, 2)}, m.Steps(at(2, 1), nil), "横と後ろの列は、重さ5の自分のマスより近くても下る隣に入れない")
	})

	t.Run("Steps は細い通路では一番小さい隣の1つだけを返す", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(5, 1)
		m.Cost[m.Index(at(1, 0))] = DistanceMapObstacleCost
		m.Spread([]consts.Coord[consts.Tile]{at(0, 0)})
		assert.Equal(t, []consts.Coord[consts.Tile]{at(1, 0)}, m.Steps(at(2, 0), nil), "塞がれていても替わりの隣は無く、呼ぶ側は待つ")
	})

	t.Run("的が無ければどのマスにも届かない", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(3, 1)
		m.Spread(nil)
		_, ok := m.Dist(at(1, 0))
		assert.False(t, ok)
	})

	t.Run("Steps は距離が一番小さい隣を全て向きの順に返す", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(5, 5)
		m.Spread([]consts.Coord[consts.Tile]{at(0, 0)})
		assert.Equal(t, []consts.Coord[consts.Tile]{at(2, 1), at(2, 0), at(2, 2)}, m.Steps(at(3, 1), nil))
		assert.Empty(t, m.Steps(at(0, 0), nil), "的の上では進む先が無い")
	})

	t.Run("Reset は前の地図の距離と重みを消す", func(t *testing.T) {
		t.Parallel()
		m := NewDistanceMap()
		m.Reset(3, 1)
		wall(m, at(1, 0))
		m.Spread([]consts.Coord[consts.Tile]{at(0, 0)})
		m.Reset(3, 1)
		_, ok := m.Dist(at(0, 0))
		assert.False(t, ok, "広げる前は届かない")
		m.Spread([]consts.Coord[consts.Tile]{at(0, 0)})
		d, ok := m.Dist(at(2, 0))
		require.True(t, ok)
		assert.Equal(t, int32(2), d, "前の壁は残らない")
	})
}
