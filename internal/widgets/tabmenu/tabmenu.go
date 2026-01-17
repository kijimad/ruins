package tabmenu

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
)

// TabItem はタブの項目を定義する
type TabItem struct {
	ID    string
	Label string
	Items []Item
}

// Config はタブメニューの設定
type Config struct {
	Tabs             []TabItem
	InitialTabIndex  int
	InitialItemIndex int
	WrapNavigation   bool // タブ/アイテム両方で端循環するか
	ItemsPerPage     int  // 1ページに表示する項目数（0=制限なし）
}

// Callbacks はタブメニューのコールバック
type Callbacks struct {
	OnSelectItem func(tabIndex int, itemIndex int, tab TabItem, item Item) error
	OnCancel     func()
	OnTabChange  func(oldTabIndex, newTabIndex int, tab TabItem)
	OnItemChange func(tabIndex int, oldItemIndex, newItemIndex int, item Item) error
}

// TabMenu はタブ付きメニューコンポーネント
type tabMenu struct {
	config    Config
	callbacks Callbacks

	// 状態
	currentTabIndex  int
	currentItemIndex int

	// ページネーション状態
	currentPage int // 現在のページ（0ベース）
}

// newTabMenu は新しいtabMenuを作成する
func newTabMenu(config Config, callbacks Callbacks) *tabMenu {
	tm := &tabMenu{
		config:           config,
		callbacks:        callbacks,
		currentTabIndex:  config.InitialTabIndex,
		currentItemIndex: config.InitialItemIndex,
	}

	// 初期タブのアイテム数を確認してインデックスを調整
	if len(config.Tabs) > 0 && config.InitialTabIndex < len(config.Tabs) {
		initialTab := config.Tabs[config.InitialTabIndex]
		if len(initialTab.Items) == 0 {
			tm.currentItemIndex = -1
		} else if config.InitialItemIndex >= len(initialTab.Items) {
			tm.currentItemIndex = len(initialTab.Items) - 1
		} else if config.InitialItemIndex < 0 {
			tm.currentItemIndex = 0
		}
	}

	return tm
}

// Update はキーボード入力を待ち受けて、Actionに変換してタブメニュー操作を実行する
// 本実装で使用する。テストではDoAction()を直接呼ぶ
func (tm *tabMenu) Update() (bool, error) {
	keyboardInput := input.GetSharedKeyboardInput()
	if action, ok := tm.translateKeyToAction(keyboardInput); ok {
		return false, tm.DoAction(action)
	}
	return false, nil
}

// translateKeyToAction はキーボード入力をActionに変換する
func (tm *tabMenu) translateKeyToAction(keyboardInput input.KeyboardInput) (inputmapper.ActionID, bool) {
	// 左移動キー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		return inputmapper.ActionMenuLeft, true
	}

	// 右移動キー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowRight) {
		return inputmapper.ActionMenuRight, true
	}

	// 上移動キー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowUp) || keyboardInput.IsKeyJustPressed(ebiten.KeyW) {
		return inputmapper.ActionMenuUp, true
	}

	// 下移動キー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyArrowDown) || keyboardInput.IsKeyJustPressed(ebiten.KeyS) {
		return inputmapper.ActionMenuDown, true
	}

	// Tabキー（タブ切り替え）
	if keyboardInput.IsKeyJustPressed(ebiten.KeyTab) {
		if keyboardInput.IsKeyPressed(ebiten.KeyShift) {
			return inputmapper.ActionMenuLeft, true
		}
		return inputmapper.ActionMenuRight, true
	}

	// Enterキー（セッションベース検出で複数回実行を防止）
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionMenuSelect, true
	}

	// Escapeキー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}

	return "", false
}

// DoAction はActionを受け取ってタブメニュー操作を実行する
func (tm *tabMenu) DoAction(action inputmapper.ActionID) error {
	switch action {
	case inputmapper.ActionMenuLeft:
		return tm.navigateToPreviousTab()
	case inputmapper.ActionMenuRight:
		return tm.navigateToNextTab()
	case inputmapper.ActionMenuUp:
		return tm.navigateToPreviousItem()
	case inputmapper.ActionMenuDown:
		return tm.navigateToNextItem()
	case inputmapper.ActionMenuSelect:
		return tm.selectCurrentItem()
	case inputmapper.ActionMenuCancel:
		if tm.callbacks.OnCancel != nil {
			tm.callbacks.OnCancel()
		}
		return nil
	default:
		return nil
	}
}

// navigateToPreviousTab は前のタブに移動する
func (tm *tabMenu) navigateToPreviousTab() error {
	oldIndex := tm.currentTabIndex

	if tm.currentTabIndex > 0 {
		tm.currentTabIndex--
	} else if tm.config.WrapNavigation {
		tm.currentTabIndex = len(tm.config.Tabs) - 1
	}

	if oldIndex != tm.currentTabIndex {
		// タブ変更時はアイテムインデックスとページをリセット
		newTab := tm.config.Tabs[tm.currentTabIndex]
		if len(newTab.Items) > 0 {
			tm.currentItemIndex = 0
			tm.currentPage = 0
		} else {
			tm.currentItemIndex = -1 // 空のタブでは無効なインデックス
		}

		if tm.callbacks.OnTabChange != nil {
			tm.callbacks.OnTabChange(oldIndex, tm.currentTabIndex, newTab)
		}

		// アイテムが存在する場合のみnotifyItemChangeを呼ぶ
		if len(newTab.Items) > 0 {
			if err := tm.notifyItemChange(0, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

// navigateToNextTab は次のタブに移動する
func (tm *tabMenu) navigateToNextTab() error {
	oldIndex := tm.currentTabIndex

	if tm.currentTabIndex < len(tm.config.Tabs)-1 {
		tm.currentTabIndex++
	} else if tm.config.WrapNavigation {
		tm.currentTabIndex = 0
	}

	if oldIndex != tm.currentTabIndex {
		// タブ変更時はアイテムインデックスとページをリセット
		newTab := tm.config.Tabs[tm.currentTabIndex]
		if len(newTab.Items) > 0 {
			tm.currentItemIndex = 0
			tm.currentPage = 0
		} else {
			tm.currentItemIndex = -1 // 空のタブでは無効なインデックス
		}

		if tm.callbacks.OnTabChange != nil {
			tm.callbacks.OnTabChange(oldIndex, tm.currentTabIndex, newTab)
		}

		// アイテムが存在する場合のみnotifyItemChangeを呼ぶ
		if len(newTab.Items) > 0 {
			if err := tm.notifyItemChange(0, 0); err != nil {
				return err
			}
		}
	}
	return nil
}

// navigateToPreviousItem は前のアイテムに移動する
func (tm *tabMenu) navigateToPreviousItem() error {
	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return nil
	}

	currentTab := tm.config.Tabs[tm.currentTabIndex]
	if len(currentTab.Items) == 0 {
		return nil
	}

	oldIndex := tm.currentItemIndex

	// ページネーション対応
	if tm.config.ItemsPerPage > 0 {
		pageStart := tm.currentPage * tm.config.ItemsPerPage

		// ページ内での移動
		if tm.currentItemIndex < 0 {
			tm.currentItemIndex = pageStart
		} else if tm.currentItemIndex > pageStart {
			tm.currentItemIndex--
		} else if tm.currentPage > 0 {
			// 前のページへ
			tm.currentPage--
			tm.currentItemIndex = (tm.currentPage+1)*tm.config.ItemsPerPage - 1
		} else if tm.config.WrapNavigation {
			// 最後のページへ
			tm.currentPage = (len(currentTab.Items) - 1) / tm.config.ItemsPerPage
			tm.currentItemIndex = len(currentTab.Items) - 1
		}
	} else {
		// ページネーションなし
		if tm.currentItemIndex < 0 {
			tm.currentItemIndex = 0
		} else if tm.currentItemIndex > 0 {
			tm.currentItemIndex--
		} else if tm.config.WrapNavigation {
			tm.currentItemIndex = len(currentTab.Items) - 1
		}
	}

	if oldIndex != tm.currentItemIndex {
		if err := tm.notifyItemChange(oldIndex, tm.currentItemIndex); err != nil {
			return err
		}
	}
	return nil
}

// navigateToNextItem は次のアイテムに移動する
func (tm *tabMenu) navigateToNextItem() error {
	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return nil
	}

	currentTab := tm.config.Tabs[tm.currentTabIndex]
	if len(currentTab.Items) == 0 {
		return nil
	}

	oldIndex := tm.currentItemIndex

	// ページネーション対応
	if tm.config.ItemsPerPage > 0 {
		pageStart := tm.currentPage * tm.config.ItemsPerPage
		pageEnd := pageStart + tm.config.ItemsPerPage
		if pageEnd > len(currentTab.Items) {
			pageEnd = len(currentTab.Items)
		}

		// ページ内での移動
		if tm.currentItemIndex < 0 {
			tm.currentItemIndex = pageStart
		} else if tm.currentItemIndex < pageEnd-1 {
			tm.currentItemIndex++
		} else if pageEnd < len(currentTab.Items) {
			// 次のページへ
			tm.currentPage++
			tm.currentItemIndex = tm.currentPage * tm.config.ItemsPerPage
		} else if tm.config.WrapNavigation {
			// 最初のページへ
			tm.currentPage = 0
			tm.currentItemIndex = 0
		}
	} else {
		// ページネーションなし
		if tm.currentItemIndex < 0 {
			tm.currentItemIndex = 0
		} else if tm.currentItemIndex < len(currentTab.Items)-1 {
			tm.currentItemIndex++
		} else if tm.config.WrapNavigation {
			tm.currentItemIndex = 0
		}
	}

	if oldIndex != tm.currentItemIndex {
		if err := tm.notifyItemChange(oldIndex, tm.currentItemIndex); err != nil {
			return err
		}
	}
	return nil
}

// selectCurrentItem は現在のアイテムを選択する
func (tm *tabMenu) selectCurrentItem() error {
	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return nil
	}

	currentTab := tm.config.Tabs[tm.currentTabIndex]
	if len(currentTab.Items) == 0 || tm.currentItemIndex >= len(currentTab.Items) || tm.currentItemIndex < 0 {
		return nil
	}

	currentItem := currentTab.Items[tm.currentItemIndex]

	if tm.callbacks.OnSelectItem != nil {
		if err := tm.callbacks.OnSelectItem(tm.currentTabIndex, tm.currentItemIndex, currentTab, currentItem); err != nil {
			return err
		}
	}
	return nil
}

// notifyItemChange はアイテム変更を通知する
func (tm *tabMenu) notifyItemChange(oldIndex, newIndex int) error {
	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return nil
	}

	currentTab := tm.config.Tabs[tm.currentTabIndex]
	if len(currentTab.Items) == 0 || newIndex >= len(currentTab.Items) || newIndex < 0 {
		return nil
	}

	if tm.callbacks.OnItemChange != nil {
		if err := tm.callbacks.OnItemChange(tm.currentTabIndex, oldIndex, newIndex, currentTab.Items[newIndex]); err != nil {
			return err
		}
	}
	return nil
}

// GetCurrentTabIndex は現在のタブインデックスを返す
func (tm *tabMenu) GetCurrentTabIndex() int {
	return tm.currentTabIndex
}

// GetCurrentItemIndex は現在のアイテムインデックスを返す
func (tm *tabMenu) GetCurrentItemIndex() int {
	return tm.currentItemIndex
}

// GetCurrentTab は現在のタブを返す
func (tm *tabMenu) GetCurrentTab() TabItem {
	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return TabItem{}
	}
	return tm.config.Tabs[tm.currentTabIndex]
}

// GetCurrentItem は現在のアイテムを返す
func (tm *tabMenu) GetCurrentItem() Item {
	currentTab := tm.GetCurrentTab()
	if len(currentTab.Items) == 0 || tm.currentItemIndex >= len(currentTab.Items) || tm.currentItemIndex < 0 {
		return Item{}
	}
	return currentTab.Items[tm.currentItemIndex]
}

// SetTabIndex はタブインデックスを設定する
func (tm *tabMenu) SetTabIndex(index int) error {
	if index >= 0 && index < len(tm.config.Tabs) {
		oldIndex := tm.currentTabIndex
		tm.currentTabIndex = index

		// タブ変更時はアイテムインデックスとページをリセット
		newTab := tm.config.Tabs[tm.currentTabIndex]
		if len(newTab.Items) > 0 {
			tm.currentItemIndex = 0
			tm.currentPage = 0
		} else {
			tm.currentItemIndex = -1 // 空のタブでは無効なインデックス
		}

		if oldIndex != tm.currentTabIndex {
			if tm.callbacks.OnTabChange != nil {
				tm.callbacks.OnTabChange(oldIndex, tm.currentTabIndex, newTab)
			}

			// アイテムが存在する場合のみnotifyItemChangeを呼ぶ
			if len(newTab.Items) > 0 {
				if err := tm.notifyItemChange(0, 0); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// SetItemIndex はアイテムインデックスを設定する
func (tm *tabMenu) SetItemIndex(index int) error {
	currentTab := tm.GetCurrentTab()
	if index >= 0 && index < len(currentTab.Items) {
		oldIndex := tm.currentItemIndex
		tm.currentItemIndex = index

		if oldIndex != tm.currentItemIndex {
			if err := tm.notifyItemChange(oldIndex, tm.currentItemIndex); err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateTabs はタブを更新する（動的にアイテムが変更された場合）
func (tm *tabMenu) UpdateTabs(tabs []TabItem) {
	tm.config.Tabs = tabs

	// 現在のインデックスが範囲外になった場合は調整
	if tm.currentTabIndex >= len(tabs) {
		tm.currentTabIndex = len(tabs) - 1
		if tm.currentTabIndex < 0 {
			tm.currentTabIndex = 0
		}
	}

	// 現在のタブのアイテムインデックスを調整
	if len(tabs) > 0 && tm.currentTabIndex < len(tabs) {
		currentTab := tabs[tm.currentTabIndex]
		if len(currentTab.Items) == 0 {
			tm.currentItemIndex = -1 // 空のタブでは無効なインデックス
		} else if tm.currentItemIndex >= len(currentTab.Items) {
			tm.currentItemIndex = len(currentTab.Items) - 1
		} else if tm.currentItemIndex < 0 {
			tm.currentItemIndex = 0
		}

		// UpdateTabsは内部状態の更新のみを行い、コールバックは呼ばない
		// コールバックは呼び出し元で必要に応じて実行される
	}
}

// GetVisibleItems は現在のページで表示される項目とその元のインデックスを返す
func (tm *tabMenu) GetVisibleItems() ([]Item, []int) {
	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return []Item{}, []int{}
	}

	currentTab := tm.config.Tabs[tm.currentTabIndex]

	if tm.config.ItemsPerPage <= 0 {
		indices := make([]int, len(currentTab.Items))
		for i := range indices {
			indices[i] = i
		}
		return currentTab.Items, indices
	}

	start := tm.currentPage * tm.config.ItemsPerPage
	end := start + tm.config.ItemsPerPage
	if end > len(currentTab.Items) {
		end = len(currentTab.Items)
	}

	if start >= len(currentTab.Items) {
		return []Item{}, []int{}
	}

	visibleItems := currentTab.Items[start:end]
	indices := make([]int, len(visibleItems))
	for i := range indices {
		indices[i] = start + i
	}

	return visibleItems, indices
}

// GetCurrentPage は現在のページ番号を返す(表示用なので1ベース)
func (tm *tabMenu) GetCurrentPage() int {
	return tm.currentPage + 1
}

// GetTotalPages は総ページ数を返す
func (tm *tabMenu) GetTotalPages() int {
	if tm.config.ItemsPerPage <= 0 {
		return 1
	}

	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return 1
	}

	currentTab := tm.config.Tabs[tm.currentTabIndex]
	return (len(currentTab.Items) + tm.config.ItemsPerPage - 1) / tm.config.ItemsPerPage
}

// GetPageIndicatorText はページインジケーターのテキストを返す
func (tm *tabMenu) GetPageIndicatorText() string {
	if tm.config.ItemsPerPage <= 0 || tm.GetTotalPages() <= 1 {
		return ""
	}

	arrows := ""

	// 前のページがある場合は上矢印を追加
	if tm.HasPreviousPage() {
		arrows += " ↑"
	} else {
		arrows += " 　"
	}

	// 次のページがある場合は下矢印を追加
	if tm.HasNextPage() {
		arrows += " ↓"
	} else {
		arrows += " 　"
	}

	return fmt.Sprintf("%d/%d%s", tm.GetCurrentPage(), tm.GetTotalPages(), arrows)
}

// HasPreviousPage は前のページがあるかを返す
func (tm *tabMenu) HasPreviousPage() bool {
	return tm.currentPage > 0
}

// HasNextPage は次のページがあるかを返す
func (tm *tabMenu) HasNextPage() bool {
	if tm.config.ItemsPerPage <= 0 {
		return false
	}

	if len(tm.config.Tabs) == 0 || tm.currentTabIndex >= len(tm.config.Tabs) {
		return false
	}

	currentTab := tm.config.Tabs[tm.currentTabIndex]
	nextPageStart := (tm.currentPage + 1) * tm.config.ItemsPerPage
	return nextPageStart < len(currentTab.Items)
}
