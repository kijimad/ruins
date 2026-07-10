package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeMaterialIntoInventoryWithMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.World.NewEntity()
	world.Components.Player.Add(player, nil)

	// 既存のmaterialをバックパックに配置（初期数量5）
	_, err := SpawnBackpackItem(world, "鉄くず", 5)
	require.NoError(t, err)

	// 新しいmaterialを作成（数量3）
	_, err = SpawnBackpackItem(world, "鉄くず", 3)
	require.NoError(t, err)

	// mergeStackableItemsを実行
	err = mergeStackableItems(world, "鉄くず", mergeInBackpack, player)
	require.NoError(t, err)

	// バックパック内の鉄くずは1つだけになっている
	var ironCount int
	var totalCount int
	ironQuery := ecs.NewFilter3[gc.Stackable, gc.LocationInBackpack, gc.Name](world.World).Query()
	for ironQuery.Next() {
		entity := ironQuery.Entity()
		name := world.Components.Name.Get(entity)
		if name.Name == "鉄くず" {
			ironCount++
			stackable := world.Components.Stackable.Get(entity)
			totalCount += stackable.Count
		}
	}

	assert.Equal(t, 1, ironCount, "鉄くずは1つにまとめられる")
	assert.Equal(t, 8, totalCount, "合計個数は8個")
}

func TestMergeMaterialIntoInventoryWithNewMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.World.NewEntity()
	world.Components.Player.Add(player, nil)

	// 新しいmaterialを作成（既存のものはなし）
	_, err := SpawnBackpackItem(world, "緑ハーブ", 2)
	require.NoError(t, err)

	// バックパック内のmaterial数をカウント（統合前）
	materialCountBefore := 0
	materialBeforeQuery := ecs.NewFilter2[gc.Stackable, gc.LocationInBackpack](world.World).Query()
	for materialBeforeQuery.Next() {
		materialCountBefore++
	}

	// mergeStackableItemsを実行（1個しかないので統合されない）
	err = mergeStackableItems(world, "緑ハーブ", mergeInBackpack, player)
	require.NoError(t, err)

	// バックパック内のmaterial数をカウント（統合後）
	materialCountAfter := 0
	materialAfterQuery := ecs.NewFilter2[gc.Stackable, gc.LocationInBackpack](world.World).Query()
	for materialAfterQuery.Next() {
		materialCountAfter++
	}

	// 数は変わっていない（1個だけなので統合不要）
	assert.Equal(t, materialCountBefore, materialCountAfter, "material数は変わらない")
	assert.Equal(t, 1, materialCountAfter, "緑ハーブは1個のまま")
}

func TestMergeMaterialIntoInventoryWithNonMaterial(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.World.NewEntity()
	world.Components.Player.Add(player, nil)

	// 既存のアイテム（Stackableを持たない）をバックパックに配置
	_, err := SpawnBackpackItem(world, "西洋鎧", 1)
	require.NoError(t, err)

	// 新しい同じアイテムを作成
	_, err = SpawnBackpackItem(world, "西洋鎧", 1)
	require.NoError(t, err)

	// バックパック内のアイテム数をカウント（統合前）
	itemCountBefore := 0
	itemBeforeQuery := ecs.NewFilter1[gc.LocationInBackpack](world.World).Query()
	for itemBeforeQuery.Next() {
		itemCountBefore++
	}

	// mergeStackableItemsを実行（Stackableを持たないので統合されない）
	err = mergeStackableItems(world, "西洋鎧", mergeInBackpack, player)
	require.NoError(t, err)

	// バックパック内のアイテム数をカウント（統合後）
	itemCountAfter := 0
	itemAfterQuery := ecs.NewFilter1[gc.LocationInBackpack](world.World).Query()
	for itemAfterQuery.Next() {
		itemCountAfter++
	}

	// Stackableを持たないアイテムは統合されず、2つのアイテムが存在することを確認
	assert.Equal(t, itemCountBefore, itemCountAfter, "Stackableを持たないアイテムは統合されないべき")
	assert.Equal(t, 2, itemCountAfter, "西洋鎧は2つのまま")
}

func TestMergeMaterialIntoInventoryWithoutItemOrMaterialComponent(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player := world.World.NewEntity()
	world.Components.Player.Add(player, nil)

	// Stackableコンポーネントを持たないエンティティを作成（個別アイテムとして扱われる）
	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, gc.EntitySpec{
		Name: &gc.Name{Name: "テスト"},
	})
	_, err := entities.AddEntities(world, componentList)
	require.NoError(t, err)

	// mergeStackableItemsを実行しても何もしない（エラーなし）
	err = mergeStackableItems(world, "テスト", mergeInBackpack, player)
	require.NoError(t, err, "Stackableコンポーネントを持たないエンティティは個別アイテムとして扱われ、マージ不要")
}
