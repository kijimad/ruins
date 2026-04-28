package states

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
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
			name:           "未知のアクション",
			action:         inputmapper.ActionID("unknown"),
			expectedType:   es.TransNone,
			shouldHaveFunc: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			world := testutil.InitTestWorld(t)
			state := &DungeonState{}

			transition, err := state.DoAction(world, tt.action)
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

			initialX, initialY := 10, 10
			world := testutil.InitTestWorld(t)
			playerEntity, err := worldhelper.SpawnPlayer(world, initialX, initialY, "Ash")
			require.NoError(t, err)

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
		turnPhase        gc.TurnPhase
		expectedTrans    es.TransType
		isUIAction       bool
		isMoveAction     bool
		shouldMovePlayer bool
	}{
		{
			name:          "プレイヤーターン中のUI操作",
			action:        inputmapper.ActionOpenDungeonMenu,
			turnPhase:     gc.TurnPhasePlayer,
			expectedTrans: es.TransPush,
			isUIAction:    true,
		},
		{
			name:          "AIターン中のUI操作",
			action:        inputmapper.ActionOpenDungeonMenu,
			turnPhase:     gc.TurnPhaseAI,
			expectedTrans: es.TransPush,
			isUIAction:    true,
		},
		{
			name:             "プレイヤーターン中の移動",
			action:           inputmapper.ActionMoveNorth,
			turnPhase:        gc.TurnPhasePlayer,
			expectedTrans:    es.TransNone,
			isMoveAction:     true,
			shouldMovePlayer: true,
		},
		{
			name:             "AIターン中の移動（実行されない）",
			action:           inputmapper.ActionMoveNorth,
			turnPhase:        gc.TurnPhaseAI,
			expectedTrans:    es.TransNone,
			isMoveAction:     true,
			shouldMovePlayer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			initialX, initialY := 10, 10
			world := testutil.InitTestWorld(t)

			var playerEntity ecs.Entity
			if tt.isMoveAction {
				var err error
				playerEntity, err = worldhelper.SpawnPlayer(world, initialX, initialY, "Ash")
				require.NoError(t, err)
			}

			// ターン状態を設定
			turnState, err := worldhelper.GetTurnState(world)
			require.NoError(t, err)
			turnState.Phase = tt.turnPhase

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

// TestHandleMoveInput_Cardinal は通常移動で4方向のみ受け付けることを検証する
func TestHandleMoveInput_Cardinal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      ebiten.Key
		expected inputmapper.ActionID
	}{
		{"上", ebiten.KeyW, inputmapper.ActionMoveNorth},
		{"下", ebiten.KeyS, inputmapper.ActionMoveSouth},
		{"左", ebiten.KeyA, inputmapper.ActionMoveWest},
		{"右", ebiten.KeyD, inputmapper.ActionMoveEast},
		{"上矢印", ebiten.KeyUp, inputmapper.ActionMoveNorth},
		{"下矢印", ebiten.KeyDown, inputmapper.ActionMoveSouth},
		{"左矢印", ebiten.KeyLeft, inputmapper.ActionMoveWest},
		{"右矢印", ebiten.KeyRight, inputmapper.ActionMoveEast},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := input.NewMockKeyboardInput()
			mock.SetKeyPressedWithRepeat(tt.key, true)

			action, ok := handleMoveInput(mock)
			assert.True(t, ok)
			assert.Equal(t, tt.expected, action)
		})
	}
}

// TestHandleMoveInput_NoShiftNoDiagonal はShiftなしでは2キー同時押しでも斜め移動しないことを検証する
func TestHandleMoveInput_NoShiftNoDiagonal(t *testing.T) {
	t.Parallel()

	mock := input.NewMockKeyboardInput()
	mock.SetKeyPressedWithRepeat(ebiten.KeyW, true)
	mock.SetKeyPressedWithRepeat(ebiten.KeyA, true)

	action, ok := handleMoveInput(mock)
	assert.True(t, ok)
	// Shiftなしでは最初にマッチした方向（上）が返る
	assert.Equal(t, inputmapper.ActionMoveNorth, action)
}

// TestHandleShiftDiagonalInput は斜め移動の各方向を検証する
func TestHandleShiftDiagonalInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		heldKey   ebiten.Key
		repeatKey ebiten.Key
		expected  inputmapper.ActionID
	}{
		{"左上（W押しっぱなし+A）", ebiten.KeyW, ebiten.KeyA, inputmapper.ActionMoveNorthWest},
		{"右上（W押しっぱなし+D）", ebiten.KeyW, ebiten.KeyD, inputmapper.ActionMoveNorthEast},
		{"左下（S押しっぱなし+A）", ebiten.KeyS, ebiten.KeyA, inputmapper.ActionMoveSouthWest},
		{"右下（S押しっぱなし+D）", ebiten.KeyS, ebiten.KeyD, inputmapper.ActionMoveSouthEast},
		{"左上（A押しっぱなし+W）", ebiten.KeyA, ebiten.KeyW, inputmapper.ActionMoveNorthWest},
		{"右上（D押しっぱなし+W）", ebiten.KeyD, ebiten.KeyW, inputmapper.ActionMoveNorthEast},
		{"左下（A押しっぱなし+S）", ebiten.KeyA, ebiten.KeyS, inputmapper.ActionMoveSouthWest},
		{"右下（D押しっぱなし+S）", ebiten.KeyD, ebiten.KeyS, inputmapper.ActionMoveSouthEast},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mock := input.NewMockKeyboardInput()
			// 片方はHeld（押しっぱなし）、もう片方はRepeat（リピートタイミング）
			mock.SetKeyPressed(tt.heldKey, true)
			mock.SetKeyPressedWithRepeat(tt.repeatKey, true)

			action, ok := handleShiftDiagonalInput(mock)
			assert.True(t, ok)
			assert.Equal(t, tt.expected, action)
		})
	}
}

// TestHandleShiftDiagonalInput_SingleKey はShift中に1キーだけでは移動しないことを検証する
func TestHandleShiftDiagonalInput_SingleKey(t *testing.T) {
	t.Parallel()

	keys := []ebiten.Key{ebiten.KeyW, ebiten.KeyA, ebiten.KeyS, ebiten.KeyD}
	for _, key := range keys {
		t.Run(key.String(), func(t *testing.T) {
			t.Parallel()
			mock := input.NewMockKeyboardInput()
			mock.SetKeyPressed(key, true)
			mock.SetKeyPressedWithRepeat(key, true)

			_, ok := handleShiftDiagonalInput(mock)
			assert.False(t, ok, "1キーのみでは斜め移動しないべき")
		})
	}
}

// TestHandleMoveInput_ShiftDelegates はShift押下中にhandleShiftDiagonalInputへ委譲されることを検証する
func TestHandleMoveInput_ShiftDelegates(t *testing.T) {
	t.Parallel()

	mock := input.NewMockKeyboardInput()
	mock.SetKeyPressed(ebiten.KeyShiftLeft, true)
	mock.SetKeyPressed(ebiten.KeyW, true)
	mock.SetKeyPressedWithRepeat(ebiten.KeyA, true)

	action, ok := handleMoveInput(mock)
	assert.True(t, ok)
	assert.Equal(t, inputmapper.ActionMoveNorthWest, action)
}

// TestHandleMoveInput_ShiftSingleKeyNoAction はShift+単一キーでは移動しないことを検証する
func TestHandleMoveInput_ShiftSingleKeyNoAction(t *testing.T) {
	t.Parallel()

	mock := input.NewMockKeyboardInput()
	mock.SetKeyPressed(ebiten.KeyShiftLeft, true)
	mock.SetKeyPressedWithRepeat(ebiten.KeyW, true)

	_, ok := handleMoveInput(mock)
	assert.False(t, ok, "Shift+単一キーでは移動しないべき")
}

// TestDoActionUIActionsAlwaysWork はUI系アクションはターンフェーズに関わらず動作する
func TestDoActionUIActionsAlwaysWork(t *testing.T) {
	t.Parallel()

	turnPhases := []gc.TurnPhase{
		gc.TurnPhasePlayer,
		gc.TurnPhaseAI,
		gc.TurnPhaseEnd,
	}

	for _, phase := range turnPhases {
		t.Run(fmt.Sprintf("TurnPhase_%d", phase), func(t *testing.T) {
			t.Parallel()

			world := testutil.InitTestWorld(t)

			// ターン状態を設定
			turnState, err := worldhelper.GetTurnState(world)
			require.NoError(t, err)
			turnState.Phase = phase

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
