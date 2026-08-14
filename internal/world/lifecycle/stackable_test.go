package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 実アイテムをスポーンして準備する
	_, err := SpawnBackpackItem(world, "iron", 10)
	require.NoError(t, err)

	// 素材の数量を取得
	entity, found := query.FindStackableInInventory(world, "iron")
	require.True(t, found, "素材が見つからない")
	stackable := world.Components.Stackable.Get(entity)
	assert.Equal(t, 10, stackable.Count, "素材の数量が正しく取得できない")

	// 存在しない素材の数量を取得
	_, found = query.FindStackableInInventory(world, "存在しない素材")
	assert.False(t, found, "存在しない素材が見つかってはいけない")
}

func TestPlusMinusAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 実アイテムをスポーンして準備する
	_, err := SpawnBackpackItem(world, "iron", 10)
	require.NoError(t, err)

	// 数量を増加
	err = ChangeStackableCount(world, "iron", 5)
	require.NoError(t, err)
	entity, found := query.FindStackableInInventory(world, "iron")
	require.True(t, found)
	stackable := world.Components.Stackable.Get(entity)
	assert.Equal(t, 15, stackable.Count, "数量増加が正しく動作しない")

	// 数量を減少
	err = ChangeStackableCount(world, "iron", -3)
	require.NoError(t, err)
	entity, found = query.FindStackableInInventory(world, "iron")
	require.True(t, found)
	stackable = world.Components.Stackable.Get(entity)
	assert.Equal(t, 12, stackable.Count, "数量減少が正しく動作しない")

	// 大量追加テスト（制限なし）
	err = ChangeStackableCount(world, "iron", 1000)
	require.NoError(t, err)
	entity, found = query.FindStackableInInventory(world, "iron")
	require.True(t, found)
	stackable = world.Components.Stackable.Get(entity)
	assert.Equal(t, 1012, stackable.Count, "数量が正しく加算されない")

	// 所持数を超えて減らそうとするとエラー
	err = ChangeStackableCount(world, "iron", -1500)
	require.ErrorContains(t, err, "insufficient item count")
	// エンティティは残っている
	entity, found = query.FindStackableInInventory(world, "iron")
	require.True(t, found)
	stackable = world.Components.Stackable.Get(entity)
	assert.Equal(t, 1012, stackable.Count, "個数は変更されていないべき")
}

func TestChangeStackableCount_未所持アイテムの操作(t *testing.T) {
	t.Parallel()

	t.Run("未所持で正の値を指定すると新規に生成される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		err := ChangeStackableCount(world, "healing_potion", 3)
		require.NoError(t, err)

		entity, found := query.FindStackableInInventory(world, "healing_potion")
		require.True(t, found, "新規に生成された回復薬が見つかるべき")
		assert.Equal(t, 3, world.Components.Stackable.Get(entity).Count)
	})

	t.Run("未所持で負の値を指定するとエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		err := ChangeStackableCount(world, "healing_potion", -1)
		require.Error(t, err)
		assert.ErrorContains(t, err, "stackable item not found: healing_potion")
	})
}

func TestMergeStackableItems(t *testing.T) {
	t.Parallel()
	t.Run("バックパック内の同名Stackableアイテムを統合する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		owner := world.ECS.NewEntity()

		// 未統合の2スタックを作る。SpawnBackpackItem はプレイヤーのバックパックへ入れる際に
		// 自動統合するため、統合前の状態を作るには spawnItemBase で実アイテムを生成し、
		// 所有者への配置だけをテスト用に手で行う
		item1, err := spawnItemBase(world, "bread", 3)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item1, &gc.LocationInBackpack{Owner: owner})

		item2, err := spawnItemBase(world, "bread", 2)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item2, &gc.LocationInBackpack{Owner: owner})

		// マージ実行
		err = mergeStackableItems(world, "bread", mergeInBackpack, owner)
		require.NoError(t, err)

		// バックパック内のパンは1つだけになっている
		var breadCount int
		var totalCount int
		breadQuery := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.RawID](world.ECS).Query()
		for breadQuery.Next() {
			entity := breadQuery.Entity()
			if world.Components.RawID.Get(entity).ID == "bread" {
				breadCount++
				totalCount += world.Components.Stackable.Get(entity).Count
			}
		}

		assert.Equal(t, 1, breadCount, "パンは1つにまとめられる")
		assert.Equal(t, 5, totalCount, "合計個数は5個")
	})

	t.Run("1個しかない場合はマージしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		owner := world.ECS.NewEntity()

		item, err := spawnItemBase(world, "bread", 2)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		// マージ実行
		err = mergeStackableItems(world, "bread", mergeInBackpack, owner)
		require.NoError(t, err)

		// アイテムがそのまま残っている
		assert.True(t, world.ECS.Alive(item), "アイテムがそのまま残る")
		assert.Equal(t, 2, world.Components.Stackable.Get(item).Count, "個数は変わらない")
	})

	t.Run("非Stackableアイテムは統合しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		owner := world.ECS.NewEntity()

		// 非Stackableな近接武器を2つ配置する
		item1, err := spawnItemBase(world, "claymore", 1)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item1, &gc.LocationInBackpack{Owner: owner})

		item2, err := spawnItemBase(world, "claymore", 1)
		require.NoError(t, err)
		world.Components.LocationInBackpack.Add(item2, &gc.LocationInBackpack{Owner: owner})

		// マージ実行
		err = mergeStackableItems(world, "claymore", mergeInBackpack, owner)
		require.NoError(t, err)

		// 両方のアイテムがそのまま残っている
		assert.True(t, world.ECS.Alive(item1), "item1がそのまま残る")
		assert.True(t, world.ECS.Alive(item2), "item2がそのまま残る")
	})
}
