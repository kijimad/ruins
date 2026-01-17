// Package movement provides movement validation logic shared between player and AI systems.
//
// このパッケージは移動判定の責務を持つ：
//   - エンティティ衝突チェック
//   - マップ境界チェック
//   - 通行可否判定
//
// 移動判定システムの設計:
//
// このゲームでは二重の通行可否システムが採用されています：
//
// ## 1. タイルレベルでの通行可否判定 (マップ生成時)
//
// マップ生成フェーズで使用される論理的な通行可否判定です。
//
//   - TileFloor, TileWarpNext, TileWarpEscape: 通行可能
//   - TileWall: 通行不可
//   - mapplanner.PathFinder.IsWalkable() で判定
//   - 用途: 接続性検証、部屋配置、コリドー生成
//
// ## 2. エンティティレベルでの通行可否判定 (実行時)
//
// ゲーム実行時に使用される動的な通行可否判定です。
//
//   - BlockPassコンポーネントを持つエンティティ: 通行不可
//   - movement.CanMoveTo() で判定
//   - 用途: プレイヤー・AI移動時の衝突チェック
//
// ## システム間の一貫性
//
// マップ生成時にタイルからエンティティへの変換が行われ、一貫性が保たれます：
//
//   - TileWall → BlockPass付きエンティティ (通行不可)
//   - TileFloor/TileWarpNext/TileWarpEscape → 通行可能エンティティ
//
// この設計により、マップ生成の柔軟性と実行時のパフォーマンスを両立しています。
//
// 循環importを避けるため、systemsパッケージとaiinputパッケージの両方から使用されます。
package movement
