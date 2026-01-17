// Package messagelog はゲームメッセージログの表示ウィジェットを提供する。
//
// # Overview
//
// messagelogパッケージは、ゲーム内のログメッセージ（戦闘ログ、フィールドログ、
// シーンログなど）を画面に表示するためのUIウィジェットを提供します。
// ebitenUIとgamelogパッケージを統合し、色付きメッセージの表示、
// 自動スクロール、動的な更新機能を提供します。
//
// # Responsibilities
//
// messagelogパッケージの責務：
//   - ゲームログエントリの視覚的表示
//   - 色付きテキストフラグメントのレンダリング
//   - ログ更新時の自動UI再構築
//   - 表示行数制限とスクロール機能
//   - gamelog.SafeSliceとの統合
//   - ebitenUIコンポーネントとの統合
//
// # Usage
//
// 基本的な使用方法：
//
//	// ウィジェット設定
//	config := messagelog.WidgetConfig{
//		MaxLines:   5,
//		LineHeight: 20,
//		Spacing:    3,
//		Padding: messagelog.Insets{
//			Top: 2, Bottom: 2, Left: 2, Right: 2,
//		},
//	}
//
//	// ウィジェット作成
//	widget := messagelog.NewWidget(config, world)
//
//	// ログストア設定
//	widget.SetStore(gamelog.FieldLog)
//
//	// ゲームループ内で更新・描画
//	func Update() {
//		widget.Update()
//	}
//
//	func Draw(screen *ebiten.Image) {
//		widget.Draw(screen, x, y, width, height)
//	}
//
// # Design Principles
//
//   - Dependency Injection: ログストアを注入可能で、テスト時にモック使用
//   - Stateful Widget: 前回状態と比較して効率的な更新
//   - Configuration-Driven: 設定により外観をカスタマイズ可能
//   - EbitenUI Integration: ebitenUIの機能をフル活用
//   - Separation of Concerns: 表示ロジックのみに集中、背景描画は別パッケージ
//
// # 使い分け
//
//   - このパッケージはメッセージログ表示に特化
//   - 背景描画はwidgets/styledパッケージを使用
//   - HUDとの統合はwidgets/hudパッケージを使用
package messagelog
