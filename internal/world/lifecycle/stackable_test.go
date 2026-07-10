package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/mlange-42/ark/ecs"
)

func TestGetAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用素材エンティティを作成
	materialEntity := world.World.NewEntity()
	world.Components.Stackable.Add(materialEntity, &gc.Stackable{Count: 10})
	world.Components.LocationInBackpack.Add(materialEntity, &gc.LocationInBackpack{})
	world.Components.Name.Add(materialEntity, &gc.Name{Name: "鉄"})

	// 素材の数量を取得
	entity, found := query.FindStackableInInventory(world, "鉄")
	require.True(t, found, "素材が見つからない")
	stackable := world.Components.Stackable.Get(entity)
	assert.Equal(t, 10, stackable.Count, "素材の数量が正しく取得できない")

	// 存在しない素材の数量を取得
	_, found = query.FindStackableInInventory(world, "存在しない素材")
	assert.False(t, found, "存在しない素材が見つかってはいけない")

	// クリーンアップ
	world.World.RemoveEntity(materialEntity)
}

func TestPlusMinusAmount(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テスト用素材エンティティを作成
	materialEntity := world.World.NewEntity()
	world.Components.Material.Add(materialEntity, &gc.Material{})
	world.Components.Stackable.Add(materialEntity, &gc.Stackable{Count: 10})
	world.Components.LocationInBackpack.Add(materialEntity, &gc.LocationInBackpack{})
	world.Components.Name.Add(materialEntity, &gc.Name{Name: "鉄"})

	// 数量を増加
	err := ChangeStackableCount(world, "鉄", 5)
	require.NoError(t, err)
	entity, found := query.FindStackableInInventory(world, "鉄")
	require.True(t, found)
	stackable := world.Components.Stackable.Get(entity)
	assert.Equal(t, 15, stackable.Count, "数量増加が正しく動作しない")

	// 数量を減少
	err = ChangeStackableCount(world, "鉄", -3)
	require.NoError(t, err)
	entity, found = query.FindStackableInInventory(world, "鉄")
	require.True(t, found)
	stackable = world.Components.Stackable.Get(entity)
	assert.Equal(t, 12, stackable.Count, "数量減少が正しく動作しない")

	// 大量追加テスト（制限なし）
	err = ChangeStackableCount(world, "鉄", 1000)
	require.NoError(t, err)
	entity, found = query.FindStackableInInventory(world, "鉄")
	require.True(t, found)
	stackable = world.Components.Stackable.Get(entity)
	assert.Equal(t, 1012, stackable.Count, "数量が正しく加算されない")

	// 所持数を超えて減らそうとするとエラー
	err = ChangeStackableCount(world, "鉄", -1500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "アイテム数が不足しています")
	// エンティティは残っている
	entity, found = query.FindStackableInInventory(world, "鉄")
	require.True(t, found)
	stackable = world.Components.Stackable.Get(entity)
	assert.Equal(t, 1012, stackable.Count, "個数は変更されていないべき")
}

func TestMergeStackableItems(t *testing.T) {
	t.Parallel()
	t.Run("バックパック内の同名Stackableアイテムを統合する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		owner := world.World.NewEntity()

		// バックパック内にパンを3個追加
		item1 := world.World.NewEntity()
		world.Components.Material.Add(item1, &gc.Material{})
		world.Components.Name.Add(item1, &gc.Name{Name: "パン"})
		world.Components.Stackable.Add(item1, &gc.Stackable{Count: 3})
		world.Components.LocationInBackpack.Add(item1, &gc.LocationInBackpack{Owner: owner})

		// バックパック内にパンを2個追加
		item2 := world.World.NewEntity()
		world.Components.Material.Add(item2, &gc.Material{})
		world.Components.Name.Add(item2, &gc.Name{Name: "パン"})
		world.Components.Stackable.Add(item2, &gc.Stackable{Count: 2})
		world.Components.LocationInBackpack.Add(item2, &gc.LocationInBackpack{Owner: owner})

		// マージ実行
		err := mergeStackableItems(world, "パン", mergeInBackpack, owner)
		require.NoError(t, err)

		// バックパック内のパンは1つだけになっている
		var breadCount int
		var totalCount int
		world.Manager.Join(
			world.Components.Stackable,
			world.Components.LocationInBackpack,
			world.Components.Name,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			name := world.Components.Name.Get(entity)
			if name.Name == "パン" {
				breadCount++
				stackable := world.Components.Stackable.Get(entity)
				totalCount += stackable.Count
			}
		}))

		assert.Equal(t, 1, breadCount, "パンは1つにまとめられる")
		assert.Equal(t, 5, totalCount, "合計個数は5個")
	})

	t.Run("1個しかない場合はマージしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		owner := world.World.NewEntity()

		// バックパック内にパンを1個だけ追加
		item := world.World.NewEntity()
		world.Components.Material.Add(item, &gc.Material{})
		world.Components.Name.Add(item, &gc.Name{Name: "パン"})
		world.Components.Stackable.Add(item, &gc.Stackable{Count: 2})
		world.Components.LocationInBackpack.Add(item, &gc.LocationInBackpack{Owner: owner})

		// マージ実行
		err := mergeStackableItems(world, "パン", mergeInBackpack, owner)
		require.NoError(t, err)

		// アイテムがそのまま残っている
		assert.True(t, world.Components.Name.Has(item), "アイテムがそのまま残る")
		stackableComp := world.Components.Stackable.Get(item)
		assert.Equal(t, 2, stackableComp.Count, "個数は変わらない")
	})

	t.Run("非Stackableアイテムは統合しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)

		owner := world.World.NewEntity()

		// バックパック内に剣を2つ追加（Stackableなし）
		item1 := world.World.NewEntity()
		world.Components.Melee.Add(item1, &gc.Melee{})
		world.Components.Name.Add(item1, &gc.Name{Name: "剣"})
		world.Components.LocationInBackpack.Add(item1, &gc.LocationInBackpack{Owner: owner})

		item2 := world.World.NewEntity()
		world.Components.Melee.Add(item2, &gc.Melee{})
		world.Components.Name.Add(item2, &gc.Name{Name: "剣"})
		world.Components.LocationInBackpack.Add(item2, &gc.LocationInBackpack{Owner: owner})

		// マージ実行
		err := mergeStackableItems(world, "剣", mergeInBackpack, owner)
		require.NoError(t, err)

		// 両方のアイテムがそのまま残っている
		assert.True(t, world.Components.Name.Has(item1), "item1がそのまま残る")
		assert.True(t, world.Components.Name.Has(item2), "item2がそのまま残る")
	})
}
