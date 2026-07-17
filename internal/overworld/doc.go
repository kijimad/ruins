// Package overworld はシームレスワールドのチャンク生成を、worldstream の帯管理と
// 実 mapgen（mapplanner/mapspawner）へ橋渡しする。
//
// worldstream 自身は mapplanner/mapspawner に依存しない（ChunkGen 注入で分離）。本パッケージが
// その注入実装（決定的なチャンク seed と Plan→SpawnAt のアダプタ）を提供する。
// 詳細設計は docs/design/20260717_60.md §5。
package overworld
