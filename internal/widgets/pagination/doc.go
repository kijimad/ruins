// Package pagination はリスト表示のペジネーション機能を提供する
//
// # Overview
//
// paginationパッケージは、アイテムリストのページ分割表示に必要な
// 計算と状態管理を提供します。
//
// # Features
//
//   - ページ範囲の計算（開始/終了インデックス）
//   - ページナビゲーション状態（前/次ページの有無）
//   - ページインジケーターテキストの生成
//   - ジェネリックなスライス操作（表示範囲の抽出）
//
// # Basic Usage
//
//	// Paginationの作成
//	pg := pagination.New(itemIndex, itemCount, itemsPerPage)
//
//	// 表示範囲の取得
//	start, end := pg.GetVisibleRange()
//	visibleItems := items[start:end]
//
//	// または VisibleEntries を使用
//	for _, entry := range pagination.VisibleEntries(items, pg) {
//	    fmt.Printf("Index: %d, Item: %v\n", entry.Index, entry.Item)
//	}
//
//	// ページインジケーター
//	if pg.IsEnabled() {
//	    text := pg.GetIndicatorText() // "↑ 2/5 ↓"
//	}
package pagination
