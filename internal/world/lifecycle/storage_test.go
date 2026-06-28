package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
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
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

	// バックパック内にあることを確認
	assert.True(t, item.HasComponent(world.Components.LocationInBackpack))
	assert.False(t, item.HasComponent(world.Components.LocationInStorage))

	// 収納に移動
	require.NoError(t, MoveToStorage(world, item, storageEntity))

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
	items := query.GetStorageItems(world, storageEntity)
	assert.Empty(t, items)

	// アイテムを2つ収納に入れる
	item1, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	item2, err := SpawnFieldItem(world, "手榴弾", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	require.NoError(t, MoveToStorage(world, item1, storageEntity))
	require.NoError(t, MoveToStorage(world, item2, storageEntity))

	items = query.GetStorageItems(world, storageEntity)
	assert.Len(t, items, 2)
}

func TestGetStorageCurrentWeight(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// 空の収納の重量は0
	assert.Equal(t, 0.0, query.GetStorageCurrentWeight(world, storageEntity))

	// 重さを持つアイテムを収納に入れる
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, item, storageEntity))

	// WeightDirtySystemが行う処理を手動で実行
	query.UpdateWeightCapacity(world, storageEntity)

	weight := query.GetStorageCurrentWeight(world, storageEntity)
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
	assert.True(t, query.CanAddToStorage(world, storageEntity, item))
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
	assert.False(t, query.CanAddToStorage(world, storageEntity, item), "重量超過時は追加不可")

	// 元に戻す
	wc.Max = maxWeight
	assert.True(t, query.CanAddToStorage(world, storageEntity, item), "容量内なら追加可能")
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

	require.NoError(t, MoveToStorage(world, item, storageEntity))

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
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

	// マーカーをクリア
	playerEntity.RemoveComponent(world.Components.WeightDirty)

	// バックパック→収納に移動すると、元のOwner（Player）にもWeightDirtyが付与される
	require.NoError(t, MoveToStorage(world, item, storageEntity))

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
	require.NoError(t, MoveToStorage(world, item, storageEntity))

	// マーカーをクリア
	storageEntity.RemoveComponent(world.Components.WeightDirty)

	// 収納→バックパックに移動すると、元のOwner（Storage）にWeightDirtyが付与される
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

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
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

	// マーカーをクリア
	playerEntity.RemoveComponent(world.Components.WeightDirty)

	MoveToField(world, item, &playerEntity)

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
	require.NoError(t, MoveToStorage(world, item, storageEntity))
	assert.True(t, item.HasComponent(world.Components.LocationInStorage))

	require.NoError(t, MoveToBackpack(world, item, playerEntity))
	assert.True(t, item.HasComponent(world.Components.LocationInBackpack))
	assert.False(t, item.HasComponent(world.Components.LocationInStorage))
}

func TestMoveToBackpack_MergesStackableFromStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// バックパックに回復薬 x3 を配置
	_, err = SpawnBackpackItem(world, "回復薬", 3)
	require.NoError(t, err)

	// 収納に回復薬 x2 を配置
	storageItem, err := SpawnStorageItem(world, "回復薬", 2, storageEntity)
	require.NoError(t, err)

	// 収納からバックパックへ移動（統合されるべき）
	require.NoError(t, MoveToBackpack(world, storageItem, playerEntity))

	// バックパック内の回復薬エンティティは1つだけになっている
	var entityCount int
	var totalCount int
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.LocationInBackpack,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == "回復薬" {
			entityCount++
			stackable := world.Components.Stackable.Get(entity).(*gc.Stackable)
			totalCount += stackable.Count
		}
	}))

	assert.Equal(t, 1, entityCount, "回復薬は1つのエンティティに統合されるべき")
	assert.Equal(t, 5, totalCount, "合計個数は5個")
}

func TestMoveToBackpack_NoMergeForNonStackable(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// 非Stackableアイテムを2つバックパックに配置
	_, err = SpawnBackpackItem(world, "西洋鎧", 1)
	require.NoError(t, err)
	_, err = SpawnBackpackItem(world, "西洋鎧", 1)
	require.NoError(t, err)

	// 非Stackableアイテムは統合されず2つ存在する
	var entityCount int
	world.Manager.Join(
		world.Components.LocationInBackpack,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == "西洋鎧" {
			entityCount++
		}
	}))

	assert.Equal(t, 2, entityCount, "非Stackableアイテムは統合されない")
}

func TestMoveToStorage_MergesStackable(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	_, err = SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// 収納に回復薬 x3 を配置
	_, err = SpawnStorageItem(world, "回復薬", 3, storageEntity)
	require.NoError(t, err)

	// バックパックに回復薬 x1 を配置し、収納に移動する
	backpackItem, err := SpawnBackpackItem(world, "回復薬", 1)
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, backpackItem, storageEntity))

	// 収納内の回復薬エンティティは1つに統合されている
	var entityCount int
	var totalCount int
	for _, entity := range query.GetStorageItems(world, storageEntity) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == "回復薬" {
			entityCount++
			stackable := world.Components.Stackable.Get(entity).(*gc.Stackable)
			totalCount += stackable.Count
		}
	}

	assert.Equal(t, 1, entityCount, "回復薬は1つのエンティティに統合されるべき")
	assert.Equal(t, 4, totalCount, "合計個数は4個")
}

func TestMoveToStorage_DoesNotMergeAcrossStorages(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	const potion = "回復薬"

	storageA, err := SpawnProp(world, "木箱", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)
	storageB, err := SpawnProp(world, "木箱", consts.Tile(1), consts.Tile(0))
	require.NoError(t, err)

	_, err = SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	// 木箱Aに回復薬 x3、木箱Bに回復薬 x2
	_, err = SpawnStorageItem(world, potion, 3, storageA)
	require.NoError(t, err)
	_, err = SpawnStorageItem(world, potion, 2, storageB)
	require.NoError(t, err)

	// バックパックに回復薬 x1 を配置し、木箱Aに移動する
	backpackItem, err := SpawnBackpackItem(world, potion, 1)
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, backpackItem, storageA))

	// 木箱Aの回復薬は統合されて4個
	var countA int
	for _, entity := range query.GetStorageItems(world, storageA) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == potion {
			stackable := world.Components.Stackable.Get(entity).(*gc.Stackable)
			countA += stackable.Count
		}
	}
	assert.Equal(t, 4, countA, "木箱Aの回復薬は4個")

	// 木箱Bの回復薬は影響を受けず2個のまま
	var countB int
	for _, entity := range query.GetStorageItems(world, storageB) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == potion {
			stackable := world.Components.Stackable.Get(entity).(*gc.Stackable)
			countB += stackable.Count
		}
	}
	assert.Equal(t, 2, countB, "木箱Bの回復薬は影響を受けない")
}
