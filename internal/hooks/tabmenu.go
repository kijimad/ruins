package hooks

import "github.com/kijimaD/ruins/internal/inputmapper"

// TabMenuConfig はタブメニューの設定
type TabMenuConfig struct {
	TabCount   int   // タブの数
	ItemCounts []int // 各タブのアイテム数
}

// TabMenuResult はタブメニューの状態
type TabMenuResult struct {
	TabIndex  int
	ItemIndex int
}

// UseTabMenu は再利用可能なタブメニュー状態管理を提供する
// ReactのカスタムHooksに相当するパターンで、複数のUseStateを組み合わせる
// keyPrefixは状態キーの接頭辞で、複数のタブメニューを区別するために使う
// 端での循環は常に有効
func UseTabMenu(store *Store, keyPrefix string, config TabMenuConfig) TabMenuResult {
	tabCount := config.TabCount

	tabIndex := UseState(store, keyPrefix+"_tabIndex", 0, func(v int, a inputmapper.ActionID) int {
		if tabCount == 0 {
			return 0
		}
		switch a {
		case inputmapper.ActionMenuLeft:
			return (v - 1 + tabCount) % tabCount
		case inputmapper.ActionMenuRight:
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
		case inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight:
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

	return TabMenuResult{
		TabIndex:  tabIndex,
		ItemIndex: itemIndex,
	}
}
