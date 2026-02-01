package states

import (
	"fmt"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/turns"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TestDoActionUIActions はUI系アクションのテスト
// UI系アクションは常に実行可能で、ステート遷移を返す
func TestDoActionUIActions(t *testing.T) {
	t.Parallel()

	// インターフェース実装の確認（コンパイル時チェック）
	var _ es.State[w.World] = &DungeonState{}
	var _ es.ActionHandler[w.World] = &DungeonState{}

	tests := []struct {
		name              string
		action            inputmapper.ActionID
		expectedType      es.TransType
		shouldHaveFunc    bool
		expectedStateType string
		expectError       bool
	}{
		{
			name:              "ダンジョンメニューを開く",
			action:            inputmapper.ActionOpenDungeonMenu,
			expectedType:      es.TransPush,
			shouldHaveFunc:    true,
			expectedStateType: "*states.PersistentMessageState",
		},
		{
			name:              "インベントリを開く",
			action:            inputmapper.ActionOpenInventory,
			expectedType:      es.TransPush,
			shouldHaveFunc:    true,
			expectedStateType: "*states.InventoryMenuState",
		},
		{
			name:        "未知のアクション",
			action:      inputmapper.ActionID("unknown"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			world := testutil.InitTestWorld(t)
			state := &DungeonState{}

			transition, err := state.DoAction(world, tt.action)

			if tt.expectError {
				require.Error(t, err, "DoActionがエラーを返すべきです")
				return
			}

			require.NoError(t, err, "DoActionがエラーを返しました")

			assert.Equal(t, tt.expectedType, transition.Type, "トランジションタイプが不正")

			if tt.shouldHaveFunc {
				require.NotEmpty(t, transition.NewStateFuncs, "NewStateFuncsが空です")

				// ステートファクトリーが実際に動作することを確認
				newState := transition.NewStateFuncs[0]()
				require.NotNil(t, newState, "NewStateFunc が nil を返しました")

				// ステートの型を検証
				actualType := fmt.Sprintf("%T", newState)
				assert.Equal(t, tt.expectedStateType, actualType, "期待するステート型と異なります")
			}
		})
	}
}

// TestDoActionMovementActions は移動系アクションが座標を変更することを検証
func TestDoActionMovementActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		action         inputmapper.ActionID
		expectedDeltaX int
		expectedDeltaY int
	}{
		{inputmapper.ActionMoveNorth, 0, -1},
		{inputmapper.ActionMoveSouth, 0, 1},
		{inputmapper.ActionMoveEast, 1, 0},
		{inputmapper.ActionMoveWest, -1, 0},
		{inputmapper.ActionMoveNorthEast, 1, -1},
		{inputmapper.ActionMoveNorthWest, -1, -1},
		{inputmapper.ActionMoveSouthEast, 1, 1},
		{inputmapper.ActionMoveSouthWest, -1, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			t.Parallel()

			// プレイヤー付きのテストワールドを作成
			initialX, initialY := 10, 10
			world, playerEntity := setupTestWorldWithPlayer(t, initialX, initialY)

			state := &DungeonState{}

			// 移動前の座標を確認
			gridBeforeComponent := world.Components.GridElement.Get(playerEntity)
			require.NotNil(t, gridBeforeComponent, "GridElementコンポーネントが取得できません: エンティティID=%v", playerEntity)
			gridBefore := gridBeforeComponent.(*gc.GridElement)
			require.Equal(t, initialX, int(gridBefore.X), "初期X座標が不正")
			require.Equal(t, initialY, int(gridBefore.Y), "初期Y座標が不正")

			// 移動アクションを実行
			transition, err := state.DoAction(world, tt.action)
			require.NoError(t, err, "DoActionがエラーを返しました")

			// 移動アクションはステート遷移しない
			assert.Equal(t, es.TransNone, transition.Type, "トランジションタイプが不正")

			// 移動後の座標を確認
			gridAfterComponent := world.Components.GridElement.Get(playerEntity)
			require.NotNil(t, gridAfterComponent, "移動後にGridElementコンポーネントが取得できません: エンティティID=%v", playerEntity)
			gridAfter := gridAfterComponent.(*gc.GridElement)
			expectedX := initialX + tt.expectedDeltaX
			expectedY := initialY + tt.expectedDeltaY

			assert.Equal(t, expectedX, int(gridAfter.X), "移動後のX座標が不正")
			assert.Equal(t, expectedY, int(gridAfter.Y), "移動後のY座標が不正")
		})
	}
}

// TestDoActionTurnManagement はターン管理が正しく機能することを検証
func TestDoActionTurnManagement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		action           inputmapper.ActionID
		turnPhase        turns.TurnPhase
		expectedTrans    es.TransType
		isUIAction       bool
		isMoveAction     bool
		shouldMovePlayer bool
	}{
		{
			name:          "プレイヤーターン中のUI操作",
			action:        inputmapper.ActionOpenDungeonMenu,
			turnPhase:     turns.PlayerTurn,
			expectedTrans: es.TransPush,
			isUIAction:    true,
		},
		{
			name:          "AIターン中のUI操作",
			action:        inputmapper.ActionOpenDungeonMenu,
			turnPhase:     turns.AITurn,
			expectedTrans: es.TransPush,
			isUIAction:    true,
		},
		{
			name:             "プレイヤーターン中の移動",
			action:           inputmapper.ActionMoveNorth,
			turnPhase:        turns.PlayerTurn,
			expectedTrans:    es.TransNone,
			isMoveAction:     true,
			shouldMovePlayer: true,
		},
		{
			name:             "AIターン中の移動（実行されない）",
			action:           inputmapper.ActionMoveNorth,
			turnPhase:        turns.AITurn,
			expectedTrans:    es.TransNone,
			isMoveAction:     true,
			shouldMovePlayer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// プレイヤー付きのテストワールドを作成（移動テストの場合）
			var world w.World
			var playerEntity ecs.Entity
			initialX, initialY := 10, 10

			if tt.isMoveAction {
				world, playerEntity = setupTestWorldWithPlayer(t, initialX, initialY)
			} else {
				world = testutil.InitTestWorld(t)
				turnManager := turns.NewTurnManager()
				world.Resources.TurnManager = turnManager
			}

			turnManager := world.Resources.TurnManager.(*turns.TurnManager)
			turnManager.TurnPhase = tt.turnPhase

			state := &DungeonState{}

			transition, err := state.DoAction(world, tt.action)
			require.NoError(t, err, "DoActionがエラーを返しました")

			assert.Equal(t, tt.expectedTrans, transition.Type, "トランジションタイプが不正")

			// UI系アクションの場合、どのターンフェーズでもステートが追加される
			if tt.isUIAction && tt.expectedTrans == es.TransPush {
				assert.NotEmpty(t, transition.NewStateFuncs, "UI系アクションでNewStateFuncsが空です")
			}

			// 移動アクションの場合、座標変化を検証
			if tt.isMoveAction {
				gridAfterComponent := world.Components.GridElement.Get(playerEntity)
				require.NotNil(t, gridAfterComponent, "移動後にGridElementコンポーネントが取得できません: エンティティID=%v", playerEntity)
				gridAfter := gridAfterComponent.(*gc.GridElement)
				if tt.shouldMovePlayer {
					// プレイヤーターン中は移動が実行される
					expectedY := initialY - 1 // ActionMoveNorth
					assert.Equal(t, expectedY, int(gridAfter.Y), "移動が実行されていません")
				} else {
					// AIターン中は移動が実行されない
					assert.Equal(t, initialX, int(gridAfter.X), "AIターン中にX座標が変更されました")
					assert.Equal(t, initialY, int(gridAfter.Y), "AIターン中にY座標が変更されました")
				}
			}
		})
	}
}

// TestDoActionUIActionsAlwaysWork はUI系アクションがターンフェーズに関わらず動作することを検証
func TestDoActionUIActionsAlwaysWork(t *testing.T) {
	t.Parallel()

	turnPhases := []turns.TurnPhase{
		turns.PlayerTurn,
		turns.AITurn,
		turns.TurnEnd,
	}

	for _, phase := range turnPhases {
		t.Run(fmt.Sprintf("TurnPhase_%d", phase), func(t *testing.T) {
			t.Parallel()

			world := testutil.InitTestWorld(t)
			turnManager := turns.NewTurnManager()
			turnManager.TurnPhase = phase
			world.Resources.TurnManager = turnManager

			state := &DungeonState{}

			// UI系アクションを実行
			transition, err := state.DoAction(world, inputmapper.ActionOpenDungeonMenu)
			require.NoError(t, err, "DoActionがエラーを返しました")

			// どのターンフェーズでもTransPushを返すべき
			assert.Equal(t, es.TransPush, transition.Type, "トランジションタイプが不正")

			// NewStateFuncsが設定されているべき
			require.NotEmpty(t, transition.NewStateFuncs, "NewStateFuncsが空です")

			// ステートファクトリーが実際にステートを作成できることを検証
			newState := transition.NewStateFuncs[0]()
			require.NotNil(t, newState, "NewStateFunc が nil を返しました")

			// ステートが正しい型であることを検証
			assert.IsType(t, &PersistentMessageState{}, newState, "期待するステート型と異なります")
		})
	}
}

// setupTestWorldWithPlayer はプレイヤー付きのテスト用Worldを初期化するヘルパー関数
func setupTestWorldWithPlayer(t *testing.T, x, y int) (w.World, ecs.Entity) {
	t.Helper()

	world := testutil.InitTestWorld(t)

	// ターン管理を初期化
	turnManager := turns.NewTurnManager()
	world.Resources.TurnManager = turnManager

	// マップサイズを設定（移動判定に必要）
	world.Resources.Dungeon.Level.TileWidth = 50
	world.Resources.Dungeon.Level.TileHeight = 50

	// プレイヤーエンティティを作成
	playerEntity := world.Manager.NewEntity()
	playerEntity.AddComponent(world.Components.Player, &gc.Player{})
	playerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{
		X: gc.Tile(x),
		Y: gc.Tile(y),
	})

	return world, playerEntity
}

func TestGenerateNextBridgeTypes(t *testing.T) {
	t.Parallel()

	t.Run("5の倍数でない階層では全橋がランダム", func(t *testing.T) {
		t.Parallel()

		seed := uint64(12345)
		currentDepth := 3 // 5の倍数でない

		bridgeTypes := generateNextBridgeTypes(seed, currentDepth)

		// 4つの橋全てに値が設定される
		require.Len(t, bridgeTypes, 4)
		assert.Contains(t, bridgeTypes, maptemplate.ExitIDMain)
		assert.Contains(t, bridgeTypes, maptemplate.ExitIDLeft)
		assert.Contains(t, bridgeTypes, maptemplate.ExitIDCenter)
		assert.Contains(t, bridgeTypes, maptemplate.ExitIDRight)

		// 全て利用可能なタイプのいずれか
		availableTypes := []consts.PlannerTypeName{
			consts.PlannerTypeNameSmallRoom,
			consts.PlannerTypeNameBigRoom,
			consts.PlannerTypeNameCave,
			consts.PlannerTypeNameRuins,
			consts.PlannerTypeNameForest,
		}

		for _, bridgeType := range bridgeTypes {
			assert.Contains(t, availableTypes, bridgeType)
		}
	})

	t.Run("5の倍数の階層ではExitIDLeftが街広場に固定", func(t *testing.T) {
		t.Parallel()

		seed := uint64(12345)
		currentDepth := 5 // 5の倍数

		bridgeTypes := generateNextBridgeTypes(seed, currentDepth)

		// 4つの橋全てに値が設定される
		require.Len(t, bridgeTypes, 4)

		// ExitIDLeftは街広場固定
		assert.Equal(t, consts.PlannerTypeNameTownPlaza, bridgeTypes[maptemplate.ExitIDLeft])

		// 他の橋はランダム
		availableTypes := []consts.PlannerTypeName{
			consts.PlannerTypeNameSmallRoom,
			consts.PlannerTypeNameBigRoom,
			consts.PlannerTypeNameCave,
			consts.PlannerTypeNameRuins,
			consts.PlannerTypeNameForest,
		}

		assert.Contains(t, availableTypes, bridgeTypes[maptemplate.ExitIDMain])
		assert.Contains(t, availableTypes, bridgeTypes[maptemplate.ExitIDCenter])
		assert.Contains(t, availableTypes, bridgeTypes[maptemplate.ExitIDRight])
	})

	t.Run("0階では街広場固定にならない", func(t *testing.T) {
		t.Parallel()

		seed := uint64(12345)
		currentDepth := 0 // 5の倍数だが0階

		bridgeTypes := generateNextBridgeTypes(seed, currentDepth)

		// ExitIDLeftは街広場ではない（ランダム）
		assert.NotEqual(t, consts.PlannerTypeNameTownPlaza, bridgeTypes[maptemplate.ExitIDLeft])
	})

	t.Run("同じseedと階層で同じ結果を返す", func(t *testing.T) {
		t.Parallel()

		seed := uint64(99999)
		currentDepth := 7

		bridgeTypes1 := generateNextBridgeTypes(seed, currentDepth)
		bridgeTypes2 := generateNextBridgeTypes(seed, currentDepth)

		// 同じ結果が得られる
		assert.Equal(t, bridgeTypes1, bridgeTypes2)
	})

	t.Run("異なるseedで異なる結果を返す可能性がある", func(t *testing.T) {
		t.Parallel()

		currentDepth := 3

		bridgeTypes1 := generateNextBridgeTypes(111, currentDepth)
		bridgeTypes2 := generateNextBridgeTypes(222, currentDepth)

		// 少なくとも1つは異なる可能性が高い（確率的テスト）
		// 完全一致しない確率は非常に高い
		assert.NotEqual(t, bridgeTypes1, bridgeTypes2)
	})

	t.Run("10の倍数でもExitIDLeftのみ街広場固定", func(t *testing.T) {
		t.Parallel()

		seed := uint64(12345)
		currentDepth := 10 // 10の倍数

		bridgeTypes := generateNextBridgeTypes(seed, currentDepth)

		// ExitIDLeftのみ街広場固定
		assert.Equal(t, consts.PlannerTypeNameTownPlaza, bridgeTypes[maptemplate.ExitIDLeft])

		// 他の橋はランダム
		availableTypes := []consts.PlannerTypeName{
			consts.PlannerTypeNameSmallRoom,
			consts.PlannerTypeNameBigRoom,
			consts.PlannerTypeNameCave,
			consts.PlannerTypeNameRuins,
			consts.PlannerTypeNameForest,
		}

		assert.Contains(t, availableTypes, bridgeTypes[maptemplate.ExitIDMain])
		assert.Contains(t, availableTypes, bridgeTypes[maptemplate.ExitIDCenter])
		assert.Contains(t, availableTypes, bridgeTypes[maptemplate.ExitIDRight])
	})
}
