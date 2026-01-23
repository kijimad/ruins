package mapplanner

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/maptemplate"
)

// NewTemplateDebugPlanner はデバッグ用のテンプレートマップを生成するプランナーを返す
// 固定のテンプレート（small_room.toml）を使用する
// width, heightパラメータはPlannerFuncインターフェースに合わせるために存在するが、テンプレートサイズはTOMLファイルで定義されるため使用しない
func NewTemplateDebugPlanner(_ gc.Tile, _ gc.Tile, seed uint64) (*PlannerChain, error) {
	// パレットを読み込む
	paletteLoader := maptemplate.NewPaletteLoader()
	palette, err := paletteLoader.LoadFromFile("assets/levels/palettes/standard.toml")
	if err != nil {
		return nil, err
	}

	// テンプレートを読み込む
	templateLoader := maptemplate.NewTemplateLoader()
	templates, err := templateLoader.LoadFromFile("assets/levels/facilities/small_room.toml")
	if err != nil {
		return nil, err
	}

	if len(templates) == 0 {
		return nil, err
	}

	// 最初のテンプレートを使用
	template := &templates[0]

	// テンプレートプランナーチェーンを作成
	return NewTemplatePlannerChain(template, palette, seed)
}
