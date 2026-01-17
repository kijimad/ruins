// Package resources はゲーム固有のリソース管理機能を提供する。
//
// このパッケージはruinsゲーム特有のデータ構造とロジックに特化しており、
// 純粋なリソース読み込み処理は internal/engine/loader パッケージで実装されている。
//
// 主な責務:
//   - ダンジョン管理（階層、探索状態、ミニマップ設定）
//   - ゲームステート遷移イベント管理
//   - ゲーム固有の型定義（TileIdx など）
//   - レベル計算とタイル座標変換
//
// 使い分け:
//   - resources: ゲーム固有のリソース管理（Dungeon、StateEventなど）
//   - engine/loader: 純粋なリソース読み込み処理（汎用的、再利用可能）
//
// 設計思想:
//   - ゲームロジックに特化した機能を提供
//   - 循環依存を防ぐため、依存関係を最小限に抑制
//   - engine/loaderに依存しない独立性を維持
//
// 使用例:
//
//	dungeon := &resources.Dungeon{
//	    ExploredTiles: make(map[gc.GridElement]bool),
//	    MinimapSettings: resources.MinimapSettings{...},
//	}
package resources
