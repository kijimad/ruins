package messagewindow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVisibleItems(t *testing.T) {
	t.Parallel()

	items := make([]item, 10)
	for i := range items {
		items[i] = item{ID: string(rune('A' + i)), Label: string(rune('A' + i))}
	}

	t.Run("ページネーションありの場合は指定数だけ返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 3,
		}
		state := viewState{TabIndex: 0, ItemIndex: 0}
		visible, indices := getVisibleItems(config, state)
		assert.Len(t, visible, 3)
		assert.Equal(t, []int{0, 1, 2}, indices)
	})

	t.Run("2ページ目のアイテムを返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 3,
		}
		state := viewState{TabIndex: 0, ItemIndex: 4}
		visible, indices := getVisibleItems(config, state)
		assert.Len(t, visible, 3)
		assert.Equal(t, []int{3, 4, 5}, indices)
	})

	t.Run("ページネーションなしの場合は全件返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 0,
		}
		state := viewState{TabIndex: 0, ItemIndex: 0}
		visible, _ := getVisibleItems(config, state)
		assert.Len(t, visible, 10)
	})

	t.Run("空タブの場合は空を返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{Tabs: []tabItem{}}
		state := viewState{}
		visible, indices := getVisibleItems(config, state)
		assert.Empty(t, visible)
		assert.Empty(t, indices)
	})

	t.Run("開始位置が総アイテム数以上の場合は空を返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 3,
		}
		state := viewState{TabIndex: 0, ItemIndex: 100}
		visible, indices := getVisibleItems(config, state)
		assert.Empty(t, visible)
		assert.Empty(t, indices)
	})
}

func TestPageIndicatorText(t *testing.T) {
	t.Parallel()

	items := make([]item, 5)
	for i := range items {
		items[i] = item{ID: string(rune('A' + i))}
	}

	t.Run("複数ページの場合はテキストを返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 2,
		}
		state := viewState{TabIndex: 0, ItemIndex: 0}
		text := pageIndicatorText(config, state)
		assert.Contains(t, text, "1/3")
	})

	t.Run("ページネーションなしの場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 0,
		}
		state := viewState{TabIndex: 0, ItemIndex: 0}
		assert.Empty(t, pageIndicatorText(config, state))
	})

	t.Run("1ページの場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{
			Tabs:         []tabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 10,
		}
		state := viewState{TabIndex: 0, ItemIndex: 0}
		assert.Empty(t, pageIndicatorText(config, state))
	})
}

func TestTotalPages(t *testing.T) {
	t.Parallel()

	t.Run("ページネーションなしは1ページ", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{Tabs: []tabItem{{ID: "t1", Items: make([]item, 5)}}, ItemsPerPage: 0}
		assert.Equal(t, 1, totalPages(config, viewState{}))
	})

	t.Run("空タブは1ページ", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{Tabs: []tabItem{}}
		assert.Equal(t, 1, totalPages(config, viewState{}))
	})

	t.Run("10件で3件ずつは4ページ", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{Tabs: []tabItem{{ID: "t1", Items: make([]item, 10)}}, ItemsPerPage: 3}
		assert.Equal(t, 4, totalPages(config, viewState{}))
	})

	t.Run("TabIndexがタブ数以上は1ページ", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{Tabs: []tabItem{{ID: "t1", Items: make([]item, 10)}}, ItemsPerPage: 3}
		assert.Equal(t, 1, totalPages(config, viewState{TabIndex: 5}))
	})
}

func TestCurrentPage(t *testing.T) {
	t.Parallel()

	items := make([]item, 10)
	for i := range items {
		items[i] = item{ID: string(rune('A' + i)), Label: string(rune('A' + i))}
	}
	paged := tabMenuConfig{Tabs: []tabItem{{ID: "t1", Items: items}}, ItemsPerPage: 3}

	t.Run("ItemsPerPageが0以下は0を返す", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{Tabs: []tabItem{{ID: "t1", Items: items}}, ItemsPerPage: 0}
		assert.Equal(t, 0, currentPage(config, viewState{ItemIndex: 5}))
	})

	t.Run("ItemIndexが負は0を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0, currentPage(paged, viewState{ItemIndex: -1}))
	})

	t.Run("ItemIndexをItemsPerPageで割った値を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 2, currentPage(paged, viewState{ItemIndex: 7}))
	})

	t.Run("項目が無ければページは0にとどまる", func(t *testing.T) {
		t.Parallel()
		config := tabMenuConfig{ItemsPerPage: 3}
		assert.Equal(t, 0, currentPage(config, viewState{ItemIndex: 7}), "存在しない項目の位置でページを進めない")
	})
}
