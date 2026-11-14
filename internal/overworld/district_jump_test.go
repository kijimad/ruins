package overworld

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/rawschema"
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

func TestNearestDistrictNorth_先の種類に重なって出ない区域しか無ければ見つからない(t *testing.T) {
	t.Parallel()

	// 先の種類が帯の全幅でリージョンの縦をほぼ埋めるので、後の種類の区域は必ず重なって出ない
	raws := rawschema.Raws{Districts: &[]rawschema.District{
		{Id: "wall", Spacing: 3, Separation: 0, Layout: rawschema.DistrictLayoutTown, Sizes: []rawschema.DistrictSize{{W: 2, H: 3, Weight: 1}}},
		{Id: "town", Spacing: 3, Separation: 0, Layout: rawschema.DistrictLayoutTown, Sizes: []rawschema.DistrictSize{{W: 1, H: 1, Weight: 1}}},
	}}
	ds := districtsFrom(raws)
	_, ok := nearestDistrictNorth(raws, ds[1], 1, 0, 2)
	assert.False(t, ok)
	_, ok = nearestDistrictNorth(raws, ds[0], 1, 0, 2)
	assert.True(t, ok, "先の種類は見つかる")
}
