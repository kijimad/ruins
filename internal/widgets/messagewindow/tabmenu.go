package messagewindow

import (
	"github.com/kijimaD/ruins/internal/widgets/pagination"
)

// tabItem はタブの項目を定義する
type tabItem struct {
	ID    string
	Label string
	Items []item
}

// tabMenuConfig はタブメニューの描画設定
type tabMenuConfig struct {
	Tabs         []tabItem
	ItemsPerPage int // 1ページに表示する項目数（0=制限なし）
}

// viewState は外部から設定される描画状態
type viewState struct {
	TabIndex  int
	ItemIndex int
}

// pageOf は現在のタブの項目に対するページ計算を返す。ページの割り方は pagination が持ち、
// ここでは選択肢一覧の形に合わせて渡すだけにする。
func pageOf(config tabMenuConfig, state viewState) ([]item, pagination.Pagination) {
	if len(config.Tabs) == 0 || state.TabIndex < 0 || state.TabIndex >= len(config.Tabs) {
		return nil, pagination.New(0, 0, config.ItemsPerPage)
	}
	items := config.Tabs[state.TabIndex].Items
	return items, pagination.New(state.ItemIndex, len(items), config.ItemsPerPage)
}

// getVisibleItems は指定ページで表示される項目とその元のインデックスを返す
func getVisibleItems(config tabMenuConfig, state viewState) ([]item, []int) {
	items, pg := pageOf(config, state)
	entries := pagination.VisibleEntries(items, pg)
	visible := make([]item, len(entries))
	indices := make([]int, len(entries))
	for i, e := range entries {
		visible[i] = e.Item
		indices[i] = e.Index
	}
	return visible, indices
}

// currentPage は現在のページ番号を返す（0ベース）
func currentPage(config tabMenuConfig, state viewState) int {
	_, pg := pageOf(config, state)
	return pg.Page
}

// totalPages は総ページ数を返す
func totalPages(config tabMenuConfig, state viewState) int {
	_, pg := pageOf(config, state)
	return pg.GetTotalPages()
}

// pageIndicatorText はページインジケーターのテキストを返す。1ページに収まるときは空になる
func pageIndicatorText(config tabMenuConfig, state viewState) string {
	_, pg := pageOf(config, state)
	return pg.GetPageText()
}
