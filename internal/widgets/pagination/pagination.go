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
// pageはitemIndexから自動計算される。
// カーソルを持たない一覧は負の itemIndex を渡す。どのページにも属さないので先頭ページを返す
func New(itemIndex, itemCount, itemsPerPage int) Pagination {
	page := 0
	if itemsPerPage > 0 && itemCount > 0 && itemIndex > 0 {
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
	// アイテムが無いか負数のときは1ページに丸める。負数は総数として不正だが、
	// 呼び出し側にガードを書かせず、ページ計算の所有者であるここで吸収する
	if p.ItemsPerPage <= 0 || p.ItemCount <= 0 {
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
	end = min(start+p.ItemsPerPage, p.ItemCount)
	if start >= p.ItemCount {
		return 0, 0
	}
	return start, end
}

// IsEnabled はペジネーションが有効か（複数ページあるか）を返す
func (p Pagination) IsEnabled() bool {
	return p.GetTotalPages() > 1
}

// IsSelectedInPage は指定インデックスが現在のページ内で選択中かを返す
func (p Pagination) IsSelectedInPage(index int) bool {
	return index == p.ItemIndex
}

// GetPageText はシンプルなページテキストを返す
// 例: "2/5"
func (p Pagination) GetPageText() string {
	if !p.IsEnabled() {
		return ""
	}
	return fmt.Sprintf("%d/%d", p.GetCurrentPage(), p.GetTotalPages())
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
