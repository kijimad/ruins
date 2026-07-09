package gamelog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogEntry_Text(t *testing.T) {
	t.Parallel()

	t.Run("複数フラグメントのテキスト結合", func(t *testing.T) {
		t.Parallel()
		entry := LogEntry{
			Fragments: []LogFragment{
				{Color: ColorWhite, Text: "Hello "},
				{Color: ColorRed, Text: "World"},
				{Color: ColorWhite, Text: "!"},
			},
		}

		expected := "Hello World!"
		actual := entry.Text()

		assert.Equal(t, expected, actual)
	})

	t.Run("単一フラグメントのテキスト", func(t *testing.T) {
		t.Parallel()
		entry := LogEntry{
			Fragments: []LogFragment{
				{Color: ColorGreen, Text: "Success"},
			},
		}

		expected := "Success"
		actual := entry.Text()

		assert.Equal(t, expected, actual)
	})

	t.Run("空のフラグメントリスト", func(t *testing.T) {
		t.Parallel()
		entry := LogEntry{
			Fragments: []LogFragment{},
		}

		expected := ""
		actual := entry.Text()

		assert.Equal(t, expected, actual)
	})
}

func TestLogEntry_IsEmpty(t *testing.T) {
	t.Parallel()

	t.Run("空のエントリ", func(t *testing.T) {
		t.Parallel()
		entry := LogEntry{
			Fragments: []LogFragment{},
		}

		assert.True(t, entry.IsEmpty(), "Expected entry to be empty")
	})

	t.Run("nilフラグメント", func(t *testing.T) {
		t.Parallel()
		entry := LogEntry{
			Fragments: nil,
		}

		assert.True(t, entry.IsEmpty(), "Expected entry with nil fragments to be empty")
	})

	t.Run("非空のエントリ", func(t *testing.T) {
		t.Parallel()
		entry := LogEntry{
			Fragments: []LogFragment{
				{Color: ColorWhite, Text: "Not empty"},
			},
		}

		assert.False(t, entry.IsEmpty(), "Expected entry to not be empty")
	})

	t.Run("空テキストのフラグメントがある場合", func(t *testing.T) {
		t.Parallel()
		entry := LogEntry{
			Fragments: []LogFragment{
				{Color: ColorWhite, Text: ""},
			},
		}

		// フラグメントが存在するので空ではない
		assert.False(t, entry.IsEmpty(), "Expected entry with empty text fragment to not be empty")
	})
}
