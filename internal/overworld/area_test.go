package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAreaAt_区域だけが区域ごとのキーを持つ(t *testing.T) {
	t.Parallel()

	raws := testutil.InitTestWorld(t).Resources.RawMaster
	const runSeed uint64 = 12345
	const cols consts.Chunk = 7
	regions := map[string]map[consts.Coord[consts.Chunk]]bool{}
	var wasteland, urban int
	for y := consts.Chunk(-80); y < 10; y++ {
		for x := range cols {
			c := consts.Coord[consts.Chunk]{X: x, Y: y}
			area := AreaAt(raws, runSeed, c, cols, 0)
			switch chunkTypeAt(raws, runSeed, c, cols) {
			case chunkUrban:
				urban++
				r, _ := districtAt(raws, runSeed, c, cols)
				if regions[area.Key] == nil {
					regions[area.Key] = map[consts.Coord[consts.Chunk]]bool{}
				}
				regions[area.Key][r.anchor] = true
				assert.Equal(t, i18n.Msgid(r.kind.Name), area.Name)
				assert.NotEmpty(t, area.Building, "市街地は建物の名前を持つ")
			case chunkWasteland:
				wasteland++
				assert.Equal(t, Area{Name: "Wasteland"}, area)
			case chunkSettlement, chunkLandmark, chunkDungeonEntrance:
				assert.Empty(t, area.Key, "小さな場所はスプラッシュを出さない")
				assert.NotEmpty(t, area.Name)
				assert.Empty(t, area.Building)
			}
		}
	}
	require.Positive(t, urban, "区域のチャンクがある")
	require.Positive(t, wasteland, "荒れ地のチャンクがある")
	for key, anchors := range regions {
		assert.Len(t, anchors, 1, "%s は1つの区域だけを指す", key)
	}
}
