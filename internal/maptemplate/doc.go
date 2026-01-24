// Package maptemplate はマップテンプレートの読み込み機能を提供します。
//
// このパッケージはTOMLファイルからチャンクテンプレートとパレット定義を読み込み、
// マップ生成システム（mapplanner）で使用可能なデータ構造に変換します。
//
// ## チャンクとは
//
// チャンクは再利用可能なマップの部品です。小さな部屋から大きな建物、
// さらには複数の建物を配置したレイアウトまで、すべて同じChunkTemplate型で表現されます。
// チャンクは他のチャンクを含むことができ、再帰的に組み合わせて複雑なマップを構築します。
//
// ## 責務
//
// - パレット定義の読み込み（地形・Propsのマッピング定義）
// - チャンクテンプレート定義の読み込み（ASCIIマップ、配置ルール）
// - 複数パレットのマージ処理
// - チャンクの再帰的展開
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
//	// ローダーの作成
//	loader := maptemplate.NewTemplateLoader()
//
//	// すべてのチャンクとパレットを事前登録
//	loader.RegisterAllChunks([]string{"assets/levels/chunks", "assets/levels/facilities"})
//	loader.RegisterAllPalettes([]string{"assets/levels/palettes"})
//
//	// テンプレート名で展開済みチャンクとパレットを取得
//	chunk, palette, err := loader.LoadTemplateByName("office_building", 12345)
package maptemplate
