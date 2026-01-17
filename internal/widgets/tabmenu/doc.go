// Package tabmenu はキーボードナビゲーション用のタブ付きメニューコンポーネントを提供する。
//
// TabMenuコンポーネントは、複数のタブを持つメニューシステムを提供します。
// キーボードでのナビゲーションに最適化されており、以下の機能を持ちます：
//
// 機能:
// - タブ切り替え（左右矢印キー、Tab/Shift+Tab）
// - アイテム選択（上下矢印キー）
// - Enterキーでの選択
// - Escapeキーでのキャンセル
// - 循環ナビゲーション（オプション）
// - 異なるキーのみ受け付けるモード（グローバルキー重複防止）
//
// 使い分け:
// - インベントリ画面のようなカテゴリ分けされたアイテム一覧
// - 設定画面のタブ切り替え
// - 複数カテゴリを持つ任意のメニューシステム
//
// 責務:
// - タブとアイテムの階層的なナビゲーション
// - キーボード入力の処理とコールバック通知
// - 状態管理（現在のタブ・アイテムインデックス）
// - 動的なタブ・アイテム更新への対応
//
// 使用例:
//
//	tabs := []tabmenu.TabItem{
//		{
//			ID:    "items",
//			Label: "道具",
//			Items: []tabmenu.Item{
//				{ID: "potion", Label: "ポーション"},
//				{ID: "scroll", Label: "巻物"},
//			},
//		},
//		{
//			ID:    "equipment",
//			Label: "装備",
//			Items: []tabmenu.Item{
//				{ID: "sword", Label: "剣"},
//				{ID: "armor", Label: "鎧"},
//			},
//		},
//	}
//
//	config := tabmenu.TabMenuConfig{
//		Tabs:              tabs,
//		InitialTabIndex:   0,
//		InitialItemIndex:  0,
//		WrapNavigation:    true,
//		OnlyDifferentKeys: true,
//	}
//
//	callbacks := tabmenu.TabMenuCallbacks{
//		OnSelectItem: func(tabIndex int, itemIndex int, tab tabmenu.TabItem, item tabmenu.Item) {
//			// アイテム選択時の処理
//		},
//		OnTabChange: func(oldTabIndex, newTabIndex int, tab tabmenu.TabItem) {
//			// タブ変更時の処理
//		},
//		OnCancel: func() {
//			// キャンセル時の処理
//		},
//	}
//
//	keyboardInput := input.GetSharedKeyboardInput()
//	tabMenu := tabmenu.NewTabMenu(config, callbacks, keyboardInput)
//
//	// ゲームループ内で
//	tabMenu.Update()
package tabmenu
