package hooks

import "github.com/kijimaD/ruins/internal/inputmapper"

// TabMenuConfig はタブメニューの設定
type TabMenuConfig struct {
	TabCount     int   // タブの数
	ItemCounts   []int // 各タブのアイテム数
	ItemsPerPage int   // 1ページに表示するアイテム数（0=ペジネーションなし）
}

// TabMenuState はタブメニューの状態
type TabMenuState struct {
	TabIndex  int
	ItemIndex int
	Page      int // 現在のページ（0ベース）
}

// UseTabMenu は再利用可能なタブメニュー状態管理を提供する
// ReactのカスタムHooksに相当するパターンで、複数のUseStateを組み合わせる
// keyPrefixは状態キーの接頭辞で、複数のタブメニューを区別するために使う
// 端での循環は常に有効
// ペジネーションが有効な場合、ページはitemIndexから自動計算される
// タブ切り替え: Tab / Shift+Tab
// ページ移動: 左右キー
func UseTabMenu(store *Store, keyPrefix string, config TabMenuConfig) TabMenuState {
	tabCount := config.TabCount
	itemsPerPage := config.ItemsPerPage

	tabIndex := UseState(store, keyPrefix+"_tabIndex", 0, func(v int, a inputmapper.ActionID) int {
		if tabCount == 0 {
			return 0
		}
		switch a {
		case inputmapper.ActionMenuTabPrev:
			return (v - 1 + tabCount) % tabCount
		case inputmapper.ActionMenuTabNext:
			return (v + 1) % tabCount
		default:
			return v
		}
	})

	// 現在のタブのアイテム数を取得
	itemCount := 0
	if tabIndex >= 0 && tabIndex < len(config.ItemCounts) {
		itemCount = config.ItemCounts[tabIndex]
	}

	// itemIndexの更新
	// 上下: 1アイテムずつ移動（循環あり）
	// 左右: 1ページ分移動（循環なし）
	// タブ切り替え時: リセット
	itemIndex := UseState(store, keyPrefix+"_itemIndex", 0, func(v int, a inputmapper.ActionID) int {
		switch a {
		case inputmapper.ActionMenuUp:
			if itemCount == 0 {
				return 0
			}
			return (v - 1 + itemCount) % itemCount
		case inputmapper.ActionMenuDown:
			if itemCount == 0 {
				return 0
			}
			return (v + 1) % itemCount
		case inputmapper.ActionMenuLeft:
			// 前のページへ（循環なし）
			if itemsPerPage > 0 && v >= itemsPerPage {
				return v - itemsPerPage
			}
			return v
		case inputmapper.ActionMenuRight:
			// 次のページへ（循環なし）
			if itemsPerPage > 0 && v+itemsPerPage < itemCount {
				return v + itemsPerPage
			}
			return v
		case inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			// タブ切り替え時にアイテムインデックスをリセット
			return 0
		default:
			return v
		}
	})

	// itemIndexが範囲外の場合は補正
	if itemCount > 0 && itemIndex >= itemCount {
		itemIndex = itemCount - 1
	}

	// ページはitemIndexから派生する値として計算
	page := 0
	if itemsPerPage > 0 && itemCount > 0 {
		page = itemIndex / itemsPerPage
	}

	return TabMenuState{
		TabIndex:  tabIndex,
		ItemIndex: itemIndex,
		Page:      page,
	}
}
