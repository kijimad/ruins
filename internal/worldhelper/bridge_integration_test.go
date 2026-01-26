package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBridgeIntegration は橋システムの統合テスト
func TestBridgeIntegration(t *testing.T) {
	t.Parallel()

	t.Run("橋エンティティをスポーンできる", func(t *testing.T) {
		t.Parallel()
		testWorld := testutil.InitTestWorld(t)

		// 橋エンティティを生成
		bridgeEntity, err := SpawnBridge(testWorld, "A", 10, 5, 1)
		require.NoError(t, err)

		// Nameコンポーネントを確認
		nameComp := testWorld.Components.Name.Get(bridgeEntity)
		require.NotNil(t, nameComp)
		name := nameComp.(*gc.Name)
		assert.Equal(t, "橋A", name.Name)

		// GridElementコンポーネントを確認
		gridComp := testWorld.Components.GridElement.Get(bridgeEntity)
		require.NotNil(t, gridComp)
		grid := gridComp.(*gc.GridElement)
		assert.Equal(t, gc.Tile(10), grid.X)
		assert.Equal(t, gc.Tile(5), grid.Y)

		// Interactableコンポーネントを確認
		interactableComp := testWorld.Components.Interactable.Get(bridgeEntity)
		require.NotNil(t, interactableComp)
		interactable := interactableComp.(*gc.Interactable)

		// BridgeInteractionであることを確認
		bridgeInteraction, ok := interactable.Data.(gc.BridgeInteraction)
		require.True(t, ok, "Interactable.DataはBridgeInteractionである必要があります")
		assert.Equal(t, "A", bridgeInteraction.BridgeID)
	})

	t.Run("橋の自動発動設定が正しい", func(t *testing.T) {
		t.Parallel()
		testWorld := testutil.InitTestWorld(t)

		bridgeEntity, err := SpawnBridge(testWorld, "A", 10, 5, 1)
		require.NoError(t, err)

		interactable := testWorld.Components.Interactable.Get(bridgeEntity).(*gc.Interactable)
		config := interactable.Data.Config()

		// 自動発動であることを確認
		assert.Equal(t, gc.ActivationWayAuto, config.ActivationWay)
		// 同じタイルで発動することを確認
		assert.Equal(t, gc.ActivationRangeSameTile, config.ActivationRange)
	})

	t.Run("橋にはBridgeIDのみが設定される", func(t *testing.T) {
		t.Parallel()
		testWorld := testutil.InitTestWorld(t)

		// 通常の階層（5の倍数でない）
		bridgeEntity, err := SpawnBridge(testWorld, "A", 10, 5, 1)
		require.NoError(t, err)

		interactable := testWorld.Components.Interactable.Get(bridgeEntity).(*gc.Interactable)
		bridgeInteraction, ok := interactable.Data.(gc.BridgeInteraction)
		require.True(t, ok, "Interactable.DataはBridgeInteractionである必要があります")

		// BridgeIDが正しく設定されていることを確認
		assert.Equal(t, "A", bridgeInteraction.BridgeID)
	})

	t.Run("5の倍数の階層ではPlazaWarpInteractionが設定される", func(t *testing.T) {
		t.Parallel()
		testWorld := testutil.InitTestWorld(t)

		// 5の倍数の階層の橋A
		bridgeEntity, err := SpawnBridge(testWorld, "A", 10, 5, 5)
		require.NoError(t, err)

		interactable := testWorld.Components.Interactable.Get(bridgeEntity).(*gc.Interactable)
		_, ok := interactable.Data.(gc.PlazaWarpInteraction)
		assert.True(t, ok, "5の倍数の階層ではPlazaWarpInteractionが設定される")
	})
}
