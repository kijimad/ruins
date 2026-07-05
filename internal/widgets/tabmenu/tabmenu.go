package tabmenu

import "fmt"

// TabItem はタブの項目を定義する
type TabItem struct {
	ID    string
	Label string
	Items []Item
}

// Config はタブメニューの描画設定
type Config struct {
	Tabs         []TabItem
	ItemsPerPage int // 1ページに表示する項目数（0=制限なし）
}

// ViewState は外部から設定される描画状態
type ViewState struct {
	TabIndex  int
	ItemIndex int
}

// getVisibleItems は指定ページで表示される項目とその元のインデックスを返す
func getVisibleItems(config Config, state ViewState) ([]Item, []int) {
	if len(config.Tabs) == 0 || state.TabIndex >= len(config.Tabs) {
		return []Item{}, []int{}
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
		return []Item{}, []int{}
	}

	visibleItems := currentTab.Items[start:end]
	indices := make([]int, len(visibleItems))
	for i := range indices {
		indices[i] = start + i
	}

	return visibleItems, indices
}

// currentPage は現在のページ番号を返す（0ベース）
func currentPage(config Config, state ViewState) int {
	if config.ItemsPerPage <= 0 || state.ItemIndex < 0 {
		return 0
	}
	return state.ItemIndex / config.ItemsPerPage
}

// totalPages は総ページ数を返す
func totalPages(config Config, state ViewState) int {
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
func pageIndicatorText(config Config, state ViewState) string {
	total := totalPages(config, state)
	if config.ItemsPerPage <= 0 || total <= 1 {
		return ""
	}

	page := currentPage(config, state)
	arrows := ""

	if page > 0 {
		arrows += " ↑"
	} else {
		arrows += " 　"
	}

	if (page+1)*config.ItemsPerPage < len(config.Tabs[state.TabIndex].Items) {
		arrows += " ↓"
	} else {
		arrows += " 　"
	}

	return fmt.Sprintf("%d/%d%s", page+1, total, arrows)
}
