package gamelog

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 新しい機能のテスト
func TestSafeSliceMaxSize(t *testing.T) {
	t.Parallel()
	t.Run("最大サイズを超えた場合、古い要素が削除される", func(t *testing.T) {
		t.Parallel()
		// 最大サイズ3のSafeSliceを作成
		sl := NewSafeSlice(3)

		// 3つの要素を追加
		sl.Push("message1")
		sl.Push("message2")
		sl.Push("message3")

		content := sl.GetHistory()
		assert.Equal(t, 3, len(content))
		assert.Equal(t, []string{"message1", "message2", "message3"}, content)

		// 4つ目を追加（最古の要素が削除されるはず）
		sl.Push("message4")

		content = sl.GetHistory()
		assert.Equal(t, 3, len(content))
		// 最古の要素（message1）が削除され、message2, message3, message4が残るはず
		assert.Equal(t, []string{"message2", "message3", "message4"}, content)
	})

	t.Run("0サイズ指定時はデフォルトサイズが使用される", func(t *testing.T) {
		t.Parallel()
		sl := NewSafeSlice(0)

		// デフォルトサイズまで追加
		for i := 0; i < DefaultMaxLogSize+10; i++ {
			sl.Push(fmt.Sprintf("message%d", i))
		}

		content := sl.GetHistory()
		assert.Equal(t, DefaultMaxLogSize, len(content))
	})

	t.Run("負の値指定時はデフォルトサイズが使用される", func(t *testing.T) {
		t.Parallel()
		sl := NewSafeSlice(-5)

		// 少し多めに追加してテスト
		for i := 0; i < 20; i++ {
			sl.Push(fmt.Sprintf("message%d", i))
		}

		content := sl.GetHistory()
		assert.Equal(t, 20, len(content)) // デフォルトサイズ以下なので全部残る
	})
}

func TestSafeSliceMemoryLeak(t *testing.T) {
	t.Parallel()
	t.Run("大量の要素追加でもメモリリークしない", func(t *testing.T) {
		t.Parallel()
		sl := NewSafeSlice(10)

		// 大量の要素を追加
		for i := 0; i < 1000; i++ {
			sl.Push(fmt.Sprintf("message%d", i))
		}

		content := sl.GetHistory()
		// 最大サイズを超えないことを確認
		assert.Equal(t, 10, len(content))

		// 最新の10個が保持されていることを確認
		for i := 0; i < 10; i++ {
			expected := fmt.Sprintf("message%d", 990+i)
			assert.Equal(t, expected, content[i])
		}
	})
}

// ログAPIのテスト
func TestLogAPI(t *testing.T) {
	t.Parallel()

	t.Run("Push と GetRecent の基本動作", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(10)

		// メッセージを追加
		log.Push("古いメッセージ")
		log.Push("新しいメッセージ1")
		log.Push("新しいメッセージ2")
		log.Push("最新メッセージ")

		// 最新3行を取得
		recent := log.GetRecent(3)
		expected := []string{"新しいメッセージ1", "新しいメッセージ2", "最新メッセージ"}
		assert.Equal(t, expected, recent)

		// 全メッセージより多い行数を要求した場合
		all := log.GetRecent(10)
		expectedAll := []string{"古いメッセージ", "新しいメッセージ1", "新しいメッセージ2", "最新メッセージ"}
		assert.Equal(t, expectedAll, all)
	})

	t.Run("GetHistory は全履歴を表示順で取得", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(5)

		log.Push("msg1")
		log.Push("msg2")
		log.Push("msg3")

		history := log.GetHistory()
		expected := []string{"msg1", "msg2", "msg3"}
		assert.Equal(t, expected, history)
	})

	t.Run("Count と MaxHistory", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(5)

		assert.Equal(t, 0, log.Count())
		assert.Equal(t, 5, log.MaxHistory())

		log.Push("msg1")
		log.Push("msg2")
		assert.Equal(t, 2, log.Count())
	})

	t.Run("Clear は全ログを削除", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(10)

		log.Push("msg1")
		log.Push("msg2")
		assert.Equal(t, 2, log.Count())

		log.Clear()
		assert.Equal(t, 0, log.Count())
		assert.Equal(t, []string{}, log.GetHistory())
	})

	t.Run("表示順序の確認 - 下が新しい", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(10)

		log.Push("1番目（最古）")
		log.Push("2番目")
		log.Push("3番目")
		log.Push("4番目（最新）")

		// GetRecentで最新3行を取得
		recent := log.GetRecent(3)
		// 結果は [..., 3番目に新しい, 2番目に新しい, 最新] の順
		expected := []string{"2番目", "3番目", "4番目（最新）"}
		assert.Equal(t, expected, recent)

		// GetHistoryで全履歴を取得
		history := log.GetHistory()
		// 結果は [最古, ..., 2番目に新しい, 最新] の順
		expectedHistory := []string{"1番目（最古）", "2番目", "3番目", "4番目（最新）"}
		assert.Equal(t, expectedHistory, history)
	})

	t.Run("最大サイズ超過時の動作", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(3)

		log.Push("msg1")
		log.Push("msg2")
		log.Push("msg3")
		log.Push("msg4") // msg1が削除される

		assert.Equal(t, 3, log.Count())
		history := log.GetHistory()
		expected := []string{"msg2", "msg3", "msg4"}
		assert.Equal(t, expected, history)

		// GetRecentも正しく動作することを確認
		recent := log.GetRecent(2)
		expectedRecent := []string{"msg3", "msg4"}
		assert.Equal(t, expectedRecent, recent)
	})

	t.Run("空のログでのGetRecent", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(5)

		recent := log.GetRecent(3)
		assert.Equal(t, []string{}, recent)

		history := log.GetHistory()
		assert.Equal(t, []string{}, history)
	})
}

func TestSafeSliceColoredEntries(t *testing.T) {
	t.Parallel()

	t.Run("色付きエントリの基本動作", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(5)

		// 色付きエントリを直接追加
		entry := LogEntry{
			Fragments: []LogFragment{
				{Text: "Test message"},
			},
		}
		log.pushColoredEntry(entry)

		// 取得してテスト
		entries := log.GetRecentEntries(1)
		assert.Equal(t, 1, len(entries))
		assert.Equal(t, "Test message", entries[0].Text())
	})

	t.Run("空のLogEntry", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(3)

		// 空のエントリを追加
		emptyEntry := LogEntry{Fragments: []LogFragment{}}
		log.pushColoredEntry(emptyEntry)

		entries := log.GetRecentEntries(1)
		assert.Equal(t, 1, len(entries))
		assert.True(t, entries[0].IsEmpty())
		assert.Equal(t, "", entries[0].Text())
	})

	t.Run("色付きエントリの最大サイズ超過時の動作", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(2)

		// 3つのエントリを追加（最大サイズは2）
		for i := 1; i <= 3; i++ {
			entry := LogEntry{
				Fragments: []LogFragment{
					{Text: fmt.Sprintf("Message %d", i)},
				},
			}
			log.pushColoredEntry(entry)
		}

		entries := log.GetRecentEntries(5) // 全て取得を試行
		assert.Equal(t, 2, len(entries))   // 最大サイズ分のみ

		// 最新の2つが残っているかチェック
		assert.Equal(t, "Message 2", entries[0].Text())
		assert.Equal(t, "Message 3", entries[1].Text())
	})

	t.Run("GetHistoryEntriesの基本動作", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(5)

		// 複数エントリを追加
		for i := 1; i <= 3; i++ {
			entry := LogEntry{
				Fragments: []LogFragment{
					{Text: fmt.Sprintf("Entry %d", i)},
				},
			}
			log.pushColoredEntry(entry)
		}

		allEntries := log.GetHistoryEntries()
		assert.Equal(t, 3, len(allEntries))

		// 順序確認（古い順）
		assert.Equal(t, "Entry 1", allEntries[0].Text())
		assert.Equal(t, "Entry 2", allEntries[1].Text())
		assert.Equal(t, "Entry 3", allEntries[2].Text())
	})

	t.Run("GetRecentEntriesの負の値や0での呼び出し", func(t *testing.T) {
		t.Parallel()
		log := NewSafeSlice(5)

		entry := LogEntry{Fragments: []LogFragment{{Text: "Test"}}}
		log.pushColoredEntry(entry)

		// 負の値
		entries := log.GetRecentEntries(-1)
		assert.Equal(t, 0, len(entries))

		// 0
		entries = log.GetRecentEntries(0)
		assert.Equal(t, 0, len(entries))
	})
}
