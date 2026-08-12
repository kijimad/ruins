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
//   - 敵・中立NPCはsoloPlannerが状態遷移とアクション計画をインラインで処理する
//   - 遠方の非交戦AIは距離カリングで処理対象から外す
//
// # 使い分け
//   - Processor: AIシステム全体の処理制御。ProcessAllで全AIエンティティを処理する
//   - Planner: 行動決定インターフェース。soloPlannerが実装する
//   - VisionSystem: 視界判定
package aiinput
