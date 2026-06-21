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

	// WeightDirtySystemが行う処理を手動で実行
	UpdateWeightCapacity(world, storageEntity)

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

	// WeightCapacityの最大重量を超えるまでアイテムを追加する
	wc := world.Components.WeightCapacity.Get(storageEntity).(*gc.WeightCapacity)
	maxWeight := wc.Max

	// 重量がmaxWeight+1kgのアイテムを作って追加不可を確認
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	// アイテムの重量を超過させるためにmaxWeightを0にする
	wc.Max = 0
	assert.False(t, CanAddToStorage(world, storageEntity, item), "重量超過時は追加不可")

	// 元に戻す
	wc.Max = maxWeight
	assert.True(t, CanAddToStorage(world, storageEntity, item), "容量内なら追加可能")
}

func TestMoveToStorage_SetsWeightDirtyOnStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	// マーカーを事前にクリア
	storageEntity.RemoveComponent(world.Components.WeightDirty)
	assert.False(t, storageEntity.HasComponent(world.Components.WeightDirty))

	MoveToStorage(world, item, storageEntity)

	assert.True(t, storageEntity.HasComponent(world.Components.WeightDirty), "MoveToStorageは収納エンティティにWeightDirtyを付与するべき")
}

func TestMoveToStorage_SetsWeightDirtyOnPreviousOwner(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err2 := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	MoveToBackpack(world, item, playerEntity)

	// マーカーをクリア
	playerEntity.RemoveComponent(world.Components.WeightDirty)

	// バックパック→収納に移動すると、元のOwner（Player）にもWeightDirtyが付与される
	MoveToStorage(world, item, storageEntity)

	assert.True(t, playerEntity.HasComponent(world.Components.WeightDirty), "移動元のOwnerにWeightDirtyが付与されるべき")
	assert.True(t, storageEntity.HasComponent(world.Components.WeightDirty), "移動先の収納にWeightDirtyが付与されるべき")
}

func TestMoveToBackpack_SetsWeightDirtyOnPreviousStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err2 := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	MoveToStorage(world, item, storageEntity)

	// マーカーをクリア
	storageEntity.RemoveComponent(world.Components.WeightDirty)

	// 収納→バックパックに移動すると、元のOwner（Storage）にWeightDirtyが付与される
	MoveToBackpack(world, item, playerEntity)

	assert.True(t, storageEntity.HasComponent(world.Components.WeightDirty), "移動元の収納にWeightDirtyが付与されるべき")
	assert.True(t, playerEntity.HasComponent(world.Components.WeightDirty), "移動先のPlayerにWeightDirtyが付与されるべき")
}

func TestMoveToField_SetsWeightDirtyOnPreviousOwner(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	playerEntity, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	MoveToBackpack(world, item, playerEntity)

	// マーカーをクリア
	playerEntity.RemoveComponent(world.Components.WeightDirty)

	MoveToField(world, item, playerEntity)

	assert.True(t, playerEntity.HasComponent(world.Components.WeightDirty), "MoveToFieldは元のOwnerにWeightDirtyを付与するべき")
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
