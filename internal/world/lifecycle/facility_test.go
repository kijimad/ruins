package lifecycle

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deployedCubeWithPlayer は展開済みキューブとプレイヤーを用意する。設備の往復テストの共通準備。
func deployedCubeWithPlayer(t *testing.T) (w.World, ecs.Entity) {
	t.Helper()
	world := testutil.InitTestWorld(t)
	_, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := SpawnCube(world, consts.Coord[consts.Tile]{X: 10, Y: 10})
	require.NoError(t, err)
	require.True(t, DeployCube(world, cube))
	return world, cube
}

func TestPlaceFacility(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)

	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	require.True(t, world.Components.Deployable.Has(item), "据付アイテムは Deployable を持つ")
	require.True(t, world.Components.WeightCapacity.Has(item), "収納設備は容量を持つ")
	query.InvalidateSpatialIndex(world)

	coord := consts.Coord[consts.Tile]{X: 11, Y: 10}
	placed, err := PlaceFacility(world, cube, item, coord)
	require.NoError(t, err)

	assert.Equal(t, item, placed, "実体は削除せず同じものを据える")
	assert.True(t, world.ECS.Alive(item), "据えても実体は消えない")
	assert.True(t, world.Components.LocationOnField.Has(item), "設備はフィールドに現れる")
	assert.False(t, world.Components.LocationInBackpack.Has(item), "バックパックからは外れる")
	assert.Equal(t, coord, world.Components.GridElement.Get(item).Coord, "選んだマスに現れる")
	assert.False(t, query.IsPickable(item, world), "据えた設備は歩いて拾えない")
}

func TestPlaceFacility_塞がったマスには据えない(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)

	// 先に(11,10)へ prop を置いてマスを塞ぐ
	_, err := SpawnProp(world, "grass", 11, 10)
	require.NoError(t, err)
	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)

	_, err = PlaceFacility(world, cube, item, consts.Coord[consts.Tile]{X: 11, Y: 10})
	require.Error(t, err, "塞がったマスへは据えられない")
	assert.True(t, world.Components.LocationInBackpack.Has(item), "据えられなければバックパックに残る")
}

func TestRemoveFacility(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)
	placed, err := PlaceFacility(world, cube, item, consts.Coord[consts.Tile]{X: 11, Y: 10})
	require.NoError(t, err)

	require.NoError(t, RemoveFacility(world, placed, player))
	assert.True(t, world.ECS.Alive(item), "撤去しても実体は消えない")
	assert.True(t, world.Components.LocationInBackpack.Has(item), "撤去でバックパックへ戻る")
	assert.False(t, world.Components.LocationOnField.Has(item), "フィールドからは外れる")
}

func TestRemoveFacility_中身が残ると撤去しない(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)
	placed, err := PlaceFacility(world, cube, item, consts.Coord[consts.Tile]{X: 11, Y: 10})
	require.NoError(t, err)

	// 設備の収納に1つ入れる
	stored, err := spawnItemBase(world, "wooden_sword")
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, stored, placed))

	require.Error(t, RemoveFacility(world, placed, player), "中身が残る収納は撤去できない")
	assert.True(t, world.Components.LocationOnField.Has(placed), "撤去しないのでフィールドに残る")
}

func TestPlaceFacility_圧縮展開で設備を畳んで戻す(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)

	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)
	coord := consts.Coord[consts.Tile]{X: 11, Y: 10}
	placed, err := PlaceFacility(world, cube, item, coord)
	require.NoError(t, err)

	StowCube(world, cube)
	require.True(t, world.Components.LocationStowed.Has(placed), "設備は既存の往復で畳まれる")
	assert.True(t, world.Components.Deployable.Has(placed), "畳んでも据付情報は残る")

	require.True(t, DeployCube(world, cube))
	assert.True(t, world.Components.LocationOnField.Has(placed), "展開で設備が戻る")
	assert.Equal(t, coord, world.Components.GridElement.Get(placed).Coord, "相対位置を保って戻る")
}
