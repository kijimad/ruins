package consts

// PlannerTypeName はマップ生成タイプの名前を表す型
// mapplanner.PlannerTypeとの循環参照を避けるため、名前のみを持つ軽量な型
type PlannerTypeName string

const (
	// PlannerTypeNameRandom はランダム選択
	PlannerTypeNameRandom PlannerTypeName = "ランダム"
	// PlannerTypeNameSmallRoom は小部屋ダンジョン
	PlannerTypeNameSmallRoom PlannerTypeName = "小部屋"
	// PlannerTypeNameBigRoom は大部屋ダンジョン
	PlannerTypeNameBigRoom PlannerTypeName = "大部屋"
	// PlannerTypeNameCave は洞窟ダンジョン
	PlannerTypeNameCave PlannerTypeName = "洞窟"
	// PlannerTypeNameRuins は廃墟ダンジョン
	PlannerTypeNameRuins PlannerTypeName = "廃墟"
	// PlannerTypeNameForest は森ダンジョン
	PlannerTypeNameForest PlannerTypeName = "森"
	// PlannerTypeNameTown は市街地
	PlannerTypeNameTown PlannerTypeName = "市街地"
	// PlannerTypeNameOfficeBuilding は事務所ビル
	PlannerTypeNameOfficeBuilding PlannerTypeName = "事務所ビル"
	// PlannerTypeNameSmallTown は小さな町
	PlannerTypeNameSmallTown PlannerTypeName = "小さな町"
)
