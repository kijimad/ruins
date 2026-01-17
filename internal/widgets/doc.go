// Package widgets はビジネスロジックと状態管理を持つ高レベルUIコンポーネントを提供する。
//
// # Overview
//
// widgetsパッケージは、状態管理とビジネスロジックを持つ高レベルなUIコンポーネントを提供します。
// 再利用可能で、テスト可能な設計を重視し、宣言的な設定とコールバック機能により
// UIとアプリケーションロジックを疎結合で連携させます。
//
// # Package Hierarchy
//
// このプロジェクトのUIアーキテクチャは3層構造になっています：
//
//	widgets/     ← 業務ロジック付きの高レベルコンポーネント（このパッケージ）
//	   ↓ 使用
//	eui/         ← プロジェクト固有スタイルの中レベルコンポーネント
//	   ↓ 使用
//	ebitenui/    ← 外部ライブラリの低レベルコンポーネント
//
// # Responsibilities
//
// widgetsパッケージの責務：
//   - 状態管理を持つUIコンポーネントの提供
//   - キーボード・マウス操作の統一的な処理
//   - イベント駆動によるビジネスロジックとの連携
//   - 設定駆動による柔軟なコンポーネント構成
//   - 単体テストが可能な設計
//
// # Usage vs Other Packages
//
// ## widgetsパッケージを使う場合
//   - メニュー、ダイアログ、フォームなど複雑な操作が必要
//   - キーボードナビゲーションが必要
//   - 状態管理が必要（選択状態、入力データなど）
//   - ビジネスロジックとの連携が必要
//   - 単体テストを書きたい
//
// ## euiパッケージを使う場合
//   - 基本的なレイアウトコンテナが欲しい
//   - プロジェクト統一スタイルのボタンやテキストが欲しい
//   - 静的な表示のみで状態管理は不要
//   - 簡単なヘルパー関数で十分
//
// # Example
//
// widgetsパッケージの典型的な使用例：
//
//	// 設定でコンポーネントを構成
//	menu := menu.NewMenu(menu.MenuConfig{
//		Items: []tabmenu.Item{
//			{ID: "start", Label: "ゲーム開始"},
//			{ID: "exit", Label: "終了"},
//		},
//		WrapNavigation: true,
//	}, menu.MenuCallbacks{
//		OnSelect: func(index int, item tabmenu.Item) {
//			// ビジネスロジック
//			switch item.ID {
//			case "start":
//				startGame()
//			case "exit":
//				exitGame()
//			}
//		},
//	})
//
//	// ゲームループ内で更新
//	func Update() {
//		menu.Update(keyboardInput)
//	}
//
// # Design Principles
//
//   - Configuration over Code: 設定による宣言的なコンポーネント構成
//   - Testability: ロジックとUIの分離によるテスタビリティ
//   - Reusability: プロジェクト間で再利用可能な設計
//   - Separation of Concerns: UIとビジネスロジックの分離
//   - Event-Driven: コールバックによるイベント駆動アーキテクチャ
package widgets
