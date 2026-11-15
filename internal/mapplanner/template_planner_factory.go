package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/maptemplate"
)

// NewPlannerChainByTemplateName は levels の絵 templateName でプランナーチェーンを作成する
func NewPlannerChainByTemplateName(templateName string, seed uint64) (*PlannerChain, error) {
	// テンプレートローダーを作成
	templateLoader := maptemplate.NewTemplateLoader()

	// すべてのチャンクを事前登録
	if err := templateLoader.RegisterAllChunks([]string{
		"levels/chunks",
		"levels/facilities",
		"levels/layouts",
	}); err != nil {
		return nil, fmt.Errorf("chunk registration error: %w", err)
	}

	// すべてのパレットを事前登録
	if err := templateLoader.RegisterAllPalettes([]string{
		"levels/palettes",
	}); err != nil {
		return nil, fmt.Errorf("palette registration error: %w", err)
	}

	// テンプレート名を指定して展開済みテンプレートとセル配列を取得
	template, _, resolvedMap, err := templateLoader.LoadTemplateByName(templateName, seed)
	if err != nil {
		return nil, fmt.Errorf("template load error: %w", err)
	}

	// テンプレートプランナーチェーンを作成
	chain, err := NewTemplatePlannerChain(template, resolvedMap, seed)
	if err != nil {
		return nil, err
	}

	return chain, nil
}
