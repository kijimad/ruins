package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestMergeMaterialIntoInventoryWithMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 既存のmaterialをバックパックに配置（初期数量5）
	_, err := SpawnItem(world, "鉄くず", 5, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)

	// 新しいmaterialを作成（数量3）
	_, err = SpawnItem(world, "鉄くず", 3, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)

	// MergeInventoryItemを実行
	err = MergeInventoryItem(world, "鉄くず")
	require.NoError(t, err)

	// バックパック内の鉄くずは1つだけになっている
	var ironCount int
	var totalCount int
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInPlayerBackpack,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == "鉄くず" {
			ironCount++
			item := world.Components.Item.Get(entity).(*gc.Item)
			totalCount += item.Count
		}
	}))

	assert.Equal(t, 1, ironCount, "鉄くずは1つにまとめられる")
	assert.Equal(t, 8, totalCount, "合計個数は8個")
}

func TestMergeMaterialIntoInventoryWithNewMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 新しいmaterialを作成（既存のものはなし）
	_, err := SpawnItem(world, "緑ハーブ", 2, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)

	// バックパック内のmaterial数をカウント（統合前）
	materialCountBefore := 0
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		materialCountBefore++
	}))

	// MergeInventoryItemを実行（1個しかないので統合されない）
	err = MergeInventoryItem(world, "緑ハーブ")
	require.NoError(t, err)

	// バックパック内のmaterial数をカウント（統合後）
	materialCountAfter := 0
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		materialCountAfter++
	}))

	// 数は変わっていない（1個だけなので統合不要）
	assert.Equal(t, materialCountBefore, materialCountAfter, "material数は変わらない")
	assert.Equal(t, 1, materialCountAfter, "緑ハーブは1個のまま")
}

func TestMergeMaterialIntoInventoryWithNonMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 既存のアイテム（Stackableを持たない）をバックパックに配置
	_, err := SpawnItem(world, "西洋鎧", 1, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)

	// 新しい同じアイテムを作成
	_, err = SpawnItem(world, "西洋鎧", 1, gc.ItemLocationInPlayerBackpack)
	require.NoError(t, err)

	// バックパック内のアイテム数をカウント（統合前）
	itemCountBefore := 0
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountBefore++
	}))

	// MergeInventoryItemを実行（Stackableを持たないので統合されない）
	err = MergeInventoryItem(world, "西洋鎧")
	require.NoError(t, err)

	// バックパック内のアイテム数をカウント（統合後）
	itemCountAfter := 0
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		itemCountAfter++
	}))

	// Stackableを持たないアイテムは統合されず、2つのアイテムが存在することを確認
	assert.Equal(t, itemCountBefore, itemCountAfter, "Stackableを持たないアイテムは統合されないべき")
	assert.Equal(t, 2, itemCountAfter, "西洋鎧は2つのまま")
}

func TestMergeMaterialIntoInventoryWithoutItemOrMaterialComponent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// Stackableコンポーネントを持たないエンティティを作成（個別アイテムとして扱われる）
	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, gc.EntitySpec{
		Name: &gc.Name{Name: "テスト"},
	})
	_, err := entities.AddEntities(world, componentList)
	require.NoError(t, err)

	// MergeInventoryItemを実行しても何もしない（エラーなし）
	err = MergeInventoryItem(world, "テスト")
	require.NoError(t, err, "Stackableコンポーネントを持たないエンティティは個別アイテムとして扱われ、マージ不要")
}
