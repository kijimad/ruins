// Package mapplanner provides map planning functionality.
//
// このパッケージは階層マップの生成機能を提供します：
//   - タイルベースのマップ生成
//   - 各種マップアルゴリズム（部屋、洞窟、森林、廃墟など）
//   - タイルとエンティティの配置計画作成
//
// ## マップ構造の概念
//
// マップは規定数のタイル（TileWidth × TileHeight）で正方形に構成されます。
// 各位置には必ず1つのタイルが存在し、タイル自体は不変です。
// タイルの上にエンティティ（壁、床、アイテム、NPCなど）を配置していきます。
//
// ## 主要データ構造
//
// - **MetaPlan**: マップ生成・エンティティ生成用の統一データ構造
//   - タイル配列（[]raw.TileRaw）、部屋情報、廊下情報、乱数生成器を含む
//   - PlannerChain内で段階的に構築される
//   - mapspawnerで直接ECSエンティティ生成に使用される
//
// ## タイル定義
//
// ### 基本タイルタイプ
// マップ生成で使用される標準タイルタイプ：
//   - planData.GenerateTile("Empty"): 空のタイル（デフォルト状態）
//   - planData.GenerateTile("Floor"): 床タイル（通行可能）
//   - planData.GenerateTile("Wall"): 壁タイル（通行不可）
//   - TileWater: 水タイル（通行可能だが特殊）
//   - TileDoor: 扉タイル（開閉可能な通路）
//   - TilePit: 落とし穴タイル（歩くと落下）
//
// ### TOMLベースタイル定義システム
// 新しいタイル定義システムでは、TOMLファイルでタイルの種類と属性を定義できます：
//
//	[[Tiles]]
//	Name = "Floor"
//	Description = "床タイル - 移動可能な基本的なタイル"
//	Walkable = true
//
// TileMasterクラスを使用してタイル定義を読み込み・管理：
//   - LoadTileFromFile(): TOMLファイルからタイル定義を読み込み
//   - GenerateTile(): 名前指定でタイルオブジェクトを生成
//
// ## エンティティ
//
// エンティティとして実装されます：
//   - 床タイル + ワープポータルエンティティ = ワープ機能のある場所
//
// ## 通行可否判定
//
// マップ生成時にはタイルの Walkable フィールドで通行可否を判定します：
//   - 通行可能: planData.GenerateTile("Floor")（Walkable=true）
//   - 通行不可: planData.GenerateTile("Wall")（Walkable=false）, planData.GenerateTile("Empty")（Walkable=false）
//
// ## マップ生成の流れ
//
// ### MetaPlan統一方式
// 1. タイル配列の初期化（全てplanData.GenerateTile("Empty")）
// 2. PlannerChainによる段階的タイル配置（MetaPlan）
// 3. mapspawner.SpawnFromMetaPlanで直接ECSエンティティ生成
//   - タイルタイプに応じて対応するエンティティ（床、壁、ワープホールなど）を配置
//   - NPCやアイテムなどの詳細なエンティティ配置も直接処理
//
// いずれの場合も実行時はエンティティベースの通行可否判定（movement.CanMoveTo）を使用
package mapplanner
