package tabmenu

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagination_GetVisibleItems(t *testing.T) {
	t.Parallel()

	items := make([]Item, 10)
	for i := range items {
		items[i] = Item{ID: string(rune('A' + i)), Label: string(rune('A' + i))}
	}

	config := Config{
		Tabs:         []TabItem{{ID: "t1", Items: items}},
		ItemsPerPage: 3,
	}
	tm := newTabMenu(config, Callbacks{})

	visible, indices := tm.GetVisibleItems()
	assert.Len(t, visible, 3)
	assert.Equal(t, []int{0, 1, 2}, indices)
}

func TestPagination_PageNavigation(t *testing.T) {
	t.Parallel()

	items := make([]Item, 7)
	for i := range items {
		items[i] = Item{ID: string(rune('A' + i))}
	}

	config := Config{
		Tabs:         []TabItem{{ID: "t1", Items: items}},
		ItemsPerPage: 3,
	}
	tm := newTabMenu(config, Callbacks{})

	// 初期ページ
	assert.Equal(t, 1, tm.GetCurrentPage())
	assert.Equal(t, 3, tm.GetTotalPages())
	assert.False(t, tm.HasPreviousPage())
	assert.True(t, tm.HasNextPage())

	// 最後のアイテムまで移動して次のページへ
	for i := 0; i < 3; i++ {
		require.NoError(t, tm.DoAction(inputmapper.ActionMenuDown))
	}

	assert.Equal(t, 2, tm.GetCurrentPage())
	assert.True(t, tm.HasPreviousPage())
	assert.True(t, tm.HasNextPage())
}

func TestPagination_GetPageIndicatorText(t *testing.T) {
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
		tm := newTabMenu(config, Callbacks{})
		text := tm.GetPageIndicatorText()
		assert.Contains(t, text, "1/3")
	})

	t.Run("ページネーションなしの場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 0,
		}
		tm := newTabMenu(config, Callbacks{})
		assert.Empty(t, tm.GetPageIndicatorText())
	})

	t.Run("1ページの場合は空文字を返す", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:         []TabItem{{ID: "t1", Items: items}},
			ItemsPerPage: 10,
		}
		tm := newTabMenu(config, Callbacks{})
		assert.Empty(t, tm.GetPageIndicatorText())
	})
}

func TestUpdateTabs(t *testing.T) {
	t.Parallel()

	tabs := []TabItem{
		{ID: "t1", Items: []Item{{ID: "a"}, {ID: "b"}}},
		{ID: "t2", Items: []Item{{ID: "c"}}},
	}
	config := Config{Tabs: tabs, InitialTabIndex: 1}
	tm := newTabMenu(config, Callbacks{})

	// タブを更新して1つだけにする
	newTabs := []TabItem{
		{ID: "t1", Items: []Item{{ID: "a"}}},
	}
	tm.UpdateTabs(newTabs)
	assert.Equal(t, 0, tm.GetCurrentTabIndex(), "範囲外のタブインデックスは調整される")
	assert.Equal(t, 0, tm.GetCurrentItemIndex())
}

func TestUpdateTabs_EmptyTab(t *testing.T) {
	t.Parallel()

	tabs := []TabItem{
		{ID: "t1", Items: []Item{{ID: "a"}}},
	}
	config := Config{Tabs: tabs}
	tm := newTabMenu(config, Callbacks{})

	// アイテムなしのタブに更新
	newTabs := []TabItem{
		{ID: "t1", Items: []Item{}},
	}
	tm.UpdateTabs(newTabs)
	assert.Equal(t, -1, tm.GetCurrentItemIndex(), "空タブではアイテムインデックスが-1になる")
}

func TestEmptyTabsEdgeCases(t *testing.T) {
	t.Parallel()

	config := Config{Tabs: []TabItem{}}
	tm := newTabMenu(config, Callbacks{})

	// 空タブでの各操作がパニックしないことを確認
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuUp))
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuDown))
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuSelect))

	tab := tm.GetCurrentTab()
	assert.Empty(t, tab.ID)

	item := tm.GetCurrentItem()
	assert.Empty(t, item.ID)

	// 空タブでも最低1ページとして扱う
	assert.Equal(t, 1, tm.GetTotalPages())
}

func TestNewTabMenu_InitialItemIndexAdjustment(t *testing.T) {
	t.Parallel()

	t.Run("InitialItemIndexが範囲外の場合は調整される", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:             []TabItem{{ID: "t1", Items: []Item{{ID: "a"}}}},
			InitialItemIndex: 10,
		}
		tm := newTabMenu(config, Callbacks{})
		assert.Equal(t, 0, tm.GetCurrentItemIndex())
	})

	t.Run("負のInitialItemIndexは0に調整される", func(t *testing.T) {
		t.Parallel()
		config := Config{
			Tabs:             []TabItem{{ID: "t1", Items: []Item{{ID: "a"}}}},
			InitialItemIndex: -1,
		}
		tm := newTabMenu(config, Callbacks{})
		assert.Equal(t, 0, tm.GetCurrentItemIndex())
	})
}

func TestDoAction_UnknownAction(t *testing.T) {
	t.Parallel()

	config := Config{
		Tabs: []TabItem{{ID: "t1", Items: []Item{{ID: "a"}}}},
	}
	tm := newTabMenu(config, Callbacks{})

	// 不明なアクションはエラーなしで無視される
	require.NoError(t, tm.DoAction("unknown_action"))
}

func TestNoWrapNavigation(t *testing.T) {
	t.Parallel()

	tabs := []TabItem{
		{ID: "t1", Items: []Item{{ID: "a"}, {ID: "b"}}},
		{ID: "t2", Items: []Item{{ID: "c"}}},
	}
	config := Config{
		Tabs:           tabs,
		WrapNavigation: false,
	}
	tm := newTabMenu(config, Callbacks{})

	// 先頭で左移動しても先頭のまま
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuLeft))
	assert.Equal(t, 0, tm.GetCurrentTabIndex())

	// 先頭で上移動しても先頭のまま
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuUp))
	assert.Equal(t, 0, tm.GetCurrentItemIndex())

	// 最後のタブに移動
	require.NoError(t, tm.SetTabIndex(1))
	// 最後で右移動しても最後のまま
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuRight))
	assert.Equal(t, 1, tm.GetCurrentTabIndex())
}

func TestSelectCurrentItem_DisabledOrEmpty(t *testing.T) {
	t.Parallel()

	selected := false
	config := Config{
		Tabs: []TabItem{{ID: "t1", Items: []Item{}}},
	}
	callbacks := Callbacks{
		OnSelectItem: func(_, _ int, _ TabItem, _ Item) error {
			selected = true
			return nil
		},
	}
	tm := newTabMenu(config, callbacks)

	// 空タブで選択してもコールバックは呼ばれない
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuSelect))
	assert.False(t, selected)
}

func TestCancelWithoutCallback(t *testing.T) {
	t.Parallel()

	config := Config{
		Tabs: []TabItem{{ID: "t1", Items: []Item{{ID: "a"}}}},
	}
	// OnCancel未設定でもパニックしない
	tm := newTabMenu(config, Callbacks{})
	require.NoError(t, tm.DoAction(inputmapper.ActionMenuCancel))
}
