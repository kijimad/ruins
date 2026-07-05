package tabmenu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVisibleItems(t *testing.T) {
	t.Parallel()

	items := make([]Item, 10)
	for i := range items {
		items[i] = Item{ID: string(rune('A' + i)), Label: string(rune('A' + i))}
	}

	t.Run("ページネーションありの場合は指定数だけ返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 3,
		}
		state := ViewState{TabIndex: 0, ItemIndex: 0}
		visible, indices := getVisibleItems(config, state)
		assert.Len(t, visible, 3)
		assert.Equal(t, []int{0, 1, 2}, indices)
	})

	t.Run("2ページ目のアイテムを返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 3,
		}
		state := ViewState{TabIndex: 0, ItemIndex: 4}
		visible, indices := getVisibleItems(config, state)
		assert.Len(t, visible, 3)
		assert.Equal(t, []int{3, 4, 5}, indices)
	})

	t.Run("ページネーションなしの場合は全件返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 0,
		}
		state := ViewState{TabIndex: 0, ItemIndex: 0}
		visible, _ := getVisibleItems(config, state)
		assert.Len(t, visible, 10)
	})

	t.Run("空タブの場合は空を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{}}
		state := ViewState{}
		visible, indices := getVisibleItems(config, state)
		assert.Empty(t, visible)
		assert.Empty(t, indices)
	})
}

func TestPageIndicatorText(t *testing.T) {
	t.Parallel()

	items := make([]Item, 5)
	for i := range items {
		items[i] = Item{ID: string(rune('A' + i))}
	}

	t.Run("複数ページの場合はテキストを返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 2,
		}
		state := ViewState{TabIndex: 0, ItemIndex: 0}
		text := pageIndicatorText(config, state)
		assert.Contains(t, text, "1/3")
	})

	t.Run("ページネーションなしの場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 0,
		}
		state := ViewState{TabIndex: 0, ItemIndex: 0}
		assert.Empty(t, pageIndicatorText(config, state))
	})

	t.Run("1ページの場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 10,
		}
		state := ViewState{TabIndex: 0, ItemIndex: 0}
		assert.Empty(t, pageIndicatorText(config, state))
	})
}

func TestTotalPages(t *testing.T) {
	t.Parallel()

	t.Run("ページネーションなしは1ページ", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{{ID: "t1", Items: make([]Item, 5)}}, ItemsPerPage: 0}
		assert.Equal(t, 1, totalPages(config, ViewState{}))
	})

	t.Run("空タブは1ページ", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{}}
		assert.Equal(t, 1, totalPages(config, ViewState{}))
	})

	t.Run("10件で3件ずつは4ページ", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{{ID: "t1", Items: make([]Item, 10)}}, ItemsPerPage: 3}
		assert.Equal(t, 4, totalPages(config, ViewState{}))
	})
}
