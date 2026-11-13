package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriver_JumpToDistrict_北で一番近い区域の南の門の前へ行く(t *testing.T) {
	t.Parallel()

	const cols, rows consts.Chunk = 7, 9
	world := testutil.InitTestWorld(t)
	s := NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, testChunkW, testChunkH, cols, rows), &NewGameParams{RunSeed: 5})
	require.NoError(t, s.Start(world))
	raws := world.Resources.RawMaster
	sb := query.GetSeamlessBand(world)
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	for range 2 {
		before := sb.AbsChunkRow(world.Components.GridElement.Get(player).Y)
		require.NoError(t, s.JumpToDistrict(world, "lab_campus"))

		pg := world.Components.GridElement.Get(player)
		row := sb.AbsChunkRow(pg.Y)
		col := consts.Chunk(pg.X / testChunkW)
		assert.Less(t, row, before, "北へ進む")
		r, ok := districtAt(raws, sb.RunSeed, consts.Coord[consts.Chunk]{X: col, Y: row - 1}, cols)
		require.True(t, ok, "プレイヤーのすぐ北が区域")
		assert.Equal(t, consts.Coord[consts.Chunk]{X: r.anchor.X + r.w/2, Y: r.anchor.Y + r.h}, consts.Coord[consts.Chunk]{X: col, Y: row}, "区域の南の端の真ん中の列のすぐ外")
		assert.Equal(t, rows/2, consts.Chunk(pg.Y/testChunkH), "帯の中央の行に留まる")
		assert.Len(t, query.CubeTiles(world), 1, "キューブを連れていく")
	}

	require.ErrorIs(t, s.JumpToDistrict(world, "no_such"), errDistrictNotFound)
}
