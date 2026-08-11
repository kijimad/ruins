package messagewindow

import (
	"fmt"
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

// getVisibleItems は指定ページで表示される項目とその元のインデックスを返す
func getVisibleItems(config tabMenuConfig, state viewState) ([]item, []int) {
	if len(config.Tabs) == 0 || state.TabIndex >= len(config.Tabs) {
		return []item{}, []int{}
	}

	currentTab := config.Tabs[state.TabIndex]

	if config.ItemsPerPage <= 0 {
		indices := make([]int, len(currentTab.Items))
		for i := range indices {
			indices[i] = i
		}
		return currentTab.Items, indices
	}

	page := currentPage(config, state)
	start := page * config.ItemsPerPage
	end := min(start+config.ItemsPerPage, len(currentTab.Items))

	if start >= len(currentTab.Items) {
		return []item{}, []int{}
	}

	visibleItems := currentTab.Items[start:end]
	indices := make([]int, len(visibleItems))
	for i := range indices {
		indices[i] = start + i
	}

	return visibleItems, indices
}

// currentPage は現在のページ番号を返す（0ベース）
func currentPage(config tabMenuConfig, state viewState) int {
	if config.ItemsPerPage <= 0 || state.ItemIndex < 0 {
		return 0
	}
	return state.ItemIndex / config.ItemsPerPage
}

// totalPages は総ページ数を返す
func totalPages(config tabMenuConfig, state viewState) int {
	if config.ItemsPerPage <= 0 {
		return 1
	}

	if len(config.Tabs) == 0 || state.TabIndex >= len(config.Tabs) {
		return 1
	}

	currentTab := config.Tabs[state.TabIndex]
	return (len(currentTab.Items) + config.ItemsPerPage - 1) / config.ItemsPerPage
}

// pageIndicatorText はページインジケーターのテキストを返す
func pageIndicatorText(config tabMenuConfig, state viewState) string {
	total := totalPages(config, state)
	if config.ItemsPerPage <= 0 || total <= 1 {
		return ""
	}

	// 位置は番号だけで示す。矢印は付けない。全メニューのページ表示を番号だけに揃える
	return fmt.Sprintf("%d/%d", currentPage(config, state)+1, total)
}
