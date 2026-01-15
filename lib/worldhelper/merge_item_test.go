package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/lib/components"
	"github.com/kijimaD/ruins/lib/engine/entities"
	"github.com/kijimaD/ruins/lib/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestMergeMaterialIntoInventoryWithMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 既存のmaterialをバックパックに配置（初期数量5）
	existingMaterial, err := SpawnItem(world, "鉄くず", 5, gc.ItemLocationInBackpack)
	require.NoError(t, err)

	// 新しいmaterialを作成（数量3）
	newMaterial, err := SpawnItem(world, "鉄くず", 3, gc.ItemLocationInBackpack)
	require.NoError(t, err)

	// MergeStackableIntoInventoryを実行
	err = MergeStackableIntoInventory(world, newMaterial, "鉄くず")
	require.NoError(t, err)

	// 既存のmaterialの数量が統合されていることを確認
	updatedMat := world.Components.Item.Get(existingMaterial).(*gc.Item)
	assert.Equal(t, 8, updatedMat.Count, "数量が正しく統合されていない")

	// 新しいmaterialエンティティが削除されていることを確認（コンポーネントが存在しない）
	assert.False(t, newMaterial.HasComponent(world.Components.Stackable), "新しいmaterialエンティティが削除されていない")
}

func TestMergeMaterialIntoInventoryWithNewMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 新しいmaterialを作成（既存のものはなし）
	newMaterial, err := SpawnItem(world, "緑ハーブ", 2, gc.ItemLocationInBackpack)
	require.NoError(t, err)

	// バックパック内のmaterial数をカウント（統合前）
	materialCountBefore := 0
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		materialCountBefore++
	}))

	// MergeStackableIntoInventoryを実行
	err = MergeStackableIntoInventory(world, newMaterial, "緑ハーブ")
	require.NoError(t, err)

	// バックパック内のmaterial数をカウント（統合後）
	materialCountAfter := 0
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		materialCountAfter++
	}))

	// 新しいmaterialとして追加されていることを確認
	assert.Equal(t, materialCountBefore, materialCountAfter, "新しいmaterialが正しく追加されていない")
	assert.True(t, newMaterial.HasComponent(world.Components.Stackable), "新しいmaterialエンティティが生きているべき")

	// 数量が維持されていることを確認
	updatedMat := world.Components.Item.Get(newMaterial).(*gc.Item)
	assert.Equal(t, 2, updatedMat.Count, "数量が維持されていない")
}

func TestMergeMaterialIntoInventoryWithNonMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 既存のアイテム（Stackableを持たない）をバックパックに配置
	existingItem, err := SpawnItem(world, "西洋鎧", 1, gc.ItemLocationInBackpack)
	require.NoError(t, err)

	// 新しい同じアイテムを作成
	newItem, err := SpawnItem(world, "西洋鎧", 1, gc.ItemLocationInBackpack)
	require.NoError(t, err)

	// バックパック内のアイテム数をカウント（統合前）
	itemCountBefore := 0
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountBefore++
	}))

	// MergeStackableIntoInventoryを実行
	err = MergeStackableIntoInventory(world, newItem, "西洋鎧")
	require.NoError(t, err)

	// バックパック内のアイテム数をカウント（統合後）
	itemCountAfter := 0
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountAfter++
	}))

	// Stackableを持たないアイテムは統合されず、2つのアイテムが存在することを確認
	assert.Equal(t, itemCountBefore, itemCountAfter, "Stackableを持たないアイテムは統合されないべき")
	assert.True(t, existingItem.HasComponent(world.Components.Item), "既存のアイテムエンティティが生きているべき")
	assert.True(t, newItem.HasComponent(world.Components.Item), "新しいアイテムエンティティが生きているべき")
}

func TestMergeMaterialIntoInventoryWithoutItemOrMaterialComponent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// Stackableコンポーネントを持たないエンティティを作成（個別アイテムとして扱われる）
	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, gc.EntitySpec{
		Name: &gc.Name{Name: "テスト"},
	})
	entities, err := entities.AddEntities(world, componentList)
	require.NoError(t, err)
	nonStackableEntity := entities[0]

	// MergeStackableIntoInventoryを実行しても何もしない（エラーなし）
	err = MergeStackableIntoInventory(world, nonStackableEntity, "テスト")
	require.NoError(t, err, "Stackableコンポーネントを持たないエンティティは個別アイテムとして扱われ、マージ不要")
}
