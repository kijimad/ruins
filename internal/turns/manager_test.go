package turns

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestTurnPhase_String(t *testing.T) {
	t.Parallel()

	t.Run("PlayerTurn", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "PlayerTurn", PlayerTurn.String())
	})

	t.Run("AITurn", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "AITurn", AITurn.String())
	})

	t.Run("TurnEnd", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "TurnEnd", TurnEnd.String())
	})

	t.Run("不正な値でpanicする", func(t *testing.T) {
		t.Parallel()
		invalidPhase := TurnPhase(999)
		assert.Panics(t, func() {
			_ = invalidPhase.String()
		})
	})
}

func TestConsumeActionPoints(t *testing.T) {
	t.Parallel()

	t.Run("TurnBasedコンポーネントがあればAPを消費", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		initialAP := turnBased.AP.Current

		ok := tm.ConsumeActionPoints(world, player, "テストアクション", 50)

		assert.True(t, ok)
		assert.Equal(t, initialAP-50, turnBased.AP.Current)
	})

	t.Run("TurnBasedコンポーネントがない場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Player, &gc.Player{})

		ok := tm.ConsumeActionPoints(world, entity, "テストアクション", 50)

		assert.False(t, ok)
	})
}

func TestCanEntityAct(t *testing.T) {
	t.Parallel()

	t.Run("APが0以上なら行動可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = 100

		assert.True(t, tm.CanEntityAct(world, player, 50))
	})

	t.Run("APが0でも行動可能", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = 0

		assert.True(t, tm.CanEntityAct(world, player, 50))
	})

	t.Run("APがマイナスなら行動不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = -1

		assert.False(t, tm.CanEntityAct(world, player, 50))
	})

	t.Run("プレイヤーはPlayerTurn以外で行動不可", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()
		tm.TurnPhase = AITurn

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = 100

		assert.False(t, tm.CanEntityAct(world, player, 50))
	})

	t.Run("TurnBasedがない場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		entity := world.Manager.NewEntity()
		entity.AddComponent(world.Components.Player, &gc.Player{})

		assert.False(t, tm.CanEntityAct(world, entity, 50))
	})
}

func TestRestoreAllActionPoints(t *testing.T) {
	t.Parallel()

	t.Run("全エンティティのAPが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = 0
		turnBased.AP.Max = 100

		err = tm.RestoreAllActionPoints(world)
		require.NoError(t, err)

		// Speed分だけ回復する
		assert.Greater(t, turnBased.AP.Current, 0)
	})

	t.Run("APはMax値を超えない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		turnBased := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		turnBased.AP.Current = 90
		turnBased.AP.Max = 100

		err = tm.RestoreAllActionPoints(world)
		require.NoError(t, err)

		assert.LessOrEqual(t, turnBased.AP.Current, turnBased.AP.Max)
	})

	t.Run("複数エンティティが回復する", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		tm := NewTurnManager()

		player, err := worldhelper.SpawnPlayer(world, 0, 0, "セレスティン")
		require.NoError(t, err)

		enemy, err := worldhelper.SpawnEnemy(world, 5, 5, "火の玉")
		require.NoError(t, err)

		playerTB := world.Components.TurnBased.Get(player).(*gc.TurnBased)
		playerTB.AP.Current = 0
		playerTB.AP.Max = 100

		enemyTB := world.Components.TurnBased.Get(enemy).(*gc.TurnBased)
		enemyTB.AP.Current = 0
		enemyTB.AP.Max = 100

		err = tm.RestoreAllActionPoints(world)
		require.NoError(t, err)

		assert.Greater(t, playerTB.AP.Current, 0)
		assert.Greater(t, enemyTB.AP.Current, 0)
	})
}

func TestConsumePlayerMovesNegative(t *testing.T) {
	t.Parallel()
	tm := NewTurnManager()

	// コストがPlayerMovesより大きい場合
	tm.ConsumePlayerMoves("HeavyAction", 150)

	assert.Equal(t, -50, tm.PlayerMoves, "移動ポイントがマイナスになる")
	assert.Equal(t, AITurn, tm.TurnPhase, "AIターンに移行")
}

func TestIsPlayerTurnAndIsAITurn(t *testing.T) {
	t.Parallel()

	t.Run("初期状態はPlayerTurn", func(t *testing.T) {
		t.Parallel()
		tm := NewTurnManager()
		assert.True(t, tm.IsPlayerTurn())
		assert.False(t, tm.IsAITurn())
	})

	t.Run("AdvanceToAITurn後はAITurn", func(t *testing.T) {
		t.Parallel()
		tm := NewTurnManager()
		tm.AdvanceToAITurn()
		assert.False(t, tm.IsPlayerTurn())
		assert.True(t, tm.IsAITurn())
	})

	t.Run("TurnEnd時はどちらもfalse", func(t *testing.T) {
		t.Parallel()
		tm := NewTurnManager()
		tm.AdvanceToAITurn()
		tm.AdvanceToTurnEnd()
		assert.False(t, tm.IsPlayerTurn())
		assert.False(t, tm.IsAITurn())
	})
}
