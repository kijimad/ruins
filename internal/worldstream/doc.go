// Package worldstream は無限シームレスワールドの「アクティブ帯」を north 方向へ
// ストリーミングするための基盤を提供する。北は画面の上すなわち -Y。
//
// rows 個の隣接チャンクを1連続座標空間に並べた単一マップ、すなわち帯を管理する。プレイヤーが
// 中央チャンクを北へ出るとシフトし、北端生成・南端破棄・座標リベースを行う。これにより帯ローカル
// 座標は常に有界に保たれ、既存の単一マップ機構を変えずに無限北進を実現する。
//
// 主な構成:
//   - TranslateAllEntities / RemoveEntitiesInYRange: 帯シフトの原子操作。純 ECS
//   - ToAbsY / ToLocalY: 絶対軸 Y（consts.AbsTileY）と帯ローカル座標の分離・変換
//   - Band: 帯の状態と ShiftNorth による北進
//
// mapplanner/mapspawner には依存しない。チャンク生成は ChunkGen 注入で分離する。実生成の
// アダプタは internal/overworld が提供する。
//
// チャンク座標は consts.Coord[consts.Chunk] をそのまま使う。Y は南北の絶対チャンクインデックスで
// 北へ無限に負へ伸び、X は帯の列 [0, Cols) に有界でストリーミングしない。比較可能な値型なので等値比較や
// map のキーにそのまま使える。
package worldstream
