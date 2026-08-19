package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveToStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 収納propを生成
	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// アイテムを生成してバックパックに配置
	playerEntity, err2 := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

	// バックパック内にあることを確認
	assert.True(t, world.Components.LocationInBackpack.Has(item))
	assert.False(t, world.Components.LocationInStorage.Has(item))

	// 収納に移動
	require.NoError(t, MoveToStorage(world, item, storageEntity))

	// 収納内にあることを確認（排他制御）
	assert.True(t, world.Components.LocationInStorage.Has(item))
	assert.False(t, world.Components.LocationInBackpack.Has(item))
	assert.False(t, world.Components.LocationOnField.Has(item))

	loc := world.Components.LocationInStorage.Get(item)
	assert.Equal(t, storageEntity, loc.Owner)
}

func TestGetStorageItems(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// 空の収納にはアイテムがない
	items := query.GetStorageItems(world, storageEntity)
	assert.Empty(t, items)

	// アイテムを2つ収納に入れる
	item1, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	item2, err := SpawnFieldItem(world, "grenade", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	require.NoError(t, MoveToStorage(world, item1, storageEntity))
	require.NoError(t, MoveToStorage(world, item2, storageEntity))

	items = query.GetStorageItems(world, storageEntity)
	assert.Len(t, items, 2)
}

func TestGetStorageCurrentWeight(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// 空の収納の重量は0
	assert.Equal(t, consts.Milligram(0), query.GetStorageCurrentWeight(world, storageEntity))

	// 重さを持つアイテムを収納に入れる
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, item, storageEntity))

	// WeightDirtySystemが行う処理を手動で実行
	query.UpdateWeightCapacity(world, storageEntity)

	weight := query.GetStorageCurrentWeight(world, storageEntity)
	assert.Greater(t, weight, consts.Milligram(0), "アイテムの重量が反映されるべき")
}

func TestCanAddToStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// 空の収納にはアイテムを追加できる
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	assert.True(t, query.CanAddToStorage(world, storageEntity, item))
}

func TestCanAddToStorage_OverWeight(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	// WeightCapacityの最大重量を超えるまでアイテムを追加する
	wc := world.Components.WeightCapacity.Get(storageEntity)
	maxWeight := wc.Max

	// 重量がmaxWeight+1kgのアイテムを作って追加不可を確認
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
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

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	// マーカーを事前にクリア
	if world.Components.WeightDirty.Has(storageEntity) {
		world.Components.WeightDirty.Remove(storageEntity)
	}
	assert.False(t, world.Components.WeightDirty.Has(storageEntity))

	require.NoError(t, MoveToStorage(world, item, storageEntity))

	assert.True(t, world.Components.WeightDirty.Has(storageEntity), "MoveToStorageは収納エンティティにWeightDirtyを付与するべき")
}

func TestMoveToStorage_SetsWeightDirtyOnPreviousOwner(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err2 := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

	// マーカーをクリア
	world.Components.WeightDirty.Remove(playerEntity)

	// バックパック→収納に移動すると、元のOwner（Player）にもWeightDirtyが付与される
	require.NoError(t, MoveToStorage(world, item, storageEntity))

	assert.True(t, world.Components.WeightDirty.Has(playerEntity), "移動元のOwnerにWeightDirtyが付与されるべき")
	assert.True(t, world.Components.WeightDirty.Has(storageEntity), "移動先の収納にWeightDirtyが付与されるべき")
}

func TestMoveToBackpack_SetsWeightDirtyOnPreviousStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err2 := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, item, storageEntity))

	// マーカーをクリア
	world.Components.WeightDirty.Remove(storageEntity)

	// 収納→バックパックに移動すると、元のOwner（Storage）にWeightDirtyが付与される
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

	assert.True(t, world.Components.WeightDirty.Has(storageEntity), "移動元の収納にWeightDirtyが付与されるべき")
	assert.True(t, world.Components.WeightDirty.Has(playerEntity), "移動先のPlayerにWeightDirtyが付与されるべき")
}

func TestMoveToField_SetsWeightDirtyOnPreviousOwner(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	playerEntity, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)
	require.NoError(t, MoveToBackpack(world, item, playerEntity))

	// マーカーをクリア
	world.Components.WeightDirty.Remove(playerEntity)

	MoveToField(world, item, &playerEntity)

	assert.True(t, world.Components.WeightDirty.Has(playerEntity), "MoveToFieldは元のOwnerにWeightDirtyを付与するべき")
}

func TestMoveToStorage_ThenBackToBackpack(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err2 := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err2)
	item, err := SpawnFieldItem(world, "healing_potion", consts.Tile(0), consts.Tile(0), 1)
	require.NoError(t, err)

	// 収納に入れて、バックパックに戻す
	require.NoError(t, MoveToStorage(world, item, storageEntity))
	assert.True(t, world.Components.LocationInStorage.Has(item))

	require.NoError(t, MoveToBackpack(world, item, playerEntity))
	assert.True(t, world.Components.LocationInBackpack.Has(item))
	assert.False(t, world.Components.LocationInStorage.Has(item))
}

func TestMoveToBackpack_MergesStackFromStorage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	playerEntity, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// バックパックに回復薬 x3 を配置
	_, err = SpawnBackpackItem(world, "healing_potion", 3)
	require.NoError(t, err)

	// 収納に回復薬 x2 を配置
	storageItem, err := SpawnStorageItem(world, "healing_potion", 2, storageEntity)
	require.NoError(t, err)

	// 収納からバックパックへ移動（統合されるべき）
	require.NoError(t, MoveToBackpack(world, storageItem, playerEntity))

	// 統合はしないので、回復薬は5個のエンティティとしてバックパックに並ぶ
	var entityCount int
	potionQuery := ecs.NewFilter2[gc.LocationInBackpack, gc.Name](world.ECS).Query()
	for potionQuery.Next() {
		if world.Components.Name.Get(potionQuery.Entity()).Name == "Healing Potion" {
			entityCount++
		}
	}

	// 移すのは storageItem 1個だけ。バックパックの3個に1個足されて4個並ぶ
	assert.Equal(t, 4, entityCount, "統合せず、移した1個が加わって4個並ぶ")
}

func TestMoveToBackpack_NoMergeForNonStack(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 非スタックアイテムを2つバックパックに配置
	_, err = SpawnBackpackItem(world, "western_armor", 1)
	require.NoError(t, err)
	_, err = SpawnBackpackItem(world, "western_armor", 1)
	require.NoError(t, err)

	// 非スタックアイテムは統合されず2つ存在する
	var entityCount int
	armorQuery := ecs.NewFilter2[gc.LocationInBackpack, gc.Name](world.ECS).Query()
	for armorQuery.Next() {
		entity := armorQuery.Entity()
		name := world.Components.Name.Get(entity)
		if name.Name == "Western Armor" {
			entityCount++
		}
	}

	assert.Equal(t, 2, entityCount, "非スタックアイテムは統合されない")
}

func TestMoveToStorage_MergesStack(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)

	_, err = SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 収納に回復薬 x3 を配置
	_, err = SpawnStorageItem(world, "healing_potion", 3, storageEntity)
	require.NoError(t, err)

	// バックパックに回復薬 x1 を配置し、収納に移動する
	backpackItem, err := SpawnBackpackItem(world, "healing_potion", 1)
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, backpackItem, storageEntity))

	// 統合はしないので、収納の回復薬は4個のエンティティとして並ぶ
	var entityCount int
	for _, entity := range query.GetStorageItems(world, storageEntity) {
		if world.Components.Name.Get(entity).Name == "Healing Potion" {
			entityCount++
		}
	}

	assert.Equal(t, 4, entityCount, "統合せず4個のエンティティが並ぶ")
}

func TestSpillStorageItems(t *testing.T) {
	t.Parallel()

	t.Run("収納内のアイテムが指定タイルのフィールドへ落ちる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
		require.NoError(t, err)

		item1, err := SpawnStorageItem(world, "healing_potion", 2, storageEntity)
		require.NoError(t, err)
		item2, err := SpawnStorageItem(world, "grenade", 1, storageEntity)
		require.NoError(t, err)

		SpillStorageItems(world, storageEntity, consts.Tile(7), consts.Tile(9))

		for _, item := range []ecs.Entity{item1, item2} {
			assert.False(t, world.Components.LocationInStorage.Has(item), "収納から外れる")
			require.True(t, world.Components.LocationOnField.Has(item), "フィールドへ落ちる")
			grid := world.Components.GridElement.Get(item)
			assert.Equal(t, consts.Tile(7), grid.X)
			assert.Equal(t, consts.Tile(9), grid.Y)
		}
	})

	t.Run("空の収納では何も起きない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		storageEntity, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			SpillStorageItems(world, storageEntity, consts.Tile(1), consts.Tile(1))
		})
		assert.Empty(t, query.GetStorageItems(world, storageEntity))
	})

	t.Run("他の収納の中身には影響しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		storageA, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
		require.NoError(t, err)
		storageB, err := SpawnProp(world, "wooden_crate", consts.Tile(1), consts.Tile(0))
		require.NoError(t, err)

		_, err = SpawnStorageItem(world, "healing_potion", 1, storageA)
		require.NoError(t, err)
		itemB, err := SpawnStorageItem(world, "healing_potion", 1, storageB)
		require.NoError(t, err)

		SpillStorageItems(world, storageA, consts.Tile(5), consts.Tile(5))

		assert.True(t, world.Components.LocationInStorage.Has(itemB), "storageBの中身は影響を受けない")
		assert.Empty(t, query.GetStorageItems(world, storageA), "storageAは空になる")
	})
}

func TestMoveToStorage_DoesNotMergeAcrossStorages(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	const potion = "Healing Potion"

	storageA, err := SpawnProp(world, "wooden_crate", consts.Tile(0), consts.Tile(0))
	require.NoError(t, err)
	storageB, err := SpawnProp(world, "wooden_crate", consts.Tile(1), consts.Tile(0))
	require.NoError(t, err)

	_, err = SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	// 木箱Aに回復薬 x3、木箱Bに回復薬 x2
	_, err = SpawnStorageItem(world, "healing_potion", 3, storageA)
	require.NoError(t, err)
	_, err = SpawnStorageItem(world, "healing_potion", 2, storageB)
	require.NoError(t, err)

	// バックパックに回復薬 x1 を配置し、木箱Aに移動する
	backpackItem, err := SpawnBackpackItem(world, "healing_potion", 1)
	require.NoError(t, err)
	require.NoError(t, MoveToStorage(world, backpackItem, storageA))

	// 木箱Aの回復薬は3個 + 移した1個で4個のエンティティ
	var countA int
	for _, entity := range query.GetStorageItems(world, storageA) {
		if world.Components.Name.Get(entity).Name == potion {
			countA++
		}
	}
	assert.Equal(t, 4, countA, "木箱Aの回復薬は4個")

	// 木箱Bの回復薬は影響を受けず2個のまま
	var countB int
	for _, entity := range query.GetStorageItems(world, storageB) {
		if world.Components.Name.Get(entity).Name == potion {
			countB++
		}
	}
	assert.Equal(t, 2, countB, "木箱Bの回復薬は影響を受けない")
}
