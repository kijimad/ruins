// Package worldstream は無限シームレスワールドの「アクティブ帯」を east 方向へ
// ストリーミングするための基盤を提供する。
//
// K 個の隣接チャンクを1連続座標空間に並べた単一マップ、すなわち帯を管理する。プレイヤーが
// 中央チャンクを東へ出るとシフトし、東端生成・西端破棄・座標リベースを行う。これにより帯ローカル
// 座標は常に有界に保たれ、既存の単一マップ機構を変えずに無限東進を実現する。
//
// 主な構成:
//   - TranslateAllEntities / RemoveEntitiesInXRange: 帯シフトの原子操作。純 ECS
//   - AbsTileX / ToAbs / ToLocal: 絶対軸 X と帯ローカル座標の分離・変換
//   - Band: 帯の状態と ShiftEast/ShiftWest によるシフト
//
// mapplanner/mapspawner には依存しない。チャンク生成は ChunkGen 注入で分離する。実生成の
// アダプタは internal/overworld が提供する。
package worldstream
