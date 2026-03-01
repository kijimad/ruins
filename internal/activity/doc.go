// Package activity はアクションとアクティビティの実装を提供する。
//
// # 概要
//
// このパッケージは、ゲーム内のあらゆるアクション（移動、攻撃、休息など）の
// 具体的な実装を提供する。CDDAスタイルの中断可能なアクティビティシステムを採用し、
// 即座実行と継続実行の両方のアクションを統一的に管理する。
//
// # 責務
//
// - アクションの具体的な実装とロジック
// - アクティビティライフサイクル管理（開始、実行、中断、再開、完了）
// - アクションコストの定義と管理
// - ターン管理システムとの連携
//
// # パッケージレベル関数
//
// このパッケージはパッケージレベル関数を提供し、アクションを統一的に管理する：
// - Execute: アクションの実行エントリーポイント
// - StartActivity: 継続アクティビティの開始
// - InterruptActivity: アクティビティの中断
// - ResumeActivity: アクティビティの再開
// - CancelActivity: アクティビティのキャンセル
// - ProcessTurn: ターン毎のアクティビティ処理
// - GetLastResult: 直近のアクティビティ実行結果を取得
//
// 責務：
// - アクティビティの作成（パラメータからActivityを生成）
// - 実行中アクティビティの状態管理
// - ライフサイクル制御（Start, DoTurn, Finish, Canceled）
// - 実行可能性の検証
// - ターン管理システムとの連携（APコスト消費）
// - 継続実行と中断・再開の管理
//
// ## 使用方法
//
// **外部システムからの使用:**
// - systems/tile_input_system.go から使用（プレイヤー入力処理）
// - aiinput/processor.go から使用（AI行動処理）
//
// ```go
// result, err := activity.Execute(activityImpl, params, world)
// ```
//
// ### 個別Activity実装
// - **MoveActivity**: 移動アクション（即座実行）
// - **AttackActivity**: 攻撃アクション（即座実行）
// - **RestActivity**: 休息アクション（継続実行、中断可能）
// - **WaitActivity**: 待機アクション（継続実行、中断可能）
//
// # 他パッケージとの関係
//
// ```
// systems → activity.Execute() → アクション実行
//
//	↓
//
// activity → turns.ConsumePlayerMoves → コスト消費
// ```
//
// ## 責務の境界
//
// - **activity**: どのような行動をするか（What Action & How）
// - **turns**: いつ実行するかの制御（When）
// - **systems**: 何の入力から実行するか（What Input）
//
// # アーキテクチャ
//
// ## 2層構造
//
// ### 1. 即座実行アクション（短期間で完了）
// - 移動、攻撃、アイテム拾得など
// - `TurnsTotal = 1`で残りAP1でも1ターンで完了
// - シンプルな実行ロジック
//
// ### 2. 継続実行アクション（複数ターンにわたる）
// - 休息、読書、クラフトなど
// - `TurnsTotal > 1`で段階的に実行
// - 中断・再開機能あり
//
// ## 統一インターフェース
//
// 全てのアクションは`Activity`構造体を通じて統一的に管理される：
//
//	type Activity struct {
//		Type       ActivityType  // アクション種別
//		State      State // 実行状態
//		TurnsTotal int          // 必要ターン数
//		TurnsLeft  int          // 残りターン数
//		// ...
//	}
//
// # 設計原則
//
// 1. **統一性**: 全アクションを同じインターフェースで管理
// 2. **拡張性**: 新しいアクションの追加が容易
// 3. **中断可能性**: 必要に応じてアクションを中断・再開
// 4. **責務分離**: アクションロジックとターン管理を分離
// 5. **検証**: 実行前・再開前の条件チェック
//
// # 使用例
//
//	// パッケージレベル関数を通じた統一的なアクション実行
//
//	// 即座実行アクション（移動）
//	params := activity.ActionParams{Actor: player, Destination: &dest}
//	result, err := activity.Execute(&activity.MoveActivity{}, params, world)
//
//	// 継続実行アクション（休息）
//	params := activity.ActionParams{Actor: player, Duration: 10}
//	result, err := activity.Execute(&activity.RestActivity{}, params, world)
//
//	// アクティビティの管理
//	activity.InterruptActivity(player, "戦闘開始", world)
//	activity.ResumeActivity(player, world)
//
//	// ターン毎の処理
//	activity.ProcessTurn(world)
//
//	// 直近の結果を取得
//	lastResult := activity.GetLastResult(player, world)
//
// # CDDAとの対応関係
//
// このパッケージの設計は Cataclysm: Dark Days Ahead の activity_actor システムを参考にしている：
//
// - CDDAのactivity_actor → Activity構造体
// - CDDAのdo_turn() → DoTurn()メソッド
// - CDDAのfinish() → Complete()メソッド
// - CDDAのcanceled() → Interrupt()メソッド
// - CDDAのmove_cost → アクションコスト概念
//
// # 拡張方法
//
// 新しいアクションを追加する場合：
//
// 1. ActivityTypeに新しい定数を追加
// 2. activityInfosに情報を追加
// 3. 具体的な実装ファイルを作成（例：new_action.go）
// 4. Activity インターフェースを実装
// 5. 必要に応じてActivity.DoTurnに処理を追加
package activity
