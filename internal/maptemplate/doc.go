// Package maptemplate はマップテンプレートの読み込み機能を提供します。
//
// このパッケージはTOMLファイルから施設テンプレートとパレット定義を読み込み、
// マップ生成システム（mapplanner）で使用可能なデータ構造に変換します。
//
// ## 責務
//
// - パレット定義の読み込み（地形・家具のマッピング定義）
// - 施設テンプレート定義の読み込み（ASCIIマップ、配置ルール）
// - 複数パレットのマージ処理
// - バリデーション（マップサイズ、文字の妥当性）
//
// ## 使い分け
//
// - **maptemplate**: TOMLファイルの読み込みとデータ構造の提供
// - **mapplanner**: テンプレートを使った実際のマップ生成処理
// - **mapspawner**: 生成されたマップからECSエンティティの生成
//
// ## 基本的な使用例
//
//	// パレットの読み込み
//	paletteLoader := maptemplate.NewPaletteLoader()
//	palette, err := paletteLoader.LoadFromFile("assets/mapgen/palettes/standard.toml")
//
//	// 施設テンプレートの読み込み
//	templateLoader := maptemplate.NewTemplateLoader()
//	templates, err := templateLoader.LoadFromFile("assets/mapgen/facilities/military_factory.toml")
//
//	// パレットの適用
//	merged := maptemplate.MergePalettes(palette, militaryPalette)
//	facility := templates[0]
//	facility.ApplyPalette(merged)
package maptemplate
