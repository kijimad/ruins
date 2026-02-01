package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/maptemplate"
)

// TemplateType はテンプレートの種類を表す
type TemplateType int

const (
	// TemplateTypeSmallRoom は小部屋テンプレート
	TemplateTypeSmallRoom TemplateType = iota
	// TemplateTypeOfficeBuilding は事務所ビルテンプレート
	TemplateTypeOfficeBuilding
	// TemplateTypeSmallTown は小さな町
	TemplateTypeSmallTown
	// TemplateTypeTownPlaza は町の広場
	TemplateTypeTownPlaza
)

// NewPlannerChainByTemplateType は指定されたテンプレートタイプでプランナーチェーンを作成する
func NewPlannerChainByTemplateType(templateType TemplateType, seed uint64) (*PlannerChain, error) {
	// テンプレートローダーを作成
	templateLoader := maptemplate.NewTemplateLoader()

	// すべてのチャンクを事前登録
	if err := templateLoader.RegisterAllChunks([]string{
		"levels/chunks",
		"levels/facilities",
		"levels/layouts",
	}); err != nil {
		return nil, fmt.Errorf("チャンク登録エラー: %w", err)
	}

	// すべてのパレットを事前登録
	if err := templateLoader.RegisterAllPalettes([]string{
		"levels/palettes",
	}); err != nil {
		return nil, fmt.Errorf("パレット登録エラー: %w", err)
	}

	// テンプレート名を決定
	var templateName string
	switch templateType {
	case TemplateTypeSmallRoom:
		templateName = "10x10_small_room"
	case TemplateTypeOfficeBuilding:
		templateName = "15x12_office_building"
	case TemplateTypeSmallTown:
		templateName = "50x50_small_town"
	case TemplateTypeTownPlaza:
		templateName = "20x20_town_plaza"
	default:
		return nil, fmt.Errorf("未知のテンプレートタイプ: %d", templateType)
	}

	// テンプレート名を指定して展開済みテンプレートとパレットを取得
	template, palette, err := templateLoader.LoadTemplateByName(templateName, seed)
	if err != nil {
		return nil, fmt.Errorf("テンプレート読み込みエラー: %w", err)
	}

	// テンプレートプランナーチェーンを作成
	chain, err := NewTemplatePlannerChain(template, palette, seed)
	if err != nil {
		return nil, err
	}

	// 橋facilityはPlan関数で統一的に追加されるため、ここでは追加しない

	return chain, nil
}
