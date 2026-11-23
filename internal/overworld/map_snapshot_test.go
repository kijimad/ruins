package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapSnapshots_最後の段階は同じ範囲の俯瞰図と記号と道が一致する(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	road, ok := roadRow(raws)
	require.True(t, ok)
	rows := consts.Chunk(road.SpanRows)
	for seed := range uint64(5) {
		snaps := MapSnapshots(raws, seed, 0, planTestCols, planTestSpawn)
		require.NotEmpty(t, snaps)
		last := snaps[len(snaps)-1]
		require.Equal(t, "landmarks", last.Label)
		want := BuildMacroView(raws, seed, 0, planTestSpawn, 1, 1, MacroRange{Cols: planTestCols, Rows: rows}, consts.Coord[consts.Tile]{}, false, nil, nil)
		require.Len(t, last.View.Cells, len(want.Cells))
		for y := range want.Cells {
			for x := range want.Cells[y] {
				got, exp := last.View.Cells[y][x], want.Cells[y][x]
				assert.Equal(t, string(exp.Glyph), string(got.Glyph), "seed=%d (%d,%d) の記号", seed, x, y)
				assert.Equal(t, exp.Color, got.Color, "seed=%d (%d,%d) の色", seed, x, y)
				assert.Equal(t, exp.Road, got.Road, "seed=%d (%d,%d) の道", seed, x, y)
			}
		}
	}
}
