package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpawnRemains_跡の行から跡の段で描く跡を置く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	coord := consts.Coord[consts.Tile]{X: 3, Y: 4}

	e, err := SpawnRemains(world, "oil_stain", coord)
	require.NoError(t, err)

	assert.Equal(t, coord, world.Components.GridElement.Get(e).Coord)
	assert.Equal(t, gc.SpriteRender{SpriteSheetName: "field", SpriteKey: "oil_stain", Depth: gc.DepthNumRemains}, *world.Components.SpriteRender.Get(e))
	assert.Equal(t, i18n.Msgid("Oil Stain"), world.Components.Name.Get(e).Name)
	assert.True(t, world.Components.Remains.Has(e))
	got, ok := query.RemainsAt(world, coord)
	require.True(t, ok)
	assert.Equal(t, e, got)
}

func TestPlaceFacility_跡のマスに装着できる(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)
	coord := consts.Coord[consts.Tile]{X: 11, Y: 10}
	_, err := SpawnRemains(world, "wood_splinters", coord)
	require.NoError(t, err)
	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	placed, err := PlaceFacility(world, cube, item, coord)
	require.NoError(t, err)
	assert.Equal(t, coord, world.Components.GridElement.Get(placed).Coord)
}

func TestStowCube_跡は畳まずにその場に残り展開できる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, DeployCube(world, cube))
	coord := consts.Coord[consts.Tile]{X: 10, Y: 9}
	remains, err := SpawnRemains(world, "wood_splinters", coord)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	StowCube(world, cube)
	assert.Equal(t, coord, world.Components.GridElement.Get(remains).Coord, "跡はその場に残る")
	assert.False(t, world.Components.LocationStowed.Has(remains), "跡は積み荷にならない")

	assert.True(t, DeployCube(world, cube), "跡のある野営でも展開できる")
}

func TestSpawnRemains_形を行の候補から座標で選ぶ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	got := map[string]bool{}
	for i := range 100 {
		e, err := SpawnRemains(world, "red_blood_stain", consts.Coord[consts.Tile]{X: consts.Tile(i), Y: 0})
		require.NoError(t, err)
		got[world.Components.SpriteRender.Get(e).SpriteKey] = true
	}
	assert.Equal(t, map[string]bool{
		"blood_red_a": true, "blood_red_b": true, "blood_red_c": true, "blood_red_d": true, "blood_red_e": true,
	}, got, "100マスに置けば5つの形が全て選ばれる")

	a, err := SpawnRemains(world, "red_blood_stain", consts.Coord[consts.Tile]{X: 3, Y: 9})
	require.NoError(t, err)
	b, err := SpawnRemains(world, "red_blood_stain", consts.Coord[consts.Tile]{X: 3, Y: 9})
	require.NoError(t, err)
	assert.Equal(t, world.Components.SpriteRender.Get(a).SpriteKey, world.Components.SpriteRender.Get(b).SpriteKey, "同じマスなら同じ形")
}

func TestSpawnProp_地面の飾りの行は置物でない飾りとして置く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	grass, err := SpawnProp(world, "grass", 3, 4)
	require.NoError(t, err)

	assert.True(t, world.Components.Decor.Has(grass))
	assert.False(t, world.Components.Prop.Has(grass), "置物を数える処理に入らない")
	assert.False(t, world.Components.LocationOnField.Has(grass), "地面の物を数える処理に入らない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 3, Y: 4}, world.Components.GridElement.Get(grass).Coord)
}

func TestDeployCube_草のある野営でも展開でき畳んでも草はその場に残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	grass, err := SpawnProp(world, "grass", 10, 9)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	require.True(t, DeployCube(world, cube), "草のある野営でも展開できる")
	StowCube(world, cube)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 10, Y: 9}, world.Components.GridElement.Get(grass).Coord, "草はその場に残る")
	assert.False(t, world.Components.LocationStowed.Has(grass), "草は積み荷にならない")
}

func TestPlaceFacility_草のマスに装着できる(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)
	coord := consts.Coord[consts.Tile]{X: 11, Y: 10}
	_, err := SpawnProp(world, "grass", coord.X, coord.Y)
	require.NoError(t, err)
	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	placed, err := PlaceFacility(world, cube, item, coord)
	require.NoError(t, err)
	assert.Equal(t, coord, world.Components.GridElement.Get(placed).Coord)
}
