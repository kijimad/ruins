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
//	uicore/      ← 保持型で宣言的な自前ツリー。Widget・Container・Text・Canvas
//	   ↓ 使用
//	ebiten/      ← 描画の最下層。EbitenCanvas が仲介する
//
// theme・styled を含めた層の全体像は widgets/menuframe のパッケージコメントを参照。
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
// ## widgets/uicore を直接使う場合
//   - 保持型のツリーを宣言的に組みたい
//   - 状態管理は呼び出し元が持ち、描画部品だけが要る
//   - Canvas を差し替えてテストしたい
//
// uicore を import できるパッケージは depguard の ui_core_inbound_guard が列挙する。
// widgets の部品はツリーを組むために全面を使い、画面 states・menuloop・systems は
// 組み上がったツリーを受け取って描くだけなので、名指しできる面を
// menuframe の TestScreenLayerUICoreSurface が許可制に絞る。
//
// # Sub-packages
//
// 代表的なサブパッケージ：
//   - styled       ← 最下層の描画部品。Cell・table・text と共通枠 chrome
//   - menuframe     ← タブ帯・パネル・モーダルの画面足場
//   - overlay       ← overlay 契約と詳細モーダル窓
//   - pagination    ← ページ計算の共通ロジック
//   - entityspec    ← エンティティを spec 行にする表示部品
//   - hud           ← HUD ウィジェット群
//   - messagewindow ← 会話ウィンドウと選択メニュー
//
// # Design Principles
//
//   - Configuration over Code: 設定による宣言的なコンポーネント構成
//   - Testability: ロジックとUIの分離によるテスタビリティ
//   - Reusability: プロジェクト間で再利用可能な設計
//   - Separation of Concerns: UIとビジネスロジックの分離
//   - Event-Driven: コールバックによるイベント駆動アーキテクチャ
package widgets
