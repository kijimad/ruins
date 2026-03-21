package hooks

import "github.com/kijimaD/ruins/internal/inputmapper"

// TabMenuConfig はタブメニューの設定
type TabMenuConfig struct {
	TabCount     int      // タブの数
	ItemCounts   []int    // 各タブのアイテム数
	ItemsPerPage int      // 1ページに表示するアイテム数（0=ペジネーションなし）
	Skips        [][]bool // 各タブのスキップ判定。trueの位置はカーソルが止まらない
}

// TabMenuState はタブメニューの状態
type TabMenuState struct {
	TabIndex  int
	ItemIndex int
	Page      int // 現在のページ（0ベース）
}

// tabMenuNav はタブメニューのナビゲーションロジックを保持する
type tabMenuNav struct {
	config TabMenuConfig
}

func (n *tabMenuNav) itemCountForTab(tabIdx int) int {
	if tabIdx >= 0 && tabIdx < len(n.config.ItemCounts) {
		return n.config.ItemCounts[tabIdx]
	}
	return 0
}

func (n *tabMenuNav) isSkip(tabIdx, itemIdx int) bool {
	if tabIdx < 0 || tabIdx >= len(n.config.Skips) {
		return false
	}
	skips := n.config.Skips[tabIdx]
	if itemIdx < 0 || itemIdx >= len(skips) {
		return false
	}
	return skips[itemIdx]
}

// skipNext は指定タブ内で指定方向にスキップ対象でない次のインデックスを返す
func (n *tabMenuNav) skipNext(tabIdx, idx, dir int) int {
	count := n.itemCountForTab(tabIdx)
	if tabIdx < 0 || tabIdx >= len(n.config.Skips) || len(n.config.Skips[tabIdx]) == 0 {
		return idx
	}
	for range count {
		if !n.isSkip(tabIdx, idx) {
			return idx
		}
		idx = (idx + dir + count) % count
	}
	return idx
}

func (n *tabMenuNav) firstSelectable(tabIdx int) int {
	return n.skipNext(tabIdx, 0, 1)
}

// reduce はアクションに応じて状態を更新する
func (n *tabMenuNav) reduce(s TabMenuState, a inputmapper.ActionID) TabMenuState {
	tabIdx := s.TabIndex
	itemIdx := s.ItemIndex
	count := n.itemCountForTab(tabIdx)

	switch a {
	case inputmapper.ActionMenuTabPrev:
		if n.config.TabCount == 0 {
			return s
		}
		newTab := (tabIdx - 1 + n.config.TabCount) % n.config.TabCount
		return TabMenuState{TabIndex: newTab, ItemIndex: n.firstSelectable(newTab)}
	case inputmapper.ActionMenuTabNext:
		if n.config.TabCount == 0 {
			return s
		}
		newTab := (tabIdx + 1) % n.config.TabCount
		return TabMenuState{TabIndex: newTab, ItemIndex: n.firstSelectable(newTab)}
	case inputmapper.ActionMenuUp:
		if count == 0 {
			return s
		}
		next := (itemIdx - 1 + count) % count
		return TabMenuState{TabIndex: tabIdx, ItemIndex: n.skipNext(tabIdx, next, -1)}
	case inputmapper.ActionMenuDown:
		if count == 0 {
			return s
		}
		next := (itemIdx + 1) % count
		return TabMenuState{TabIndex: tabIdx, ItemIndex: n.skipNext(tabIdx, next, 1)}
	case inputmapper.ActionMenuLeft:
		if n.config.ItemsPerPage > 0 && itemIdx >= n.config.ItemsPerPage {
			return TabMenuState{TabIndex: tabIdx, ItemIndex: n.skipNext(tabIdx, itemIdx-n.config.ItemsPerPage, 1)}
		}
		return s
	case inputmapper.ActionMenuRight:
		if n.config.ItemsPerPage > 0 && itemIdx+n.config.ItemsPerPage < count {
			return TabMenuState{TabIndex: tabIdx, ItemIndex: n.skipNext(tabIdx, itemIdx+n.config.ItemsPerPage, 1)}
		}
		return s
	default:
		return s
	}
}

// UseTabMenu は再利用可能なタブメニュー状態管理を提供する
// ReactのカスタムHooksに相当するパターンで、複数のUseStateを組み合わせる
// keyPrefixは状態キーの接頭辞で、複数のタブメニューを区別するために使う
// 端での循環は常に有効
// ペジネーションが有効な場合、ページはitemIndexから自動計算される
// タブ切り替え: Tab / Shift+Tab
// ページ移動: 左右キー
func UseTabMenu(store *Store, keyPrefix string, config TabMenuConfig) TabMenuState {
	nav := &tabMenuNav{config: config}

	// tabIndexとitemIndexを複合状態として管理する。
	// 単一のreducerでタブ切り替えとアイテム移動を原子的に処理するため、
	// タブ切り替え時に新タブのitemsで正しくfirstSelectableを計算できる
	init := TabMenuState{ItemIndex: nav.firstSelectable(0)}
	state := UseState(store, keyPrefix, init, nav.reduce)

	// ページはitemIndexから派生する値として計算
	itemCount := nav.itemCountForTab(state.TabIndex)
	if nav.config.ItemsPerPage > 0 && itemCount > 0 {
		state.Page = state.ItemIndex / nav.config.ItemsPerPage
	}

	return state
}
