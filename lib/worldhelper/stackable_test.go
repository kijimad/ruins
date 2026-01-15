package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用素材エンティティを作成
	materialEntity := world.Manager.NewEntity()
	materialEntity.AddComponent(world.Components.Item, &gc.Item{Count: 10})
	materialEntity.AddComponent(world.Components.Stackable, &gc.Stackable{})
	materialEntity.AddComponent(world.Components.ItemLocationInBackpack, &gc.ItemLocationInBackpack)
	materialEntity.AddComponent(world.Components.Name, &gc.Name{Name: "鉄"})

	// 素材の数量を取得
	entity, found := FindStackableInInventory(world, "鉄")
	require.True(t, found, "素材が見つからない")
	item := world.Components.Item.Get(entity).(*gc.Item)
	assert.Equal(t, 10, item.Count, "素材の数量が正しく取得できない")

	// 存在しない素材の数量を取得
	_, found = FindStackableInInventory(world, "存在しない素材")
	assert.False(t, found, "存在しない素材が見つかってはいけない")

	// クリーンアップ
	world.Manager.DeleteEntity(materialEntity)
}

func TestPlusMinusAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用素材エンティティを作成
	materialEntity := world.Manager.NewEntity()
	materialEntity.AddComponent(world.Components.Item, &gc.Item{Count: 10})
	materialEntity.AddComponent(world.Components.Stackable, &gc.Stackable{})
	materialEntity.AddComponent(world.Components.ItemLocationInBackpack, &gc.ItemLocationInBackpack)
	materialEntity.AddComponent(world.Components.Name, &gc.Name{Name: "鉄"})

	// 数量を増加
	err := ChangeStackableCount(world, "鉄", 5)
	require.NoError(t, err)
	entity, found := FindStackableInInventory(world, "鉄")
	require.True(t, found)
	item := world.Components.Item.Get(entity).(*gc.Item)
	assert.Equal(t, 15, item.Count, "数量増加が正しく動作しない")

	// 数量を減少
	err = ChangeStackableCount(world, "鉄", -3)
	require.NoError(t, err)
	entity, found = FindStackableInInventory(world, "鉄")
	require.True(t, found)
	item = world.Components.Item.Get(entity).(*gc.Item)
	assert.Equal(t, 12, item.Count, "数量減少が正しく動作しない")

	// 大量追加テスト（制限なし）
	err = ChangeStackableCount(world, "鉄", 1000)
	require.NoError(t, err)
	entity, found = FindStackableInInventory(world, "鉄")
	require.True(t, found)
	item = world.Components.Item.Get(entity).(*gc.Item)
	assert.Equal(t, 1012, item.Count, "数量が正しく加算されない")

	// 所持数を超えて減らそうとするとエラー
	err = ChangeStackableCount(world, "鉄", -1500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "アイテム数が不足しています")
	// エンティティは残っている
	entity, found = FindStackableInInventory(world, "鉄")
	require.True(t, found)
	item = world.Components.Item.Get(entity).(*gc.Item)
	assert.Equal(t, 1012, item.Count, "個数は変更されていないべき")
}

func TestMergeStackableIntoInventory(t *testing.T) {
	t.Parallel()
	t.Run("Stackableアイテムを既存アイテムにマージする（LocationなしからLocationありへ）", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 既存のアイテム（パン3個）をバックパックに追加
		existingItem := world.Manager.NewEntity()
		existingItem.AddComponent(world.Components.Item, &gc.Item{Count: 3})
		existingItem.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		existingItem.AddComponent(world.Components.Stackable, &gc.Stackable{})
		existingItem.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// 新しいアイテム（パン2個）をフィールドに追加（フィールドから拾った直後の状態）
		newItem := world.Manager.NewEntity()
		newItem.AddComponent(world.Components.Item, &gc.Item{Count: 2})
		newItem.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		newItem.AddComponent(world.Components.Stackable, &gc.Stackable{})
		newItem.AddComponent(world.Components.ItemLocationOnField, &gc.LocationOnField{})

		// マージ実行
		err := MergeStackableIntoInventory(world, newItem, "パン")
		require.NoError(t, err)

		// 既存アイテムの個数が5個になっているか確認
		existingItemComp := world.Components.Item.Get(existingItem).(*gc.Item)
		assert.Equal(t, 5, existingItemComp.Count, "既存アイテムに新しいアイテムの個数が追加される")

		// 新しいアイテムが削除されているか確認
		assert.False(t, newItem.HasComponent(world.Components.Item), "新しいアイテムエンティティは削除される")
	})

	t.Run("既存アイテムがない場合はマージしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 新しいアイテム（パン2個）を追加（まだLocationなし）
		newItem := world.Manager.NewEntity()
		newItem.AddComponent(world.Components.Item, &gc.Item{Count: 2})
		newItem.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		newItem.AddComponent(world.Components.Stackable, &gc.Stackable{})

		// マージ実行
		err := MergeStackableIntoInventory(world, newItem, "パン")
		require.NoError(t, err)

		// 新しいアイテムがそのまま残っているか確認
		assert.True(t, newItem.HasComponent(world.Components.Item), "既存アイテムがない場合は新しいアイテムがそのまま残る")
		newItemComp := world.Components.Item.Get(newItem).(*gc.Item)
		assert.Equal(t, 2, newItemComp.Count, "個数は変わらない")
	})

	t.Run("非Stackableアイテムはマージしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 既存のアイテム（剣）をバックパックに追加
		existingItem := world.Manager.NewEntity()
		existingItem.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		existingItem.AddComponent(world.Components.Name, &gc.Name{Name: "剣"})
		existingItem.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
		// Stackableコンポーネントなし

		// 新しいアイテム（剣）を追加
		newItem := world.Manager.NewEntity()
		newItem.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		newItem.AddComponent(world.Components.Name, &gc.Name{Name: "剣"})
		// Stackableコンポーネントなし

		// マージ実行
		err := MergeStackableIntoInventory(world, newItem, "剣")
		require.NoError(t, err)

		// 新しいアイテムがそのまま残っているか確認
		assert.True(t, newItem.HasComponent(world.Components.Item), "非Stackableアイテムはマージされない")
	})

	t.Run("新しいアイテムがItemLocationInBackpackを持っている場合でもマージされる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// 既存のアイテム（パン3個）をバックパックに追加
		existingItem := world.Manager.NewEntity()
		existingItem.AddComponent(world.Components.Item, &gc.Item{Count: 3})
		existingItem.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		existingItem.AddComponent(world.Components.Stackable, &gc.Stackable{})
		existingItem.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// 新しいアイテム（パン2個）を追加（ItemLocationInBackpackあり）
		newItem := world.Manager.NewEntity()
		newItem.AddComponent(world.Components.Item, &gc.Item{Count: 2})
		newItem.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		newItem.AddComponent(world.Components.Stackable, &gc.Stackable{})
		newItem.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// マージ実行
		err := MergeStackableIntoInventory(world, newItem, "パン")
		require.NoError(t, err)

		// FindStackableInInventoryが既存アイテムを見つけてマージが実行される
		// 既存アイテムの個数が5個になっているか確認
		existingItemComp := world.Components.Item.Get(existingItem).(*gc.Item)
		assert.Equal(t, 5, existingItemComp.Count, "既存アイテムに新しいアイテムの個数が追加される")

		// 新しいアイテムが削除されているか確認
		assert.False(t, newItem.HasComponent(world.Components.Item), "新しいアイテムエンティティは削除される")
	})
}
