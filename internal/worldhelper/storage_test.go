package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveToStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 収納propを生成
	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// アイテムを生成してバックパックに配置
	playerEntity, err2 := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	MoveToBackpack(world, item, playerEntity)

	// バックパック内にあることを確認
	assert.True(t, item.HasComponent(world.Components.LocationInBackpack))
	assert.False(t, item.HasComponent(world.Components.LocationInStorage))

	// 収納に移動
	MoveToStorage(world, item, storageEntity)

	// 収納内にあることを確認（排他制御）
	assert.True(t, item.HasComponent(world.Components.LocationInStorage))
	assert.False(t, item.HasComponent(world.Components.LocationInBackpack))
	assert.False(t, item.HasComponent(world.Components.LocationOnField))

	loc := world.Components.LocationInStorage.Get(item).(*gc.LocationInStorage)
	assert.Equal(t, storageEntity, loc.Owner)
}

func TestGetStorageItems(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// 空の収納にはアイテムがない
	items := GetStorageItems(world, storageEntity)
	assert.Empty(t, items)

	// アイテムを2つ収納に入れる
	item1, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	item2, err := SpawnFieldItem(world, "手榴弾", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	MoveToStorage(world, item1, storageEntity)
	MoveToStorage(world, item2, storageEntity)

	items = GetStorageItems(world, storageEntity)
	assert.Len(t, items, 2)
}

func TestGetStorageCurrentWeight(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// 空の収納の重量は0
	assert.Equal(t, 0.0, GetStorageCurrentWeight(world, storageEntity))

	// 重さを持つアイテムを収納に入れる
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	MoveToStorage(world, item, storageEntity)

	weight := GetStorageCurrentWeight(world, storageEntity)
	assert.Greater(t, weight, 0.0, "アイテムの重量が反映されるべき")
}

func TestCanAddToStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// 空の収納にはアイテムを追加できる
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	assert.True(t, CanAddToStorage(world, storageEntity, item))
}

func TestCanAddToStorage_OverWeight(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// Storageの最大重量を超えるまでアイテムを追加する
	storageComp := world.Components.Storage.Get(storageEntity).(*gc.Storage)
	maxWeight := storageComp.MaxWeight

	// 重量がmaxWeight+1kgのアイテムを作って追加不可を確認
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	// アイテムの重量を超過させるためにmaxWeightを0にする
	storageComp.MaxWeight = 0
	assert.False(t, CanAddToStorage(world, storageEntity, item), "重量超過時は追加不可")

	// 元に戻す
	storageComp.MaxWeight = maxWeight
	assert.True(t, CanAddToStorage(world, storageEntity, item), "容量内なら追加可能")
}

func TestMoveToStorage_ThenBackToBackpack(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err2 := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	// 収納に入れて、バックパックに戻す
	MoveToStorage(world, item, storageEntity)
	assert.True(t, item.HasComponent(world.Components.LocationInStorage))

	MoveToBackpack(world, item, playerEntity)
	assert.True(t, item.HasComponent(world.Components.LocationInBackpack))
	assert.False(t, item.HasComponent(world.Components.LocationInStorage))
}
