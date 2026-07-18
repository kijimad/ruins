// Package overworld はオーバーワールドのチャンク生成を、worldstream の帯管理と
// 実 mapgen の mapplanner/mapspawner へ橋渡しする。
//
// worldstream 自身は mapplanner/mapspawner に依存せず、ChunkGen 注入で分離している。本パッケージが
// その注入実装を提供する。実装は決定的なチャンク seed と Plan→SpawnAt のアダプタからなる。
// 詳細設計は docs/design/20260717_60.md §5。
package overworld
