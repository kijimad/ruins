// Package gamelog はゲームログ機能を提供する。
//
// このパッケージは、RPG風ゲームに最適化された色付きログシステムを提供します。
// メソッドチェーンによる直感的なログ作成、プリセット関数による統一的な色付け、
// スレッドセーフなログストレージを特徴としています。
//
// # 主な機能
//
//   - メソッドチェーンによる直感的なログ作成
//   - 色付きテキストフラグメント
//   - プリセット関数による統一的な色付け
//   - スレッドセーフなログストレージ
//
// # 基本的な使い方
//
//	// シンプルなログ
//	gamelog.New(gamelog.FieldLog).
//	    Append("プレイヤーがアイテムを入手した").
//	    Log()
//
//	// 色付きログ
//	gamelog.New(gamelog.FieldLog).
//	    PlayerName("Hero").
//	    Append("が").
//	    ItemName("Iron Sword").
//	    Append("を入手した。").
//	    Log()
//
// # プリセット関数
//
// ## 基本プリセット
//
//   - Success(text): 緑色 - 成功メッセージ
//   - Warning(text): 黄色 - 警告メッセージ
//   - Error(text): 赤色 - エラーメッセージ
//   - System(text): 水色 - システムメッセージ
//
// ## ゲーム要素プリセット
//
//   - PlayerName(name): 緑色 - プレイヤー名
//   - NPCName(name): 黄色 - NPC名
//   - ItemName(item): シアン色 - アイテム名
//   - Location(place): オレンジ色 - 場所名
//   - Action(action): 紫色 - アクション名
//   - Money(amount): 黄色 - 金額
//   - Damage(num): 赤色 - ダメージ数値
//
// ## 戦闘専用プリセット
//
//   - Encounter(text): 赤色 - 敵との遭遇
//   - Victory(text): 緑色 - 勝利メッセージ
//   - Defeat(text): 赤色 - 敗北メッセージ
//   - Magic(text): 紫色 - 魔法関連
//
// # ログストレージ
//
// パッケージは2つのグローバルログストレージを提供します：
//
//   - FieldLog: フィールド探索ログ用
//   - SceneLog: シーンログ用（会話やイベント時の一時的なメッセージ）
//
// 色付きエントリの取得例：
//
//	entries := gamelog.FieldLog.GetRecentEntries(5)
//	for _, entry := range entries {
//	    for _, fragment := range entry.Fragments {
//	        // fragment.Text と fragment.Color を使用
//	    }
//	}
//
// # カスタム色
//
//	import "github.com/kijimaD/ruins/internal/consts"
//
//	// 定義済み色を使用
//	gamelog.New(gamelog.FieldLog).
//	    ColorRGBA(consts.ColorBlue).
//	    Append("青色のテキスト").
//	    Log()
//
//	// カスタム色を作成
//	gamelog.New(gamelog.FieldLog).
//	    ColorRGBA(consts.NamedColor(255, 0, 0)). // 赤色
//	    Append("カスタム色のテキスト").
//	    Log()
//
// # 使い分け
//
//   - FieldLog: フィールドでの探索、アイテム入手、フロア移動などの継続的に表示されるログ
//   - SceneLog: 会話シーンやイベント中の一時的なメッセージ（表示後にクリアされる）
//
// # 責務
//
//   - ゲーム内イベントの色付きログ管理
//   - メソッドチェーンによる直感的なログ作成API提供
//   - 各種ゲーム要素に最適化されたプリセット関数提供
//   - スレッドセーフなログストレージ機能
//
// # 仕様
//
//   - フラグメント単位での色指定
//   - 最大ログサイズによる自動ローテーション
//   - 並行アクセス対応（mutex使用）
//   - JSON形式でのシリアライズ対応
package gamelog
