package lifecycle

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeItemCount(t *testing.T) {
	t.Parallel()

	t.Run("単一アイテムを消費すると削除される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		// 1個消費（負の値で減少）
		err = ChangeItemCount(world, item, -1)
		require.NoError(t, err)

		// エンティティが削除されていることを確認
		assert.False(t, world.ECS.Alive(item), "アイテムが削除されているべき")
	})

	t.Run("Stackableアイテムの一部を消費", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "healing_potion", 5)
		require.NoError(t, err)

		// 2個消費
		err = ChangeItemCount(world, item, -2)
		require.NoError(t, err)

		// 残り3個であることを確認
		stackableComp := world.Components.Stackable.Get(item)
		assert.Equal(t, 3, stackableComp.Count)
		assert.True(t, world.Components.Name.Has(item), "アイテムは残っているべき")
	})

	t.Run("Stackableアイテムを全て消費すると削除される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "healing_potion", 3)
		require.NoError(t, err)

		// 3個全て消費
		err = ChangeItemCount(world, item, -3)
		require.NoError(t, err)

		// エンティティが削除されていることを確認
		assert.False(t, world.ECS.Alive(item), "アイテムが削除されているべき")
	})

	t.Run("所持数を超えて消費しようとするとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "healing_potion", 2)
		require.NoError(t, err)

		// 5個消費（所持数を超える）
		err = ChangeItemCount(world, item, -5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient item count")

		// エンティティは削除されていない
		assert.True(t, world.Components.Name.Has(item), "アイテムは残っているべき")
		stackableComp := world.Components.Stackable.Get(item)
		assert.Equal(t, 2, stackableComp.Count, "個数は変更されていないべき")
	})

	t.Run("正の値で個数を増やせる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "healing_potion", 3)
		require.NoError(t, err)

		// 2個追加
		err = ChangeItemCount(world, item, 2)
		require.NoError(t, err)

		// 5個になっていることを確認
		stackableComp := world.Components.Stackable.Get(item)
		assert.Equal(t, 5, stackableComp.Count)
	})

	t.Run("0を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		item, err := SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		err = ChangeItemCount(world, item, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be zero")
	})

	t.Run("プレイヤーがいる場合はWeightDirtyフラグが立つ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		player, err := SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
		require.NoError(t, err)

		item, err := SpawnBackpackItem(world, "wooden_sword", 1)
		require.NoError(t, err)

		// スポーンや配置で立った WeightDirty を一旦落とし、消費操作だけで再び立つことを検証する
		ensureRemoved(world.Components.WeightDirty, player)

		// 1個消費
		err = ChangeItemCount(world, item, -1)
		require.NoError(t, err)

		// WeightDirtyフラグが立っていることを確認
		assert.True(t, world.Components.WeightDirty.Has(player), "WeightDirtyフラグが立つべき")
	})
}
