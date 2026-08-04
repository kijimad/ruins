package tabmenu

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("開始位置が総アイテム数以上の場合は空を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 3,
		}
		state := ViewState{TabIndex: 0, ItemIndex: 100}
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

	t.Run("TabIndexがタブ数以上は1ページ", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{{ID: "t1", Items: make([]Item, 10)}}, ItemsPerPage: 3}
		assert.Equal(t, 1, totalPages(config, ViewState{TabIndex: 5}))
	})
}

func TestCurrentPage(t *testing.T) {
	t.Parallel()

	t.Run("ItemsPerPageが0以下は0を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{ItemsPerPage: 0}
		assert.Equal(t, 0, currentPage(config, ViewState{ItemIndex: 5}))
	})

	t.Run("ItemIndexが負は0を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{ItemsPerPage: 3}
		assert.Equal(t, 0, currentPage(config, ViewState{ItemIndex: -1}))
	})

	t.Run("ItemIndexをItemsPerPageで割った値を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{ItemsPerPage: 3}
		assert.Equal(t, 2, currentPage(config, ViewState{ItemIndex: 7}))
	})
}

// TestComputeDisplayRows は表示行の計算を ebitenui widget を作らずに直接検証する。
// UpdateTabDisplayContainer の配線ロジックはここで純粋にテストし、行→widget の見た目は golden が見る。
func TestComputeDisplayRows(t *testing.T) {
	t.Parallel()

	t.Run("ページネーションありなら先頭にインジケーター行、続けて可視アイテム行", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs: []TabItem{{ID: "t", Items: []Item{
				{ID: "i1", Label: "A"}, {ID: "i2", Label: "B"},
				{ID: "i3", Label: "C"}, {ID: "i4", Label: "D"},
			}}},
			ItemsPerPage: 2,
		}
		rows := computeDisplayRows(config, ViewState{TabIndex: 0, ItemIndex: 0})

		require.Len(t, rows, 3)
		assert.Equal(t, displayPageIndicator, rows[0].Kind)
		assert.Equal(t, displayItem, rows[1].Kind)
		assert.Equal(t, "A", rows[1].Label)
		assert.Equal(t, displayItem, rows[2].Kind)
		assert.Equal(t, "B", rows[2].Label)
	})

	t.Run("ページネーションなしならインジケーター行は入らない", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{{ID: "t", Items: []Item{{ID: "i1", Label: "A"}, {ID: "i2", Label: "B"}}}}}
		rows := computeDisplayRows(config, ViewState{TabIndex: 0, ItemIndex: 0})

		require.Len(t, rows, 2)
		assert.Equal(t, displayItem, rows[0].Kind)
		assert.Equal(t, displayItem, rows[1].Kind)
	})

	t.Run("アイテムが空なら空表示の行が1つ入る", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{{ID: "t", Items: []Item{}}}}
		rows := computeDisplayRows(config, ViewState{TabIndex: 0, ItemIndex: 0})

		require.Len(t, rows, 1)
		assert.Equal(t, displayEmptyPlaceholder, rows[0].Kind)
	})

	t.Run("選択中のアイテム行だけ Selected=true", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{{ID: "t", Items: []Item{{ID: "i1", Label: "A"}, {ID: "i2", Label: "B"}}}}}
		rows := computeDisplayRows(config, ViewState{TabIndex: 0, ItemIndex: 1})

		require.Len(t, rows, 2)
		assert.False(t, rows[0].Selected)
		assert.True(t, rows[1].Selected)
	})

	t.Run("ItemIndex が負ならどのアイテム行も非選択", func(t *testing.T) {
		t.Parallel()
		config := Config{Tabs: []TabItem{{ID: "t", Items: []Item{{ID: "i1", Label: "A"}, {ID: "i2", Label: "B"}}}}}
		rows := computeDisplayRows(config, ViewState{TabIndex: 0, ItemIndex: -1})

		require.Len(t, rows, 2)
		assert.False(t, rows[0].Selected)
		assert.False(t, rows[1].Selected)
	})
}
