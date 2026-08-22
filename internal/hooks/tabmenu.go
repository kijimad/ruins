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

// clamp はアイテム数変化に対してインデックスを安全な範囲に収める
func (n *tabMenuNav) clamp(s TabMenuState) TabMenuState {
	count := n.itemCountForTab(s.TabIndex)
	if count == 0 {
		s.ItemIndex = 0
	} else if s.ItemIndex >= count {
		s.ItemIndex = count - 1
	}
	return s
}

// navHandlers は移動系 Action と状態遷移の対応表。移動系の集合の唯一の定義で、
// DispatchNav の消費判定も reduce の分岐もここから導く。集合と挙動が別の場所に複製されると
// 片方だけ更新してずれるため、1つの表に束ねる
var navHandlers = map[inputmapper.ActionID]func(*tabMenuNav, TabMenuState) TabMenuState{
	inputmapper.ActionMenuTabPrev: (*tabMenuNav).tabPrev,
	inputmapper.ActionMenuTabNext: (*tabMenuNav).tabNext,
	inputmapper.ActionMenuUp:      (*tabMenuNav).up,
	inputmapper.ActionMenuDown:    (*tabMenuNav).down,
	inputmapper.ActionMenuLeft:    (*tabMenuNav).left,
	inputmapper.ActionMenuRight:   (*tabMenuNav).right,
}

// reduce はアクションに応じて状態を更新する。移動系でなければ何もしない
func (n *tabMenuNav) reduce(s TabMenuState, a inputmapper.ActionID) TabMenuState {
	s = n.clamp(s)
	if h, ok := navHandlers[a]; ok {
		return h(n, s)
	}
	return s
}

// tabPrev は前のタブへ循環して移り、カーソルを新タブの先頭選択可能行へ置く
func (n *tabMenuNav) tabPrev(s TabMenuState) TabMenuState {
	if n.config.TabCount == 0 {
		return s
	}
	newTab := (s.TabIndex - 1 + n.config.TabCount) % n.config.TabCount
	return TabMenuState{TabIndex: newTab, ItemIndex: n.firstSelectable(newTab)}
}

// tabNext は次のタブへ循環して移り、カーソルを新タブの先頭選択可能行へ置く
func (n *tabMenuNav) tabNext(s TabMenuState) TabMenuState {
	if n.config.TabCount == 0 {
		return s
	}
	newTab := (s.TabIndex + 1) % n.config.TabCount
	return TabMenuState{TabIndex: newTab, ItemIndex: n.firstSelectable(newTab)}
}

// up はカーソルを1つ上へ動かす。端では循環し、見出し行は飛ばす
func (n *tabMenuNav) up(s TabMenuState) TabMenuState {
	count := n.itemCountForTab(s.TabIndex)
	if count == 0 {
		return s
	}
	next := (s.ItemIndex - 1 + count) % count
	return TabMenuState{TabIndex: s.TabIndex, ItemIndex: n.skipNext(s.TabIndex, next, -1)}
}

// down はカーソルを1つ下へ動かす。端では循環し、見出し行は飛ばす
func (n *tabMenuNav) down(s TabMenuState) TabMenuState {
	count := n.itemCountForTab(s.TabIndex)
	if count == 0 {
		return s
	}
	next := (s.ItemIndex + 1) % count
	return TabMenuState{TabIndex: s.TabIndex, ItemIndex: n.skipNext(s.TabIndex, next, 1)}
}

// left はページを1つ前へ繰る。ページ送りが無効か先頭ページなら何もしない
func (n *tabMenuNav) left(s TabMenuState) TabMenuState {
	if n.config.ItemsPerPage > 0 && s.ItemIndex >= n.config.ItemsPerPage {
		return TabMenuState{TabIndex: s.TabIndex, ItemIndex: n.skipNext(s.TabIndex, s.ItemIndex-n.config.ItemsPerPage, 1)}
	}
	return s
}

// right はページを1つ次へ繰る。ページ送りが無効か末尾ページなら何もしない
func (n *tabMenuNav) right(s TabMenuState) TabMenuState {
	count := n.itemCountForTab(s.TabIndex)
	if n.config.ItemsPerPage > 0 && s.ItemIndex+n.config.ItemsPerPage < count {
		return TabMenuState{TabIndex: s.TabIndex, ItemIndex: n.skipNext(s.TabIndex, s.ItemIndex+n.config.ItemsPerPage, 1)}
	}
	return s
}

// SetTab は指定タブへ直接カーソルを移す。新タブの先頭選択可能行へカーソルを置き、範囲外はクランプする。
// キー再生を避けてタブを設定する用途に使う。UseTabMenu と同じ nav ロジックを通す
func SetTab(store *Store, keyPrefix string, config TabMenuConfig, tab int) {
	nav := &tabMenuNav{config: config}
	store.states[keyPrefix] = nav.clamp(TabMenuState{TabIndex: tab, ItemIndex: nav.firstSelectable(tab)})
}

// DispatchNav はカーソル移動系の Action なら Dispatch して真を返し、それ以外は何もせず偽を返す。
// 呼び出し側は集合を知る必要がなく、消費されなかった Action をそのまま後続へ渡せばよい。
// 消費の可否は navHandlers の所属と一致するので、reduce の実行を待たず同期的に答えられる
func (m *Mount[Props]) DispatchNav(a inputmapper.ActionID) bool {
	if _, ok := navHandlers[a]; !ok {
		return false
	}
	m.Dispatch(a)
	return true
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
	state := nav.clamp(UseState(store, keyPrefix, init, nav.reduce))
	// クランプ後の値をstoreに反映して、UI側が参照する値と一致させる
	store.states[keyPrefix] = state

	// ページはitemIndexから派生する値として計算
	itemCount := nav.itemCountForTab(state.TabIndex)
	if nav.config.ItemsPerPage > 0 && itemCount > 0 {
		state.Page = state.ItemIndex / nav.config.ItemsPerPage
	}

	return state
}
