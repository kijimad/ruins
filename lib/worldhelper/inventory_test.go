package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeItemCount(t *testing.T) {
	t.Parallel()

	t.Run("単一アイテムを消費すると削除される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=1のアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 1})
		item.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "テストアイテム"})

		// 1個消費（負の値で減少）
		err := ChangeItemCount(world, item, -1)
		require.NoError(t, err)

		// エンティティが削除されていることを確認
		assert.False(t, item.HasComponent(world.Components.Item), "アイテムが削除されているべき")
	})

	t.Run("Stackableアイテムの一部を消費", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=5のStackableアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 5})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{})
		item.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		// 2個消費
		err := ChangeItemCount(world, item, -2)
		require.NoError(t, err)

		// 残り3個であることを確認
		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 3, itemComp.Count)
		assert.True(t, item.HasComponent(world.Components.Item), "アイテムは残っているべき")
	})

	t.Run("Stackableアイテムを全て消費すると削除される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=3のStackableアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 3})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{})
		item.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		// 3個全て消費
		err := ChangeItemCount(world, item, -3)
		require.NoError(t, err)

		// エンティティが削除されていることを確認
		assert.False(t, item.HasComponent(world.Components.Item), "アイテムが削除されているべき")
	})

	t.Run("所持数を超えて消費しようとするとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=2のアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 2})
		item.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		// 5個消費（所持数を超える）
		err := ChangeItemCount(world, item, -5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "アイテム数が不足しています")

		// エンティティは削除されていない
		assert.True(t, item.HasComponent(world.Components.Item), "アイテムは残っているべき")
		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 2, itemComp.Count, "個数は変更されていないべき")
	})

	t.Run("正の値で個数を増やせる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=3のアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 3})
		item.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// 2個追加
		err := ChangeItemCount(world, item, 2)
		require.NoError(t, err)

		// 5個になっていることを確認
		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 5, itemComp.Count)
	})

	t.Run("0を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 5})

		err := ChangeItemCount(world, item, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be zero")
	})

	t.Run("Itemコンポーネントがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Itemコンポーネントのないエンティティ
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Name, &gc.Name{Name: "無効なエンティティ"})

		err := ChangeItemCount(world, item, -1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not have Item component")
	})

	t.Run("プレイヤーがいる場合は重量が再計算される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.Pools, &gc.Pools{})
		player.AddComponent(world.Components.Attributes, &gc.Attributes{
			Strength: gc.Attribute{Base: 10},
		})

		// 重いアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Item, &gc.Item{Count: 2})
		item.AddComponent(world.Components.Weight, &gc.Weight{Kg: 5.0})
		item.AddComponent(world.Components.ItemLocationInBackpack, &gc.LocationInBackpack{})

		// 初期重量を計算
		UpdateCarryingWeight(world, player)
		pools := world.Components.Pools.Get(player).(*gc.Pools)
		initialWeight := pools.Weight.Current
		assert.Equal(t, 10.0, initialWeight) // 5kg × 2個

		// 1個消費
		err := ChangeItemCount(world, item, -1)
		require.NoError(t, err)

		// アイテムのCountが1に減っていることを確認
		itemComp := world.Components.Item.Get(item).(*gc.Item)
		assert.Equal(t, 1, itemComp.Count, "Count should be 1 after consuming 1 item")

		// 手動で再計算して確認（ChangeItemCount内で自動的に呼ばれているはず）
		UpdateCarryingWeight(world, player)

		// 重量が自動的に再計算されていることを確認
		pools = world.Components.Pools.Get(player).(*gc.Pools)
		assert.Equal(t, 5.0, pools.Weight.Current) // 5kg × 1個
	})
}
