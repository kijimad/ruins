package gamelog

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
)

// NewLoggerWithTestStore はテスト用ストアを使用するLoggerを作成
func NewLoggerWithTestStore() (*Logger, *SafeSlice) {
	store := NewSafeSlice(FieldLogMaxSize)
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
	if store.Count() != 1 {
		t.Errorf("Expected 1 log entry, got %d", store.Count())
	}

	// テキストの内容をチェック
	messages := store.GetRecent(1)
	expected := "Player attacks Goblin for 15 damage!"
	if messages[0] != expected {
		t.Errorf("Expected '%s', got '%s'", expected, messages[0])
	}

	// 色付きエントリもチェック
	entries := store.GetRecentEntries(1)
	if len(entries) != 1 {
		t.Errorf("Expected 1 colored entry, got %d", len(entries))
	}

	entry := entries[0]
	if len(entry.Fragments) != 6 {
		t.Errorf("Expected 6 fragments, got %d", len(entry.Fragments))
	}

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
		if entry.Fragments[i].Text != expected.text {
			t.Errorf("Fragment %d: expected text '%s', got '%s'", i, expected.text, entry.Fragments[i].Text)
		}
	}

	// NPCの名前が黄色、ダメージが赤色かチェック
	if entry.Fragments[2].Color != consts.ColorYellow {
		t.Errorf("Expected NPC name to be yellow, got %v", entry.Fragments[2].Color)
	}
	if entry.Fragments[4].Color != consts.ColorRed {
		t.Errorf("Expected damage to be red, got %v", entry.Fragments[4].Color)
	}
}

func TestLoggerColorMethod(t *testing.T) {
	t.Parallel()
	logger, store := NewLoggerWithTestStore()

	// カスタム色での使用例
	logger.
		ColorRGBA(consts.ColorCyan). // Cyan
		Append("John").
		ColorRGBA(consts.ColorWhite).
		Append(" considers attacking ").
		ColorRGBA(consts.ColorCyan).
		Append("Orc").
		Log()

	entries := store.GetRecentEntries(1)
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(entries))
	}

	fragments := entries[0].Fragments
	if len(fragments) != 3 {
		t.Errorf("Expected 3 fragments, got %d", len(fragments))
	}

	// 色のチェック
	if fragments[0].Color != consts.ColorCyan {
		t.Errorf("Expected first fragment to be cyan")
	}
	if fragments[1].Color != consts.ColorWhite {
		t.Errorf("Expected second fragment to be white")
	}
	if fragments[2].Color != consts.ColorCyan {
		t.Errorf("Expected third fragment to be cyan")
	}
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

	if fragments[1].Color != consts.ColorCyan {
		t.Errorf("Expected item name to be cyan")
	}
	if fragments[1].Text != "Iron Sword" {
		t.Errorf("Expected item name 'Iron Sword', got '%s'", fragments[1].Text)
	}
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

	if fragments[0].Color != consts.ColorGreen {
		t.Errorf("Expected player name to be green")
	}
	if fragments[0].Text != "Hero" {
		t.Errorf("Expected player name 'Hero', got '%s'", fragments[0].Text)
	}
}

func TestLoggerMultipleLogs(t *testing.T) {
	t.Parallel()
	logger, store := NewLoggerWithTestStore()

	// 複数のログを追加
	logger.Append("First message").Log()
	logger.Append("Second message").Log()
	logger.NPCName("Enemy").Append(" appears!").Log()

	if store.Count() != 3 {
		t.Errorf("Expected 3 log entries, got %d", store.Count())
	}

	entries := store.GetRecentEntries(3)
	if len(entries) != 3 {
		t.Errorf("Expected 3 colored entries, got %d", len(entries))
	}

	// 最後のエントリをチェック
	lastEntry := entries[2]
	if len(lastEntry.Fragments) != 2 {
		t.Errorf("Expected 2 fragments in last entry, got %d", len(lastEntry.Fragments))
	}
	if lastEntry.Fragments[0].Color != consts.ColorYellow {
		t.Errorf("Expected enemy name to be yellow")
	}
}

func TestLoggerEmptyFragments(t *testing.T) {
	t.Parallel()

	t.Run("空のフラグメントでLogを呼んでも何も追加されない", func(t *testing.T) {
		t.Parallel()
		logger, store := NewLoggerWithTestStore()

		// フラグメントを追加せずにLogを呼ぶ
		logger.Log()

		if store.Count() != 0 {
			t.Errorf("Expected 0 log entries when logging empty fragments, got %d", store.Count())
		}
	})

	t.Run("フラグメント追加後にLogし、再度空の状態でLogを呼ぶ", func(t *testing.T) {
		t.Parallel()
		logger, store := NewLoggerWithTestStore()

		// 最初にフラグメントを追加してLog
		logger.Append("Test message").Log()

		if store.Count() != 1 {
			t.Errorf("Expected 1 log entry after first log, got %d", store.Count())
		}

		// 空の状態で再度Log
		logger.Log()

		// カウントは変わらない
		if store.Count() != 1 {
			t.Errorf("Expected 1 log entry after empty log, got %d", store.Count())
		}
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

		if store.Count() != 2 {
			t.Errorf("Expected 2 log entries, got %d", store.Count())
		}

		messages := store.GetRecent(2)
		// GetRecentは時系列順で返す（古い順）
		expected := []string{"First", "Second"}
		for i, exp := range expected {
			if i >= len(messages) {
				t.Errorf("Message index %d out of range, got %d messages", i, len(messages))
				continue
			}
			if messages[i] != exp {
				t.Errorf("Message %d: expected '%s', got '%s'", i, exp, messages[i])
			}
		}
	})
}
