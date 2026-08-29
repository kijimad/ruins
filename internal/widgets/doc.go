// Package widgets は状態と操作を持つ高レベルの UI 部品を提供する。
//
// # 層
//
//	画面 states・menuloop
//	   ↓ 使用
//	widgets      ← 状態と操作を持つ部品。このパッケージ
//	   ↓ 使用
//	uicore       ← 保持型ツリーの実体。Widget・Container・Text・Canvas
//	   ↓ 使用
//	ebiten       ← 描画の最下層。EbitenCanvas が仲介する
//
// theme・styled を含めた全体像は widgets/menuframe のパッケージコメントを参照。
//
// # uicore の使用範囲
//
// uicore を import できるパッケージは depguard の ui_core_inbound_guard が列挙する。
// 部品はツリーを組むので全面を使う。画面 states・menuloop・systems は組み上がった
// ツリーを受け取って描くだけなので、名指しできるシンボルを menuframe の
// TestScreenLayerUICoreSurface が許可制に絞る。
//
// # サブパッケージ
//
//   - theme         ← 色と寸法のトークン。依存ゼロ
//   - styled        ← 一覧のデータ型 Cell・table・text と共通枠 chrome
//   - uicore        ← 保持型ツリーの実体
//   - menuframe     ← タブ帯・パネル・モーダルの画面足場
//   - overlay       ← overlay 契約と詳細モーダル窓
//   - pagination    ← ページ計算
//   - entityspec    ← エンティティを spec 行にする表示部品
//   - hud           ← HUD ウィジェット群
//   - messagelog    ← 色付きメッセージログ
//   - messagewindow ← 会話ウィンドウと選択メニュー
package widgets
