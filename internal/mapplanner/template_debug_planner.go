package mapplanner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/maptemplate"
)

// TemplateType はデバッグで使用するテンプレートの種類を表す
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

// NewTemplateDebugPlanner はデバッグ用のテンプレートマップを生成するプランナーを返す
// 固定のテンプレート（small_room.toml）を使用する
// width, heightパラメータはPlannerFuncインターフェースに合わせるために存在するが、テンプレートサイズはTOMLファイルで定義されるため使用しない
func NewTemplateDebugPlanner(_ gc.Tile, _ gc.Tile, seed uint64) (*PlannerChain, error) {
	return NewTemplateDebugPlannerWithType(TemplateTypeSmallRoom, seed)
}

// NewTemplateDebugPlannerWithType は指定されたテンプレートタイプでプランナーを作成する
func NewTemplateDebugPlannerWithType(templateType TemplateType, seed uint64) (*PlannerChain, error) {
	// テンプレートローダーを作成
	templateLoader := maptemplate.NewTemplateLoader()

	// すべてのチャンクを事前登録
	if err := templateLoader.RegisterAllChunks([]string{
		"assets/levels/chunks",
		"assets/levels/facilities",
		"assets/levels/layouts",
	}); err != nil {
		return nil, fmt.Errorf("チャンク登録エラー: %w", err)
	}

	// すべてのパレットを事前登録
	if err := templateLoader.RegisterAllPalettes([]string{
		"assets/levels/palettes",
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
	return NewTemplatePlannerChain(template, palette, seed)
}
