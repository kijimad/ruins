package overworld_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/stretchr/testify/assert"
)

func TestPlacementAt(t *testing.T) {
	t.Parallel()

	p := overworld.Placement{Spacing: 8, Separation: 2, Salt: 0x99}

	winners := func(rows consts.Chunk, regions int) []worldstream.ChunkCoord {
		var got []worldstream.ChunkCoord
		for x := consts.Chunk(0); x < p.Spacing*consts.Chunk(regions); x++ {
			for y := range rows {
				c := worldstream.ChunkCoord{X: x, Y: y}
				if p.At(42, c, rows) {
					got = append(got, c)
				}
			}
		}
		return got
	}

	t.Run("各リージョンにちょうど1つ当選する", func(t *testing.T) {
		t.Parallel()
		got := winners(3, 10)
		assert.Len(t, got, 10, "10リージョンで当選は10チャンク")
		for i, c := range got {
			assert.Equal(t, consts.Chunk(i), c.X/p.Spacing, "当選 %v は自リージョン内にある", c)
		}
	})

	t.Run("隣接する当選どうしは Separation より離れる", func(t *testing.T) {
		t.Parallel()
		got := winners(3, 20)
		for i := 1; i < len(got); i++ {
			gap := int(got[i].X - got[i-1].X)
			assert.Greater(t, gap, int(p.Separation), "当選 %v と %v の X 間隔", got[i-1], got[i])
		}
	})

	t.Run("行数1の帯でも当選が帯内に出る", func(t *testing.T) {
		t.Parallel()
		got := winners(1, 10)
		assert.Len(t, got, 10, "行数1でも各リージョンに1つ当選する")
		for _, c := range got {
			assert.Equal(t, consts.Chunk(0), c.Y, "行数1では当選行は0になる")
		}
	})

}
