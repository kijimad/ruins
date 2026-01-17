// Package messagedata はメッセージウィンドウに表示するデータ構造を提供する
//
// # 概要
//
// このパッケージは、ゲーム内のメッセージウィンドウシステムで使用するデータ構造と
// ユーティリティ関数を提供します。会話メッセージとシステムメッセージ、選択肢、
// メッセージ連鎖などを統一的に管理し、UI層から独立したメッセージデータの操作を可能にします。
//
// # 主要な構造体
//
//   - MessageData: メッセージの内容と動作を表すデータ構造
//   - Choice: 選択肢のデータ（テキスト、アクション、次のメッセージ）
//   - Size: カスタムサイズ情報
//
// # メッセージの種類
//
//	// 会話メッセージ（キャラクター同士の対話）
//	dialog := messagedata.NewDialogMessage("元気ですか？", "村人")
//
//	// システムメッセージ（ゲームからの通知、話者は"システム"）
//	system := messagedata.NewSystemMessage("アイテムを入手しました")
//
// # 選択肢機能
//
//	// 単純な選択肢（アクションのみ）
//	msg := messagedata.NewDialogMessage("どうしますか？", "").
//	    WithChoice("はい", func(_ w.World) { fmt.Println("はいが選ばれました") }).
//	    WithChoice("いいえ", func(_ w.World) { fmt.Println("いいえが選ばれました") })
//
//	// 選択肢から別のメッセージへの分岐
//	battleResult := messagedata.NewSystemMessage("戦闘開始").
//	    SystemMessage("激しい戦闘").
//	    SystemMessage("勝利！")
//
//	escapeResult := messagedata.NewSystemMessage("逃走成功").
//	    SystemMessage("安全な場所に到着")
//
//	encounter := messagedata.NewDialogMessage("敵に遭遇した！", "").
//	    WithChoiceMessage("戦う", battleResult).
//	    WithChoiceMessage("逃げる", escapeResult)
//
// # メッセージ連鎖
//
// メッセージを連鎖させて、複数のメッセージを順次表示できます。
//
//	// 複数メッセージの連鎖
//	story := messagedata.NewSystemMessage("物語が始まる").
//	    DialogMessage("こんにちは", "主人公").
//	    DialogMessage("元気ですね", "村人").
//	    SystemMessage("会話が終了しました")
//
// # アーキテクチャ上の責務
//
// このパッケージは **データモデル層** として以下の責務を持ちます：
//
//   - メッセージデータ構造の定義と管理
//   - ビジネスロジック（メッセージ連鎖、選択肢分岐）の実装
//   - UI実装に依存しないピュアなデータ操作
//   - 異なるプレゼンテーション層で再利用可能なデータ提供
//
// 対して messagewindow パッケージは **プレゼンテーション層** として：
//
//   - UI描画とユーザー入力処理
//   - 画面表示ロジック（レイアウト、スタイリング）
//   - Ebitenエンジン固有の実装詳細
//   - このパッケージのMessageDataを使用してUI構築
//
// この分離により以下のメリットを実現：
//
//   - 責務の明確化: データとUIの関心事を分離
//   - 再利用性: 同じメッセージデータで異なるUI実装が可能
//   - テスタビリティ: UI依存を排除した軽量なユニットテスト
//   - 拡張性: UI変更時にデータ層への影響を最小化
//
// # 設計原則
//
//   - シンプルさ: 会話とシステムの2種類のメッセージに集約
//   - UI層からの独立: メッセージデータはUI実装に依存しない
//   - 連鎖性: ビルダーパターンによる直感的なAPI
//   - 選択肢分岐: 複雑なメッセージフローに対応
//
// # 使用例
//
//	// 複雑な選択肢分岐の例
//	questMessage := messagedata.NewDialogMessage("クエストを受けますか？", "依頼人").
//	    WithChoiceMessage("受ける",
//	        messagedata.NewSystemMessage("クエスト開始").
//	            SystemMessage("目標: 魔物を倒す")).
//	    WithChoiceMessage("断る",
//	        messagedata.NewDialogMessage("またいつでもどうぞ", "依頼人"))
package messagedata
