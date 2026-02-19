package dungeon

import "github.com/kijimaD/ruins/internal/mapplanner"

// Definition はダンジョンの設定を保持する
type Definition struct {
	Name           string          // ダンジョン名
	TotalFloors    int             // 総階層数
	EnemyTableName string          // 敵テーブル名
	ItemTableName  string          // アイテムテーブル名
	PlannerPool    []PlannerWeight // 使用するマップ種類と重み
}

// PlannerWeight はマップ種類と出現重みのペアを表す
type PlannerWeight struct {
	PlannerType mapplanner.PlannerType
	Weight      int
}
