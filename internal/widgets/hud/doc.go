// Package hud はHUD（Head-Up Display）統合ウィジェットを提供する。
//
// # Overview
//
// hudパッケージは、ゲーム画面に表示されるHUD要素の統合と配置を管理します。
// 各種UIウィジェット（メッセージログ、ミニマップ、ステータス表示など）を
// 統一的に管理し、画面サイズに応じた適切な配置を行います。
//
// # Responsibilities
//
// # 描画の経路と意匠
//
// 描画はすべて internal/ui の Canvas 越しに行う。メニューと同じ面を通るので、記録用の実装で
// 検証でき、描画の下地を差し替えるときも1箇所で済む。
//
// パネルの枠はメニューと同じ9スライスのテクスチャを Chrome から敷く。矩形を組んで枠に見せる
// 方法は採らない。同じ「パネル」という概念に描き方が2つあると、意匠を変えるとき両方を直す
// ことになり、片方だけ直す事故が起きる。
//
// 図形の描画は持たない。矢印はアイコンフォントの字、地図の印は四角にして、記号の出どころを
// 文字とテクスチャの2つに絞る。ただしデバッグ表示の視界円だけは例外で、世界の座標に従う
// 多角形なので描画先を直に受け取る。
//
// hudパッケージの責務：
//   - HUD要素の統合管理
//   - 画面サイズに応じた配置計算
//   - ゲームシステム（World）との統合
//   - 各ウィジェットの更新順序制御
//   - HUD要素の有効/無効制御
//
// # Usage
//
// 基本的な使用方法：
//
//	// HUDメッセージエリア作成
//	messageArea := hud.NewMessageArea(world)
//
//	// ゲームループ内で更新・描画
//	func Update(world w.World) {
//		messageArea.Update(world)
//	}
//
//	func Draw(world w.World, screen *ebiten.Image) {
//		messageArea.Draw(world, screen)
//	}
//
// # Design Principles
//
//   - System Integration: ゲームシステムとウィジェットの橋渡し
//   - Responsive Layout: 画面サイズに応じた動的配置
//   - Widget Composition: 複数ウィジェットの組み合わせ
//   - Configuration-Driven: 設定による外観カスタマイズ
//   - World-Aware: ゲームワールド状態を考慮した表示制御
//
// # 使い分け
//
//   - このパッケージはHUDシステムとの統合に特化
//   - 個別ウィジェット機能はwidgets/messagelogやwidgets/styledを使用
//   - ゲームロジックとの統合が必要な場合に使用
package hud
