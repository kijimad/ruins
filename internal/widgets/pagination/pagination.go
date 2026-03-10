package pagination

import "fmt"

// Pagination はペジネーション状態を管理する
type Pagination struct {
	ItemIndex    int // 現在選択中のアイテムインデックス（全体での位置）
	Page         int // 現在のページ（0ベース）
	ItemCount    int // アイテム総数
	ItemsPerPage int // 1ページあたりのアイテム数
}

// New はPaginationを作成する
// pageはitemIndexから自動計算される
func New(itemIndex, itemCount, itemsPerPage int) Pagination {
	page := 0
	if itemsPerPage > 0 && itemCount > 0 {
		page = itemIndex / itemsPerPage
	}
	return Pagination{
		ItemIndex:    itemIndex,
		Page:         page,
		ItemCount:    itemCount,
		ItemsPerPage: itemsPerPage,
	}
}

// GetCurrentPage は現在のページ番号を返す（表示用なので1ベース）
func (p Pagination) GetCurrentPage() int {
	return p.Page + 1
}

// GetTotalPages は総ページ数を返す
func (p Pagination) GetTotalPages() int {
	if p.ItemsPerPage <= 0 || p.ItemCount == 0 {
		return 1
	}
	return (p.ItemCount + p.ItemsPerPage - 1) / p.ItemsPerPage
}

// GetVisibleRange は現在のページで表示するアイテムの範囲を返す（start, end）
func (p Pagination) GetVisibleRange() (start, end int) {
	if p.ItemsPerPage <= 0 {
		return 0, p.ItemCount
	}
	start = p.Page * p.ItemsPerPage
	end = start + p.ItemsPerPage
	if end > p.ItemCount {
		end = p.ItemCount
	}
	if start >= p.ItemCount {
		return 0, 0
	}
	return start, end
}

// HasPreviousPage は前のページがあるかを返す
func (p Pagination) HasPreviousPage() bool {
	return p.Page > 0
}

// HasNextPage は次のページがあるかを返す
func (p Pagination) HasNextPage() bool {
	if p.ItemsPerPage <= 0 {
		return false
	}
	nextPageStart := (p.Page + 1) * p.ItemsPerPage
	return nextPageStart < p.ItemCount
}

// IsEnabled はペジネーションが有効か（複数ページあるか）を返す
func (p Pagination) IsEnabled() bool {
	return p.GetTotalPages() > 1
}

// IsSelectedInPage は指定インデックスが現在のページ内で選択中かを返す
func (p Pagination) IsSelectedInPage(index int) bool {
	return index == p.ItemIndex
}

// GetIndicatorText はページインジケーターのテキストを返す
// 例: "↑ 2/5 ↓"
func (p Pagination) GetIndicatorText() string {
	if !p.IsEnabled() {
		return ""
	}

	text := fmt.Sprintf("%d/%d", p.GetCurrentPage(), p.GetTotalPages())

	if p.HasPreviousPage() {
		text = "↑ " + text
	} else {
		text = "  " + text
	}

	if p.HasNextPage() {
		text = text + " ↓"
	}

	return text
}

// GetPageText はシンプルなページテキストを返す
// 例: "2/5"
func (p Pagination) GetPageText() string {
	if !p.IsEnabled() {
		return ""
	}
	return fmt.Sprintf("%d/%d", p.GetCurrentPage(), p.GetTotalPages())
}

// SliceVisible は任意のスライスから現在ページの要素を抽出する
func SliceVisible[T any](items []T, p Pagination) []T {
	start, end := p.GetVisibleRange()
	if start >= len(items) {
		return []T{}
	}
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// IndexedItem は元のインデックスを保持したアイテム
type IndexedItem[T any] struct {
	Index int
	Item  T
}

// VisibleEntries は現在ページの要素と元のインデックスを返す
func VisibleEntries[T any](items []T, p Pagination) []IndexedItem[T] {
	start, end := p.GetVisibleRange()
	if start >= len(items) {
		return nil
	}
	if end > len(items) {
		end = len(items)
	}

	result := make([]IndexedItem[T], end-start)
	for i := start; i < end; i++ {
		result[i-start] = IndexedItem[T]{
			Index: i,
			Item:  items[i],
		}
	}

	return result
}
