package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TestBridgeIntegration は橋システムの統合テスト
func TestBridgeIntegration(t *testing.T) {
	t.Parallel()

	t.Run("橋エンティティをスポーンできる", func(t *testing.T) {
		t.Parallel()
		testWorld := testutil.InitTestWorld(t)

		// 橋エンティティを生成
		bridgeEntity, err := SpawnBridge(testWorld, "A", 10, 5, 1, 12345)
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
		assert.NotZero(t, bridgeInteraction.NextFloorSeed)
	})

	t.Run("橋ごとに異なるシード値が設定される", func(t *testing.T) {
		t.Parallel()
		testWorld := testutil.InitTestWorld(t)

		// 3つの橋を生成
		bridgeA, err := SpawnBridge(testWorld, "A", 10, 5, 1, 12345)
		require.NoError(t, err)
		bridgeB, err := SpawnBridge(testWorld, "B", 20, 5, 1, 12345)
		require.NoError(t, err)
		bridgeC, err := SpawnBridge(testWorld, "C", 30, 5, 1, 12345)
		require.NoError(t, err)

		// 各橋のシード値を取得
		getNextFloorSeed := func(entity ecs.Entity) uint64 {
			interactable := testWorld.Components.Interactable.Get(entity).(*gc.Interactable)
			bridge := interactable.Data.(gc.BridgeInteraction)
			return bridge.NextFloorSeed
		}

		seedA := getNextFloorSeed(bridgeA)
		seedB := getNextFloorSeed(bridgeB)
		seedC := getNextFloorSeed(bridgeC)

		// 全て異なることを確認
		assert.NotEqual(t, seedA, seedB, "橋AとBのシードは異なるべき")
		assert.NotEqual(t, seedA, seedC, "橋AとCのシードは異なるべき")
		assert.NotEqual(t, seedB, seedC, "橋BとCのシードは異なるべき")
	})

	t.Run("シード値の計算が正しい", func(t *testing.T) {
		t.Parallel()

		baseDepth := 1
		gameSeed := uint64(12345)

		seedA := CalculateBridgeSeed(baseDepth, gameSeed, "A")
		seedB := CalculateBridgeSeed(baseDepth, gameSeed, "B")
		seedC := CalculateBridgeSeed(baseDepth, gameSeed, "C")
		seedD := CalculateBridgeSeed(baseDepth, gameSeed, "D")

		// 期待値の検証
		// baseSeed = 1 + 12345 = 12346
		assert.Equal(t, uint64(13346), seedA) // 12346 + 1000
		assert.Equal(t, uint64(14346), seedB) // 12346 + 2000
		assert.Equal(t, uint64(15346), seedC) // 12346 + 3000
		assert.Equal(t, uint64(16346), seedD) // 12346 + 4000
	})

	t.Run("橋の自動発動設定が正しい", func(t *testing.T) {
		t.Parallel()
		testWorld := testutil.InitTestWorld(t)

		bridgeEntity, err := SpawnBridge(testWorld, "A", 10, 5, 1, 12345)
		require.NoError(t, err)

		interactable := testWorld.Components.Interactable.Get(bridgeEntity).(*gc.Interactable)
		config := interactable.Data.Config()

		// 自動発動であることを確認
		assert.Equal(t, gc.ActivationWayAuto, config.ActivationWay)
		// 同じタイルで発動することを確認
		assert.Equal(t, gc.ActivationRangeSameTile, config.ActivationRange)
	})
}
