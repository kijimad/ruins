package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
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
		item.AddComponent(world.Components.Material, &gc.Material{})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "テストアイテム"})

		// 1個消費（負の値で減少）
		err := ChangeItemCount(world, item, -1)
		require.NoError(t, err)

		// エンティティが削除されていることを確認
		assert.False(t, item.HasComponent(world.Components.Name), "アイテムが削除されているべき")
	})

	t.Run("Stackableアイテムの一部を消費", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=5のStackableアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Material, &gc.Material{})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{Count: 5})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		// 2個消費
		err := ChangeItemCount(world, item, -2)
		require.NoError(t, err)

		// 残り3個であることを確認
		stackableComp := world.Components.Stackable.MustGet(item)
		assert.Equal(t, 3, stackableComp.Count)
		assert.True(t, item.HasComponent(world.Components.Name), "アイテムは残っているべき")
	})

	t.Run("Stackableアイテムを全て消費すると削除される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=3のStackableアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Material, &gc.Material{})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{Count: 3})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		// 3個全て消費
		err := ChangeItemCount(world, item, -3)
		require.NoError(t, err)

		// エンティティが削除されていることを確認
		assert.False(t, item.HasComponent(world.Components.Name), "アイテムが削除されているべき")
	})

	t.Run("所持数を超えて消費しようとするとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=2のStackableアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Material, &gc.Material{})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{Count: 2})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})
		item.AddComponent(world.Components.Name, &gc.Name{Name: "回復薬"})

		// 5個消費（所持数を超える）
		err := ChangeItemCount(world, item, -5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "アイテム数が不足しています")

		// エンティティは削除されていない
		assert.True(t, item.HasComponent(world.Components.Name), "アイテムは残っているべき")
		stackableComp := world.Components.Stackable.MustGet(item)
		assert.Equal(t, 2, stackableComp.Count, "個数は変更されていないべき")
	})

	t.Run("正の値で個数を増やせる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// Count=3のStackableアイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Material, &gc.Material{})
		item.AddComponent(world.Components.Stackable, &gc.Stackable{Count: 3})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})

		// 2個追加
		err := ChangeItemCount(world, item, 2)
		require.NoError(t, err)

		// 5個になっていることを確認
		stackableComp := world.Components.Stackable.MustGet(item)
		assert.Equal(t, 5, stackableComp.Count)
	})

	t.Run("0を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item := world.Manager.NewEntity()
		err := ChangeItemCount(world, item, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be zero")
	})

	t.Run("プレイヤーがいる場合はWeightDirtyフラグが立つ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		// プレイヤーを作成
		player := world.Manager.NewEntity()
		player.AddComponent(world.Components.Player, &gc.Player{})
		player.AddComponent(world.Components.WeightCapacity, &gc.WeightCapacity{})
		player.AddComponent(world.Components.Abilities, &gc.Abilities{
			Strength: gc.Ability{Base: 10},
		})

		// アイテムを作成
		item := world.Manager.NewEntity()
		item.AddComponent(world.Components.Material, &gc.Material{})
		item.AddComponent(world.Components.LocationInBackpack, &gc.LocationInBackpack{})

		// 1個消費
		err := ChangeItemCount(world, item, -1)
		require.NoError(t, err)

		// WeightDirtyフラグが立っていることを確認
		assert.True(t, player.HasComponent(world.Components.WeightDirty), "WeightDirtyフラグが立つべき")
	})
}
