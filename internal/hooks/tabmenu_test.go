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
func setupTabMenuState(store *Store, p tabMenuTestProps) {
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
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	tabIndex, _ := GetState[int](mount, "menu_tabIndex")
	itemIndex, _ := GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 0, tabIndex)
	assert.Equal(t, 0, itemIndex)
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
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	// 右に移動
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	tabIndex, _ := GetState[int](mount, "menu_tabIndex")
	assert.Equal(t, 1, tabIndex)

	// さらに右に移動
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	tabIndex, _ = GetState[int](mount, "menu_tabIndex")
	assert.Equal(t, 2, tabIndex)

	// 循環して最初に戻る
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	tabIndex, _ = GetState[int](mount, "menu_tabIndex")
	assert.Equal(t, 0, tabIndex, "最後のタブから右に移動すると最初のタブに循環する")
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
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	// 最初のタブで左に移動すると最後のタブに循環
	mount.Dispatch(inputmapper.ActionMenuLeft)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	tabIndex, _ := GetState[int](mount, "menu_tabIndex")
	assert.Equal(t, 1, tabIndex, "最初のタブから左に移動すると最後のタブに循環する")
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
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	// 下に移動
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	itemIndex, _ := GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 1, itemIndex)

	// さらに下に移動
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	itemIndex, _ = GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 2, itemIndex)

	// 循環して最初に戻る
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	itemIndex, _ = GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 0, itemIndex, "最後のアイテムから下に移動すると最初に循環する")
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
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	// 最初のアイテムで上に移動すると最後に循環
	mount.Dispatch(inputmapper.ActionMenuUp)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	itemIndex, _ := GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 2, itemIndex, "最初のアイテムから上に移動すると最後に循環する")
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
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	// アイテムを選択
	mount.Dispatch(inputmapper.ActionMenuDown)
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	itemIndex, _ := GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 2, itemIndex)

	// タブを切り替え
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	itemIndex, _ = GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 0, itemIndex, "タブ切り替え時にアイテムインデックスがリセットされる")
}

func TestUseTabMenu_EmptyTabs(t *testing.T) {
	t.Parallel()
	mount := NewMount[tabMenuTestProps]()
	props := tabMenuTestProps{
		Tabs: []testTabData{},
	}
	mount.SetProps(props)
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	tabIndex, _ := GetState[int](mount, "menu_tabIndex")
	itemIndex, _ := GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 0, tabIndex)
	assert.Equal(t, 0, itemIndex)

	// 操作しても変わらない
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	tabIndex, _ = GetState[int](mount, "menu_tabIndex")
	assert.Equal(t, 0, tabIndex)
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
	setupTabMenuState(mount.Store(), props)

	mount.Update()

	itemIndex, _ := GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 0, itemIndex)

	// 操作しても変わらない
	mount.Dispatch(inputmapper.ActionMenuDown)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	itemIndex, _ = GetState[int](mount, "menu_itemIndex")
	assert.Equal(t, 0, itemIndex)
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
	setupTabMenuState(mount.Store(), props)
	mount.Update()

	// 2番目のタブに移動
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupTabMenuState(mount.Store(), props)
	mount.Update()
	tabIndex, _ := GetState[int](mount, "menu_tabIndex")
	assert.Equal(t, 1, tabIndex)

	// タブが2つに減る
	newProps := tabMenuTestProps{
		Tabs: []testTabData{
			{ID: "tab1", Items: []string{"a"}},
			{ID: "tab2", Items: []string{"b"}},
		},
	}
	mount.SetProps(newProps)
	setupTabMenuState(mount.Store(), newProps)
	mount.Update()

	// 右に移動すると2タブなので循環して0になる
	// tabIndex=1, tabCount=2 → (1+1) % 2 = 0
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupTabMenuState(mount.Store(), newProps)
	mount.Update()
	tabIndex, _ = GetState[int](mount, "menu_tabIndex")
	assert.Equal(t, 0, tabIndex, "Propsが変わると新しいタブ数で循環する")
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
	tab1, _ := GetState[int](mount, "menu1_tabIndex")
	tab2, _ := GetState[int](mount, "menu2_tabIndex")
	assert.Equal(t, 0, tab1)
	assert.Equal(t, 0, tab2)

	// 同じDispatchで両方更新される
	mount.Dispatch(inputmapper.ActionMenuRight)
	setupMultiMenu(mount.Store(), props)
	mount.Update()
	tab1, _ = GetState[int](mount, "menu1_tabIndex")
	tab2, _ = GetState[int](mount, "menu2_tabIndex")
	assert.Equal(t, 1, tab1)
	assert.Equal(t, 1, tab2)
}
