package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestSpawnFieldItem(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// フィールドアイテムを生成
	item, err := SpawnFieldItem(world, "回復薬", gc.Tile(5), gc.Tile(10))
	require.NoError(t, err, "SpawnFieldItem should not return error")
	require.NotNil(t, item, "アイテムエンティティが生成されるべき")

	// Nameコンポーネントの確認
	require.True(t, item.HasComponent(world.Components.Name), "Nameコンポーネントが必要")
	name := world.Components.Name.Get(item).(*gc.Name)
	assert.Equal(t, "回復薬", name.Name, "アイテム名が正しくない")

	// GridElementコンポーネントの確認
	require.True(t, item.HasComponent(world.Components.GridElement), "GridElementコンポーネントが必要")
	gridElement := world.Components.GridElement.Get(item).(*gc.GridElement)
	assert.Equal(t, gc.Tile(5), gridElement.X, "行位置が正しくない")
	assert.Equal(t, gc.Tile(10), gridElement.Y, "列位置が正しくない")

	// SpriteRenderコンポーネントの確認
	require.True(t, item.HasComponent(world.Components.SpriteRender), "SpriteRenderコンポーネントが必要")
	sprite := world.Components.SpriteRender.Get(item).(*gc.SpriteRender)
	assert.Equal(t, "field", sprite.SpriteSheetName, "スプライトシート名が正しくない")
	assert.Equal(t, "heal_item", sprite.SpriteKey, "スプライトキーが正しくない")
	assert.Equal(t, gc.DepthNumRug, sprite.Depth, "描画深度が正しくない")

	// ItemLocationOnFieldコンポーネントの確認
	assert.True(t, item.HasComponent(world.Components.ItemLocationOnField), "ItemLocationOnFieldコンポーネントが必要")

	// クリーンアップ
	world.Manager.DeleteEntity(item)
}

func TestSpawnMultipleFieldItems(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 複数のフィールドアイテムを生成
	items := []struct {
		itemName string
		row      gc.Tile
		col      gc.Tile
	}{
		{"回復薬", gc.Tile(1), gc.Tile(1)},
		{"手榴弾", gc.Tile(2), gc.Tile(2)},
		{"ルビー原石", gc.Tile(3), gc.Tile(3)},
	}

	createdItems := make([]ecs.Entity, 0, len(items))

	for _, itemData := range items {
		item, err := SpawnFieldItem(world, itemData.itemName, itemData.row, itemData.col)
		require.NoError(t, err, "SpawnFieldItem should not return error")
		createdItems = append(createdItems, item)

		// 位置の確認
		gridElement := world.Components.GridElement.Get(item).(*gc.GridElement)
		assert.Equal(t, itemData.row, gridElement.X, "行位置が正しくない")
		assert.Equal(t, itemData.col, gridElement.Y, "列位置が正しくない")
	}

	// フィールド上のアイテム数を確認
	fieldItemCount := 0
	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationOnField,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		fieldItemCount++
	}))

	assert.Equal(t, len(items), fieldItemCount, "フィールド上のアイテム数が正しくない")

	// クリーンアップ
	for _, item := range createdItems {
		world.Manager.DeleteEntity(item)
	}
}
