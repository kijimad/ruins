package gamelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewLoggerWithTestStore はテスト用ストアを使用するLoggerを作成
func NewLoggerWithTestStore() (*Logger, *SafeSlice) {
	store := NewSafeSlice(GameLogMaxSize)
	logger := New(store)
	return logger, store
}

func TestLoggerBasicUsage(t *testing.T) {
	t.Parallel()
	logger, store := NewLoggerWithTestStore()

	// メソッドチェーンでのログ作成をテスト
	logger.
		Append("Player").
		Append(" attacks ").
		NPCName("Goblin").
		Append(" for ").
		Damage(15).
		Append(" damage!").
		Log()

	// ログが追加されたかチェック
	assert.Equal(t, 1, store.Count(), "Expected 1 log entry")

	// テキストの内容をチェック
	messages := store.GetRecent(1)
	expected := "Player attacks Goblin for 15 damage!"
	assert.Equal(t, expected, messages[0])

	// 色付きエントリもチェック
	entries := store.GetRecentEntries(1)
	require.Len(t, entries, 1, "Expected 1 colored entry")

	entry := entries[0]
	require.Len(t, entry.Fragments, 6, "Expected 6 fragments")

	// 各フラグメントをチェック
	expectedFragments := []struct {
		text  string
		color string
	}{
		{"Player", "white"},
		{" attacks ", "white"},
		{"Goblin", "yellow"},
		{" for ", "white"},
		{"15", "red"},
		{" damage!", "white"},
	}

	for i, expected := range expectedFragments {
		assert.Equal(t, expected.text, entry.Fragments[i].Text, "Fragment %d text", i)
	}

	// NPCの名前が黄色、ダメージが赤色かチェック
	assert.Equal(t, ColorYellow, entry.Fragments[2].Color, "Expected NPC name to be yellow")
	assert.Equal(t, ColorRed, entry.Fragments[4].Color, "Expected damage to be red")
}

func TestLoggerColorMethod(t *testing.T) {
	t.Parallel()
	logger, store := NewLoggerWithTestStore()

	// カスタム色での使用例
	logger.
		ColorRGBA(ColorCyan). // Cyan
		Append("John").
		ColorRGBA(ColorWhite).
		Append(" considers attacking ").
		ColorRGBA(ColorCyan).
		Append("Orc").
		Log()

	entries := store.GetRecentEntries(1)
	require.Len(t, entries, 1, "Expected 1 entry")

	fragments := entries[0].Fragments
	require.Len(t, fragments, 3, "Expected 3 fragments")

	// 色のチェック
	assert.Equal(t, ColorCyan, fragments[0].Color, "Expected first fragment to be cyan")
	assert.Equal(t, ColorWhite, fragments[1].Color, "Expected second fragment to be white")
	assert.Equal(t, ColorCyan, fragments[2].Color, "Expected third fragment to be cyan")
}

func TestLoggerItemName(t *testing.T) {
	t.Parallel()
	logger, store := NewLoggerWithTestStore()

	logger.
		Append("You pick up ").
		ItemName("Iron Sword").
		Append(".").
		Log()

	entries := store.GetRecentEntries(1)
	fragments := entries[0].Fragments

	assert.Equal(t, ColorCyan, fragments[1].Color, "Expected item name to be cyan")
	assert.Equal(t, "Iron Sword", fragments[1].Text, "Expected item name 'Iron Sword'")
}

func TestLoggerPlayerName(t *testing.T) {
	t.Parallel()
	logger, store := NewLoggerWithTestStore()

	logger.
		PlayerName("Hero").
		Append(" enters the dungeon").
		Log()

	entries := store.GetRecentEntries(1)
	fragments := entries[0].Fragments

	assert.Equal(t, ColorGreen, fragments[0].Color, "Expected player name to be green")
	assert.Equal(t, "Hero", fragments[0].Text, "Expected player name 'Hero'")
}

func TestLoggerMultipleLogs(t *testing.T) {
	t.Parallel()
	logger, store := NewLoggerWithTestStore()

	// 複数のログを追加
	logger.Append("First message").Log()
	logger.Append("Second message").Log()
	logger.NPCName("Enemy").Append(" appears!").Log()

	assert.Equal(t, 3, store.Count(), "Expected 3 log entries")

	entries := store.GetRecentEntries(3)
	require.Len(t, entries, 3, "Expected 3 colored entries")

	// 最後のエントリをチェック
	lastEntry := entries[2]
	require.Len(t, lastEntry.Fragments, 2, "Expected 2 fragments in last entry")
	assert.Equal(t, ColorYellow, lastEntry.Fragments[0].Color, "Expected enemy name to be yellow")
}

func TestLoggerEmptyFragments(t *testing.T) {
	t.Parallel()

	t.Run("空のフラグメントでLogを呼んでも何も追加されない", func(t *testing.T) {
		t.Parallel()
		logger, store := NewLoggerWithTestStore()

		// フラグメントを追加せずにLogを呼ぶ
		logger.Log()

		assert.Equal(t, 0, store.Count(), "Expected 0 log entries when logging empty fragments")
	})

	t.Run("フラグメント追加後にLogし、再度空の状態でLogを呼ぶ", func(t *testing.T) {
		t.Parallel()
		logger, store := NewLoggerWithTestStore()

		// 最初にフラグメントを追加してLog
		logger.Append("Test message").Log()

		assert.Equal(t, 1, store.Count(), "Expected 1 log entry after first log")

		// 空の状態で再度Log
		logger.Log()

		// カウントは変わらない
		assert.Equal(t, 1, store.Count(), "Expected 1 log entry after empty log")
	})

	t.Run("同じLoggerインスタンスでの複数回ログ出力", func(t *testing.T) {
		t.Parallel()
		logger, store := NewLoggerWithTestStore()

		// 1回目
		logger.Append("First").Log()
		// 2回目
		logger.Append("Second").Log()
		// 3回目 - 空
		logger.Log()

		assert.Equal(t, 2, store.Count(), "Expected 2 log entries")

		messages := store.GetRecent(2)
		// GetRecentは時系列順で返す（古い順）
		expected := []string{"First", "Second"}
		require.Len(t, messages, len(expected), "Expected %d messages", len(expected))
		for i, exp := range expected {
			assert.Equal(t, exp, messages[i], "Message %d", i)
		}
	})
}
