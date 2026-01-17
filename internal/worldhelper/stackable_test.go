package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
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

func TestMergeInventoryItem(t *testing.T) {
	t.Parallel()
	t.Run("バックパック内の同名Stackableアイテムを統合する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// バックパック内にパンを3個追加
		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.Item, &gc.Item{Count: 3})
		item1.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		item1.AddComponent(world.Components.Stackable, &gc.Stackable{})
		item1.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// バックパック内にパンを2個追加
		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.Item, &gc.Item{Count: 2})
		item2.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		item2.AddComponent(world.Components.Stackable, &gc.Stackable{})
		item2.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// マージ実行
		err := MergeInventoryItem(world, "パン")
		require.NoError(t, err)

		// バックパック内のパンは1つだけになっている
		var breadCount int
		var totalCount int
		world.Manager.Join(
			world.Components.Stackable,
			world.Components.ItemLocationInBackpack,
			world.Components.Name,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			name := world.Components.Name.Get(entity).(*gc.Name)
			if name.Name == "パン" {
				breadCount++
				item := world.Components.Item.Get(entity).(*gc.Item)
				totalCount += item.Count
			}
		}))

		assert.Equal(t, 1, breadCount, "パンは1つにまとめられる")
		assert.Equal(t, 5, totalCount, "合計個数は5個")
	})

	t.Run("1個しかない場合はマージしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// バックパック内にパンを1個だけ追加
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 2})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "パン"})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{})
		item.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// マージ実行
		err := MergeInventoryItem(world, "パン")
		require.NoError(t, err)

		// アイテムがそのまま残っている
		assert.True(t, item.HasComponent(world.Components.Item), "アイテムがそのまま残る")
		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 2, itemComp.Count, "個数は変わらない")
	})

	t.Run("非Stackableアイテムは統合しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// バックパック内に剣を2つ追加（Stackableなし）
		item1 := world.Manager.NewEntity()
		item1.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item1.AddComponent(world.Components.Name, &gc.Name{Name: "剣"})
		item1.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		item2 := world.Manager.NewEntity()
		item2.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item2.AddComponent(world.Components.Name, &gc.Name{Name: "剣"})
		item2.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// マージ実行
		err := MergeInventoryItem(world, "剣")
		require.NoError(t, err)

		// 両方のアイテムがそのまま残っている
		assert.True(t, item1.HasComponent(world.Components.Item), "item1がそのまま残る")
		assert.True(t, item2.HasComponent(world.Components.Item), "item2がそのまま残る")
	})
}
