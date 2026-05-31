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
//	// ログストアを取得する（ECSシングルトンから）
//	store := worldhelper.GetGameLog(world)
//
//	// シンプルなログ
//	gamelog.New(store).
//	    Append("プレイヤーがアイテムを入手した").
//	    Log()
//
//	// 色付きログ
//	gamelog.New(store).
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
//   - Magic(text): マゼンタ色 - 魔法関連
//
// # ログストレージ
//
// ログストアはECSシングルトンエンティティとして保持されます。
// worldhelper.GetGameLog(world) で取得します。
//
// 色付きエントリの取得例：
//
//	store := worldhelper.GetGameLog(world)
//	entries := store.GetRecentEntries(5)
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
//	gamelog.New(store).
//	    ColorRGBA(consts.ColorBlue).
//	    Append("青色のテキスト").
//	    Log()
//
//	// カスタム色を作成
//	gamelog.New(store).
//	    ColorRGBA(consts.NamedColor(255, 0, 0)). // 赤色
//	    Append("カスタム色のテキスト").
//	    Log()
//
// # 使い分け
//
//   - GameLog: フィールドでの探索、アイテム入手、フロア移動などの継続的に表示されるログ
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
