// Package aiinput はAIエンティティの行動決定と処理を提供する。
//
// # 責務
//   - AIエンティティの行動決定ロジック
//   - 状態遷移と行動計画の統合処理
//   - 視界判定とプレイヤー検出
//   - ターンベースのAP消費ループ
//
// # 仕様
//   - Plannerインターフェースで行動決定を抽象化し、runAPLoopで統一的にAP消費ループを実行する
//   - 敵・中立NPCはroamingPlannerが状態遷移とアクション計画をインラインで処理する
//   - 隊員はsquadPlannerが優先度チェーンで行動を決定する
//   - 処理順序は敵→隊員の2フェーズで、隊員は敵の移動結果を反映した判断ができる
//
// # 使い分け
//   - Processor: AIシステム全体の処理制御。ProcessAllで全AIエンティティを処理する
//   - Planner: 行動決定インターフェース。roamingPlannerとsquadPlannerが実装する
//   - VisionSystem: 視界判定
package aiinput
