package hooks

import (
	"testing"

	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/stretchr/testify/assert"
)

type tabMenuTestProps struct {
	Tabs []testTabData
}

type testTabData struct {
	ID    string
	Items []string
}

// setupTabMenuState はタブメニュー用のUseTabMenuを登録する
func setupTabMenuState(t *testing.T, store *Store, p tabMenuTestProps) {
	t.Helper()
	itemCounts := make([]int, len(p.Tabs))
	for i, tab := range p.Tabs {
		itemCounts[i] = len(tab.Items)
	}

	UseTabMenu(store, "menu", TabMenuConfig{
		TabCount:   len(p.Tabs),
		ItemCounts: itemCounts,
	})
}

func TestUseTabMenu_InitialState(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c"}},
			{ID: "tab2", Items: []string{"x", "y"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.TabIndex)
	assert.Equal(t, 0, s.ItemIndex)
}

func TestUseTabMenu_TabNavigation(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b"}},
			{ID: "tab2", Items: []string{"x", "y"}},
			{ID: "tab3", Items: []string{"1", "2"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	// 右に移動
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 1, s.TabIndex)

	// さらに右に移動
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 2, s.TabIndex)

	// 循環して最初に戻る
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.TabIndex, "最後のタブから右に移動すると最初のタブに循環する")
}

func TestUseTabMenu_TabNavigationLeft(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b"}},
			{ID: "tab2", Items: []string{"x", "y"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	// 最初のタブで左に移動すると最後のタブに循環
	mount.Dispatch(inputmapper.ActionMenuTabPrev)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 1, s.TabIndex, "最初のタブから左に移動すると最後のタブに循環する")
}

func TestUseTabMenu_ItemNavigation(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	// 下に移動
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 1, s.ItemIndex)

	// さらに下に移動
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 2, s.ItemIndex)

	// 循環して最初に戻る
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.ItemIndex, "最後のアイテムから下に移動すると最初に循環する")
}

func TestUseTabMenu_ItemNavigationUp(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	// 最初のアイテムで上に移動すると最後に循環
	mount.Dispatch(inputmapper.ActionMenuUp)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 2, s.ItemIndex, "最初のアイテムから上に移動すると最後に循環する")
}

func TestUseTabMenu_TabChangeResetsItemIndex(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c"}},
			{ID: "tab2", Items: []string{"x", "y"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	// アイテムを選択
	mount.Dispatch(inputmapper.ActionMenuDown)
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 2, s.ItemIndex)

	// タブを切り替え
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.ItemIndex, "タブ切り替え時にアイテムインデックスがリセットされる")
}

func TestUseTabMenu_EmptyTabs(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.TabIndex)
	assert.Equal(t, 0, s.ItemIndex)

	// 操作しても変わらない
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.TabIndex)
}

func TestUseTabMenu_EmptyItems(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)

	mount.Update()

	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.ItemIndex)

	// 操作しても変わらない
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.ItemIndex)
}

func TestUseTabMenu_PropsChange(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()

	// 最初は3タブ
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a"}},
			{ID: "tab2", Items: []string{"b"}},
			{ID: "tab3", Items: []string{"c"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()

	// 2番目のタブに移動
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupTabMenuState(t, mount.Store(), props)
	mount.Update()
	s, _ := GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 1, s.TabIndex)

	// タブが2つに減る
	newProps := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a"}},
			{ID: "tab2", Items: []string{"b"}},
		},
	}
	mount.SetProps(newProps)
	setupTabMenuState(t, mount.Store(), newProps)
	mount.Update()

	// 右に移動すると2タブなので循環して0になる
	// tabIndex=1, tabCount=2 → (1+1) % 2 = 0
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupTabMenuState(t, mount.Store(), newProps)
	mount.Update()
	s, _ = GetState[TabMenuState](mount, "menu")
	assert.Equal(t, 0, s.TabIndex, "Propsが変わると新しいタブ数で循環する")
}

func TestUseTabMenu_MultipleInstances(t *testing.T) {
	t.Parallel()

	setupMultiMenu := func(store *Store, p tabMenuTestProps) {
		itemCounts := make([]int, len(p.Tabs))
		for i, tab := range p.Tabs {
			itemCounts[i] = len(tab.Items)
		}

		UseTabMenu(store, "menu1", TabMenuConfig{
			TabCount:   len(p.Tabs),
			ItemCounts: itemCounts,
		})

		UseTabMenu(store, "menu2", TabMenuConfig{
			TabCount:   len(p.Tabs),
			ItemCounts: itemCounts,
		})
	}

	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b"}},
			{ID: "tab2", Items: []string{"x", "y"}},
		},
	}
	mount.SetProps(props)
	setupMultiMenu(mount.Store(), props)

	mount.Update()

	// 両方独立して状態を持つ
	s1, _ := GetState[TabMenuState](mount, "menu1")
	s2, _ := GetState[TabMenuState](mount, "menu2")
	assert.Equal(t, 0, s1.TabIndex)
	assert.Equal(t, 0, s2.TabIndex)

	// 同じDispatchで両方更新される
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	setupMultiMenu(mount.Store(), props)
	mount.Update()
	s1, _ = GetState[TabMenuState](mount, "menu1")
	s2, _ = GetState[TabMenuState](mount, "menu2")
	assert.Equal(t, 1, s1.TabIndex)
	assert.Equal(t, 1, s2.TabIndex)
}

func TestUseTabMenu_Skip(t *testing.T) {
	t.Parallel()

	skipConfig := func() TabMenuConfig {
		return TabMenuConfig{
			TabCount:   1,
			ItemCounts: []int{5},
			Skips:      [][]bool{{true, false, false, true, false}},
		}
	}

	t.Run("下移動でスキップ対象を飛ばす", func(t *testing.T) {
		t.Parallel()
		mount := NewMount[tabMenuTestProps]()
		UseTabMenu(mount.Store(), "menu", skipConfig())
		mount.Update()

		// 初期状態でindex 0はスキップされ、1になる
		s, _ := GetState[TabMenuState](mount, "menu")
		assert.Equal(t, 1, s.ItemIndex, "初期位置がスキップ対象の場合は次に移動する")

		// 下に移動: 1→2
		mount.Dispatch(inputmapper.ActionMenuDown)
		UseTabMenu(mount.Store(), "menu", skipConfig())
		mount.Update()
		s, _ = GetState[TabMenuState](mount, "menu")
		assert.Equal(t, 2, s.ItemIndex)

		// 下に移動: 2→4（3はスキップ）
		mount.Dispatch(inputmapper.ActionMenuDown)
		UseTabMenu(mount.Store(), "menu", skipConfig())
		mount.Update()
		s, _ = GetState[TabMenuState](mount, "menu")
		assert.Equal(t, 4, s.ItemIndex, "スキップ対象を飛ばして次のアイテムに移動する")
	})

	t.Run("上移動でスキップ対象を飛ばす", func(t *testing.T) {
		t.Parallel()
		mount := NewMount[tabMenuTestProps]()
		UseTabMenu(mount.Store(), "menu", skipConfig())
		mount.Update()

		// index 4まで移動
		mount.Dispatch(inputmapper.ActionMenuDown) // 1→2
		mount.Dispatch(inputmapper.ActionMenuDown) // 2→4 (3スキップ)
		UseTabMenu(mount.Store(), "menu", skipConfig())
		mount.Update()
		s, _ := GetState[TabMenuState](mount, "menu")
		assert.Equal(t, 4, s.ItemIndex)

		// 上に移動: 4→2（3はスキップ）
		mount.Dispatch(inputmapper.ActionMenuUp)
		UseTabMenu(mount.Store(), "menu", skipConfig())
		mount.Update()
		s, _ = GetState[TabMenuState](mount, "menu")
		assert.Equal(t, 2, s.ItemIndex, "スキップ対象を飛ばして前のアイテムに移動する")
	})

	t.Run("タブ切り替え時にスキップ対象でない最初のアイテムに移動する", func(t *testing.T) {
		t.Parallel()
		mount := NewMount[tabMenuTestProps]()
		UseTabMenu(mount.Store(), "menu", TabMenuConfig{
			TabCount:   2,
			ItemCounts: []int{2, 3},
			Skips:      [][]bool{nil, {true, false, false}},
		})
		mount.Update()

		// tab2に切り替え
		mount.Dispatch(inputmapper.ActionMenuTabNext)
		UseTabMenu(mount.Store(), "menu", TabMenuConfig{
			TabCount:   2,
			ItemCounts: []int{2, 3},
			Skips:      [][]bool{nil, {true, false, false}},
		})
		mount.Update()
		s, _ := GetState[TabMenuState](mount, "menu")
		assert.Equal(t, 1, s.ItemIndex, "スキップ対象でない最初のアイテムに移動する")
	})
}

// ペジネーション用のセットアップ
func setupTabMenuStateWithPagination(t *testing.T, store *Store, p tabMenuTestProps, itemsPerPage int) TabMenuState {
	t.Helper()
	itemCounts := make([]int, len(p.Tabs))
	for i, tab := range p.Tabs {
		itemCounts[i] = len(tab.Items)
	}

	return UseTabMenu(store, "menu", TabMenuConfig{
		TabCount:     len(p.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: itemsPerPage,
	})
}

func TestUseTabMenu_Pagination_InitialState(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e", "f", "g"}},
		},
	}
	mount.SetProps(props)
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()

	assert.Equal(t, 0, result.TabIndex)
	assert.Equal(t, 0, result.ItemIndex)
	assert.Equal(t, 0, result.Page)
}

func TestUseTabMenu_Pagination_NavigateWithinPage(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()

	// ページ内で下に移動
	mount.Dispatch(inputmapper.ActionMenuDown)
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 1, result.ItemIndex)
	assert.Equal(t, 0, result.Page, "同じページ内にとどまる")

	mount.Dispatch(inputmapper.ActionMenuDown)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 2, result.ItemIndex)
	assert.Equal(t, 0, result.Page, "まだページ0内")
}

func TestUseTabMenu_Pagination_NavigateToNextPage(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()

	// ページ0の最後まで移動
	mount.Dispatch(inputmapper.ActionMenuDown) // index 1
	mount.Dispatch(inputmapper.ActionMenuDown) // index 2
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 2, result.ItemIndex)
	assert.Equal(t, 0, result.Page)

	// 次のページに移動
	mount.Dispatch(inputmapper.ActionMenuDown) // index 3
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 3, result.ItemIndex)
	assert.Equal(t, 1, result.Page, "ページ1に移動")
}

func TestUseTabMenu_Pagination_NavigateToPreviousPage(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e"}},
		},
	}
	mount.SetProps(props)

	// ページ1のindex 3から開始するために移動
	setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	mount.Dispatch(inputmapper.ActionMenuDown) // index 1
	mount.Dispatch(inputmapper.ActionMenuDown) // index 2
	mount.Dispatch(inputmapper.ActionMenuDown) // index 3, page 1
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 3, result.ItemIndex)
	assert.Equal(t, 1, result.Page)

	// 上に移動して前のページに戻る
	mount.Dispatch(inputmapper.ActionMenuUp) // index 2, page 0
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 2, result.ItemIndex)
	assert.Equal(t, 0, result.Page, "ページ0に戻る")
}

func TestUseTabMenu_Pagination_WrapToLastPage(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()

	// 最初のアイテムで上に移動すると最後に循環
	mount.Dispatch(inputmapper.ActionMenuUp)
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 4, result.ItemIndex, "最後のアイテムに循環")
	assert.Equal(t, 1, result.Page, "最後のページに移動")
}

func TestUseTabMenu_Pagination_WrapToFirstPage(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e"}},
		},
	}
	mount.SetProps(props)

	// 最後のアイテムまで移動
	setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	for range 4 {
		mount.Dispatch(inputmapper.ActionMenuDown)
		setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
		mount.Update()
	}
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 4, result.ItemIndex)
	assert.Equal(t, 1, result.Page)

	// 下に移動すると最初に循環
	mount.Dispatch(inputmapper.ActionMenuDown)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 0, result.ItemIndex, "最初のアイテムに循環")
	assert.Equal(t, 0, result.Page, "最初のページに戻る")
}

func TestUseTabMenu_Pagination_TabChangeResetsPage(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e"}},
			{ID: "tab2", Items: []string{"x", "y", "z"}},
		},
	}
	mount.SetProps(props)

	// ページ1まで移動
	setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	mount.Dispatch(inputmapper.ActionMenuDown) // index 1
	mount.Dispatch(inputmapper.ActionMenuDown) // index 2
	mount.Dispatch(inputmapper.ActionMenuDown) // index 3, page 1
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 1, result.Page)

	// タブを切り替え
	mount.Dispatch(inputmapper.ActionMenuTabNext)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 1, result.TabIndex)
	assert.Equal(t, 0, result.ItemIndex, "アイテムインデックスがリセット")
	assert.Equal(t, 0, result.Page, "ページがリセット")
}

func TestUseTabMenu_Pagination_DifferentPageSizes(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
		},
	}
	mount.SetProps(props)

	// itemsPerPage=5 でテスト
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 5)
	mount.Update()
	assert.Equal(t, 0, result.Page)

	// 5アイテム移動してページ1に到達
	for range 5 {
		mount.Dispatch(inputmapper.ActionMenuDown)
		result = setupTabMenuStateWithPagination(t, mount.Store(), props, 5)
		mount.Update()
	}
	assert.Equal(t, 5, result.ItemIndex)
	assert.Equal(t, 1, result.Page)

	// itemsPerPage=2 でテスト
	mount2 := NewMount[tabMenuTestProps]()
	mount2.SetProps(props)
	result2 := setupTabMenuStateWithPagination(t, mount2.Store(), props, 2)
	mount2.Update()
	assert.Equal(t, 0, result2.Page)

	// 2アイテム移動してページ1に到達
	mount2.Dispatch(inputmapper.ActionMenuDown)
	mount2.Dispatch(inputmapper.ActionMenuDown)
	result2 = setupTabMenuStateWithPagination(t, mount2.Store(), props, 2)
	mount2.Update()
	assert.Equal(t, 2, result2.ItemIndex)
	assert.Equal(t, 1, result2.Page)
}

func TestUseTabMenu_Pagination_PageNavigationWithLeftRight(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}},
		},
	}
	mount.SetProps(props)
	setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()

	// 右キーで次のページへ
	mount.Dispatch(inputmapper.ActionMenuRight)
	result := setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 3, result.ItemIndex, "1ページ分移動")
	assert.Equal(t, 1, result.Page)

	// さらに右キーで次のページへ
	mount.Dispatch(inputmapper.ActionMenuRight)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 6, result.ItemIndex)
	assert.Equal(t, 2, result.Page)

	// 最後のページでは右キーで移動しない
	mount.Dispatch(inputmapper.ActionMenuRight)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 6, result.ItemIndex, "最後のページでは移動しない")
	assert.Equal(t, 2, result.Page)

	// 左キーで前のページへ
	mount.Dispatch(inputmapper.ActionMenuLeft)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 3, result.ItemIndex)
	assert.Equal(t, 1, result.Page)

	// さらに左キーで前のページへ
	mount.Dispatch(inputmapper.ActionMenuLeft)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 0, result.ItemIndex)
	assert.Equal(t, 0, result.Page)

	// 最初のページでは左キーで移動しない
	mount.Dispatch(inputmapper.ActionMenuLeft)
	result = setupTabMenuStateWithPagination(t, mount.Store(), props, 3)
	mount.Update()
	assert.Equal(t, 0, result.ItemIndex, "最初のページでは移動しない")
	assert.Equal(t, 0, result.Page)
}
