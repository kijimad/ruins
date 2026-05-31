// Package loader はゲームリソースの読み込み機能を提供する。
//
// 主な責務:
//   - TOMLファイルからのメタデータ読み込み
//   - フォント、スプライトシート、Rawデータの読み込み
//
// 使い分け:
//   - loader: 純粋なリソース読み込み処理
//   - resources: ゲーム固有のリソース管理
//
// 使用例:
//
//	rw, err := loader.LoadRaws()
//	sprites, err := loader.LoadSpriteSheets(rw)
//	fonts, err := loader.LoadFonts()
package loader
