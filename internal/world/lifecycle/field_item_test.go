package lifecycle

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/mlange-42/ark/ecs"
)

func TestSpawnFieldItem(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// フィールドアイテムを生成
	item, err := SpawnFieldItem(world, "回復薬", consts.Tile(5), consts.Tile(10), 1)
	require.NoError(t, err, "SpawnFieldItem should not return error")
	require.NotNil(t, item, "アイテムエンティティが生成されるべき")

	// Nameコンポーネントの確認
	require.True(t, world.Components.Name.Has(item), "Nameコンポーネントが必要")
	name := world.Components.Name.Get(item)
	assert.Equal(t, "回復薬", name.Name, "アイテム名が正しくない")

	// GridElementコンポーネントの確認
	require.True(t, world.Components.GridElement.Has(item), "GridElementコンポーネントが必要")
	gridElement := world.Components.GridElement.Get(item)
	assert.Equal(t, consts.Tile(5), gridElement.X, "行位置が正しくない")
	assert.Equal(t, consts.Tile(10), gridElement.Y, "列位置が正しくない")

	// SpriteRenderコンポーネントの確認
	require.True(t, world.Components.SpriteRender.Has(item), "SpriteRenderコンポーネントが必要")
	sprite := world.Components.SpriteRender.Get(item)
	assert.Equal(t, "field", sprite.SpriteSheetName, "スプライトシート名が正しくない")
	assert.Equal(t, "healing_potion", sprite.SpriteKey, "スプライトキーが正しくない")
	assert.Equal(t, gc.DepthNumRug, sprite.Depth, "描画深度が正しくない")

	// LocationOnFieldコンポーネントの確認
	assert.True(t, world.Components.LocationOnField.Has(item), "LocationOnFieldコンポーネントが必要")

	// クリーンアップ
	world.World.RemoveEntity(item)
}

func TestSpawnMultipleFieldItems(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 複数のフィールドアイテムを生成
	items := []struct {
		itemName string
		row      consts.Tile
		col      consts.Tile
	}{
		{"回復薬", consts.Tile(1), consts.Tile(1)},
		{"手榴弾", consts.Tile(2), consts.Tile(2)},
		{"ルビー原石", consts.Tile(3), consts.Tile(3)},
	}

	createdItems := make([]ecs.Entity, 0, len(items))

	for _, itemData := range items {
		item, err := SpawnFieldItem(world, itemData.itemName, itemData.row, itemData.col, 1)
		require.NoError(t, err, "SpawnFieldItem should not return error")
		createdItems = append(createdItems, item)

		// 位置の確認
		gridElement := world.Components.GridElement.Get(item)
		assert.Equal(t, itemData.row, gridElement.X, "行位置が正しくない")
		assert.Equal(t, itemData.col, gridElement.Y, "列位置が正しくない")
	}

	// フィールド上のアイテム数を確認
	fieldItemCount := 0
	fieldItemQuery := ecs.NewFilter2[gc.LocationOnField, gc.GridElement](world.World).Query()
	for fieldItemQuery.Next() {
		fieldItemCount++
	}

	assert.Equal(t, len(items), fieldItemCount, "フィールド上のアイテム数が正しくない")

	// クリーンアップ
	for _, item := range createdItems {
		world.World.RemoveEntity(item)
	}
}
