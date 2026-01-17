package turns

import (
	"testing"

	"github.com/kijimaD/ruins/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// テスト用のログレベル設定
	logger.SetConfig(logger.Config{
		DefaultLevel:   logger.LevelIgnore,
		CategoryLevels: map[logger.Category]logger.Level{},
	})
	m.Run()
}

func TestNewTurnManager(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	assert.Equal(t, 100, tm.PlayerMoves, "初期移動ポイントが正しく設定されている")
	assert.Equal(t, PlayerTurn, tm.TurnPhase, "初期ターンフェーズがPlayerTurn")
	assert.Equal(t, 1, tm.TurnNumber, "初期ターン番号が1")
	assert.True(t, tm.CanPlayerAct(), "初期状態でプレイヤーが行動可能")
}

func TestConsumePlayerMoves(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	// 移動アクション（コスト100）
	tm.ConsumePlayerMoves("Move", 100)

	assert.Equal(t, 0, tm.PlayerMoves, "移動後の移動ポイントが0")
	assert.Equal(t, AITurn, tm.TurnPhase, "移動ポイント消費後にAIターンに移行")
	assert.False(t, tm.CanPlayerAct(), "移動ポイント0でプレイヤーが行動不可")
}

func TestConsumePlayerMovesPartial(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	// アイテム拾得（コスト50）
	tm.ConsumePlayerMoves("PickupItem", 50)

	assert.Equal(t, 50, tm.PlayerMoves, "部分消費後の移動ポイントが50")
	assert.Equal(t, PlayerTurn, tm.TurnPhase, "移動ポイントが残っているのでPlayerTurnを継続")
	assert.True(t, tm.CanPlayerAct(), "移動ポイントが残っているのでプレイヤー行動可能")
}

func TestAdvanceToAITurn(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	tm.AdvanceToAITurn()

	assert.Equal(t, AITurn, tm.TurnPhase, "強制的にAIターンに移行")
	assert.False(t, tm.CanPlayerAct(), "AIターンでプレイヤー行動不可")
}

func TestAdvanceToTurnEnd(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()
	tm.AdvanceToAITurn()

	tm.AdvanceToTurnEnd()

	assert.Equal(t, TurnEnd, tm.TurnPhase, "ターン終了フェーズに移行")
}

func TestStartNewTurn(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	// ターンを進める
	tm.ConsumePlayerMoves("Move", 100) // PlayerTurn -> AITurn
	tm.AdvanceToTurnEnd()              // AITurn -> TurnEnd
	tm.StartNewTurn()                  // TurnEnd -> PlayerTurn（新ターン）

	assert.Equal(t, 2, tm.TurnNumber, "ターン番号が2に増加")
	assert.Equal(t, 100, tm.PlayerMoves, "新ターンで移動ポイントがリセット")
	assert.Equal(t, PlayerTurn, tm.TurnPhase, "新ターンでPlayerTurnに戻る")
	assert.True(t, tm.CanPlayerAct(), "新ターンでプレイヤーが行動可能")
}

func TestTurnCycle(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	// 完全なターンサイクルをテスト
	initialTurn := tm.TurnNumber

	// 1. プレイヤーアクション
	assert.True(t, tm.IsPlayerTurn())
	tm.ConsumePlayerMoves("Move", 100)

	// 2. AIターン
	assert.True(t, tm.IsAITurn())
	tm.AdvanceToTurnEnd()

	// 3. ターン終了
	assert.Equal(t, TurnEnd, tm.TurnPhase)
	tm.StartNewTurn()

	// 4. 新ターン開始
	assert.True(t, tm.IsPlayerTurn())
	assert.Equal(t, initialTurn+1, tm.TurnNumber)
}

func TestMultipleActionsInTurn(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	// 複数の軽いアクション
	tm.ConsumePlayerMoves("PickupItem", 50) // 50ポイント消費
	assert.True(t, tm.CanPlayerAct(), "まだ行動可能")
	assert.Equal(t, 50, tm.PlayerMoves)

	tm.ConsumePlayerMoves("PickupItem", 50) // さらに50ポイント消費
	assert.False(t, tm.CanPlayerAct(), "移動ポイント尽きて行動不可")
	assert.Equal(t, 0, tm.PlayerMoves)
	assert.True(t, tm.IsAITurn(), "AIターンに移行")
}

func TestWarpAction(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	// ワープアクション（コスト0）
	tm.ConsumePlayerMoves("Warp", 0)

	assert.Equal(t, 100, tm.PlayerMoves, "ワープは移動ポイント消費なし")
	assert.True(t, tm.CanPlayerAct(), "ワープ後も行動可能")
	assert.True(t, tm.IsPlayerTurn(), "ワープ後もPlayerTurn継続")
}
