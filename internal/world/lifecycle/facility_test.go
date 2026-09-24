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
	query.InvalidateSpatialIndex(world)

	coord := consts.Coord[consts.Tile]{X: 11, Y: 10}
	prop, err := PlaceFacility(world, cube, item, coord)
	require.NoError(t, err)

	assert.False(t, world.ECS.Alive(item), "据付アイテムは消費される")
	assert.True(t, world.Components.Prop.Has(prop), "据えた設備は prop")
	assert.True(t, world.Components.LocationOnField.Has(prop), "設備はフィールドに現れる")
	assert.Equal(t, coord, world.Components.GridElement.Get(prop).Coord, "選んだマスに現れる")
	require.True(t, world.Components.DeployedFacility.Has(prop), "撤去で戻すアイテムを覚える")
	assert.Equal(t, "deployable_storage", world.Components.DeployedFacility.Get(prop).ItemID)
	assert.True(t, world.Components.WeightCapacity.Has(prop), "ストレージ設備は収納容量を持つ")
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
	assert.True(t, world.ECS.Alive(item), "据えられなければアイテムは消費しない")
}

func TestRemoveFacility(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)
	prop, err := PlaceFacility(world, cube, item, consts.Coord[consts.Tile]{X: 11, Y: 10})
	require.NoError(t, err)

	require.NoError(t, RemoveFacility(world, prop, player))
	assert.False(t, world.ECS.Alive(prop), "撤去で設備 prop は消える")
	restored := query.BackpackDeployables(world, player)
	require.Len(t, restored, 1, "撤去でアイテムがバックパックへ戻る")
	assert.Equal(t, "deployable_storage", world.Components.RawID.Get(restored[0]).ID)
}

func TestRemoveFacility_中身が残ると撤去しない(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)
	prop, err := PlaceFacility(world, cube, item, consts.Coord[consts.Tile]{X: 11, Y: 10})
	require.NoError(t, err)

	// 設備の収納に1つ入れる
	stored, err := spawnItemBase(world, "wooden_sword")
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, stored, prop))

	require.Error(t, RemoveFacility(world, prop, player), "中身が残る収納は撤去できない")
	assert.True(t, world.ECS.Alive(prop), "撤去しないので prop は残る")
}

func TestPlaceFacility_圧縮展開で設備を畳んで戻す(t *testing.T) {
	t.Parallel()
	world, cube := deployedCubeWithPlayer(t)

	item, err := SpawnBackpackItem(world, "deployable_storage", 1)
	require.NoError(t, err)
	query.InvalidateSpatialIndex(world)
	coord := consts.Coord[consts.Tile]{X: 11, Y: 10}
	prop, err := PlaceFacility(world, cube, item, coord)
	require.NoError(t, err)

	StowCube(world, cube)
	require.True(t, world.Components.LocationStowed.Has(prop), "設備は既存の往復で畳まれる")
	assert.True(t, world.Components.DeployedFacility.Has(prop), "畳んでも撤去情報は残る")

	require.True(t, DeployCube(world, cube))
	assert.True(t, world.Components.LocationOnField.Has(prop), "展開で設備が戻る")
	assert.Equal(t, coord, world.Components.GridElement.Get(prop).Coord, "相対位置を保って戻る")
	assert.True(t, world.Components.DeployedFacility.Has(prop), "戻しても撤去情報は残る")
}
