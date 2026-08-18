package lifecycle

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := SpawnBackpackItem(world, "iron", 10)
	require.NoError(t, err)

	// 素材の数量は保存されず、同一スタックのエンティティ数から導出する
	entity, found := query.FindStackInInventory(world, "iron")
	require.True(t, found, "素材が見つからない")
	assert.Equal(t, 10, query.GetEntityCount(world, entity), "素材の数量が正しく取得できない")

	_, found = query.FindStackInInventory(world, "存在しない素材")
	assert.False(t, found, "存在しない素材が見つかってはいけない")
}

func TestPlusMinusAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := SpawnBackpackItem(world, "iron", 10)
	require.NoError(t, err)

	count := func() int {
		entity, found := query.FindStackInInventory(world, "iron")
		require.True(t, found)
		return query.GetEntityCount(world, entity)
	}

	require.NoError(t, ChangeStackCount(world, "iron", 5))
	assert.Equal(t, 15, count(), "数量増加が正しく動作しない")

	require.NoError(t, ChangeStackCount(world, "iron", -3))
	assert.Equal(t, 12, count(), "数量減少が正しく動作しない")

	// 大量追加テスト。上限は無い
	require.NoError(t, ChangeStackCount(world, "iron", 1000))
	assert.Equal(t, 1012, count(), "数量が正しく加算されない")

	// 所持数を超えて減らそうとするとエラー。個数は変わらない
	err = ChangeStackCount(world, "iron", -1500)
	require.ErrorContains(t, err, "insufficient item count")
	assert.Equal(t, 1012, count(), "個数は変更されていないべき")
}

func TestChangeStackCount_未所持アイテムの操作(t *testing.T) {
	t.Parallel()

	t.Run("未所持で正の値を指定すると新規に生成される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		require.NoError(t, ChangeStackCount(world, "healing_potion", 3))

		entity, found := query.FindStackInInventory(world, "healing_potion")
		require.True(t, found, "新規に生成された回復薬が見つかるべき")
		assert.Equal(t, 3, query.GetEntityCount(world, entity))
	})

	t.Run("未所持で負の値を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		err := ChangeStackCount(world, "healing_potion", -1)
		require.Error(t, err)
		assert.ErrorContains(t, err, "stackable item not found: healing_potion")
	})
}
