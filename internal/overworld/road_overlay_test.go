package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRoadOverlay_計画の道のチャンクに出入り口の方角を印す(t *testing.T) {
	t.Parallel()
	const runSeed uint64 = 12345
	const cols consts.Chunk = 9
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	area := MacroRange{OriginX: 0, OriginY: -30, Cols: cols, Rows: 60}

	want := map[consts.Coord[consts.Chunk]]RoadDir{}
	for y := area.OriginY; y < area.OriginY+area.Rows; y++ {
		for c, ports := range planAt(raws, runSeed, y, cols, 0).roads {
			if c.Y == y {
				want[c] = portDirs(ports)
			}
		}
	}
	require.NotEmpty(t, want, "前提: 表示範囲に道がある")
	assert.Equal(t, want, buildRoadOverlay(raws, runSeed, area, cols, 0))
}
