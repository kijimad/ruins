package tabmenu

import (
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
)

// View は tabMenu と uiBuilder を統合した構造体
// tabMenu（状態管理）と uiBuilder（UI表示）を一緒に管理する
type View struct {
	tabMenu   *tabMenu
	uiBuilder *uiBuilder
}

// NewView は View を作成する
func NewView(config Config, callbacks Callbacks, world w.World) *View {
	// OnItemChange コールバックにフォーカス更新を自動追加
	originalOnItemChange := callbacks.OnItemChange
	callbacks.OnItemChange = func(tabIndex int, oldItemIndex, newItemIndex int, item Item) error {
		// 元のコールバックを呼び出し
		if originalOnItemChange != nil {
			if err := originalOnItemChange(tabIndex, oldItemIndex, newItemIndex, item); err != nil {
				return err
			}
		}
		return nil
	}

	tabMenu := newTabMenu(config, callbacks)
	uiBuilder := newUIBuilder(world)

	view := &View{
		tabMenu:   tabMenu,
		uiBuilder: uiBuilder,
	}

	return view
}

// BuildUI はメニューのUIを構築する
func (v *View) BuildUI() *widget.Container {
	return v.uiBuilder.BuildUI(v.tabMenu)
}

// Update はキーボード入力を処理する
func (v *View) Update() error {
	_, err := v.tabMenu.Update()
	if err != nil {
		return err
	}

	// フォーカスが変わった可能性があるのでUIを更新
	v.uiBuilder.UpdateFocus(v.tabMenu)

	return nil
}

// DoAction はアクションを実行する
func (v *View) DoAction(actionID inputmapper.ActionID) error {
	err := v.tabMenu.DoAction(actionID)
	if err != nil {
		return err
	}

	// アクション実行後にUIを更新
	v.uiBuilder.UpdateFocus(v.tabMenu)

	return nil
}

// GetCurrentItemIndex は現在のアイテムインデックスを取得する
func (v *View) GetCurrentItemIndex() int {
	return v.tabMenu.GetCurrentItemIndex()
}

// GetCurrentTab は現在のタブを取得する
func (v *View) GetCurrentTab() TabItem {
	return v.tabMenu.GetCurrentTab()
}

// UpdateTabs はタブを更新する
func (v *View) UpdateTabs(tabs []TabItem) {
	v.tabMenu.UpdateTabs(tabs)
	v.uiBuilder.UpdateFocus(v.tabMenu)
}

// GetCurrentTabIndex は現在のタブインデックスを取得する
func (v *View) GetCurrentTabIndex() int {
	return v.tabMenu.GetCurrentTabIndex()
}

// GetPageIndicatorText はページインジケーターのテキストを取得する
func (v *View) GetPageIndicatorText() string {
	return v.tabMenu.GetPageIndicatorText()
}

// GetVisibleItems は現在のページで表示されるアイテムとインデックスを取得する
func (v *View) GetVisibleItems() ([]Item, []int) {
	return v.tabMenu.GetVisibleItems()
}

// SetTabIndex はタブインデックスを設定する
func (v *View) SetTabIndex(index int) error {
	err := v.tabMenu.SetTabIndex(index)
	if err != nil {
		return err
	}
	v.uiBuilder.UpdateFocus(v.tabMenu)
	return nil
}

// SetItemIndex はアイテムインデックスを設定する
func (v *View) SetItemIndex(index int) error {
	err := v.tabMenu.SetItemIndex(index)
	if err != nil {
		return err
	}
	v.uiBuilder.UpdateFocus(v.tabMenu)
	return nil
}

// GetCurrentPage は現在のページ番号を返す
func (v *View) GetCurrentPage() int {
	return v.tabMenu.GetCurrentPage()
}

// UpdateTabDisplayContainer はタブ表示コンテナを更新する
// ページインジケーター、アイテム一覧、空の場合のメッセージを表示する
func (v *View) UpdateTabDisplayContainer(container *widget.Container) {
	v.uiBuilder.UpdateTabDisplayContainer(container, v.tabMenu)
}
