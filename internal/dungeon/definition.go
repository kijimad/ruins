package dungeon

import "github.com/kijimaD/ruins/internal/mapplanner"

// Definition はダンジョンの設定を保持する
type Definition struct {
	Name            string                  // ダンジョン名
	Description     string                  // ダンジョン説明文
	ImageKey        string                  // 背景画像のスプライトキー
	TotalFloors     int                     // 総階層数
	EnemyTableName  string                  // 敵テーブル名
	ItemTableName   string                  // アイテムテーブル名
	PlannerPool     []PlannerWeight         // 使用するマップ種類と重み
	BaseTemperature int                     // 基本気温（摂氏）
	BossPlannerType *mapplanner.PlannerType // 最終階層で使用するボスフロアプランナー。nilの場合はボスフロアなし
}

// PlannerWeight はマップ種類と出現重みのペアを表す
type PlannerWeight struct {
	PlannerType mapplanner.PlannerType
	Weight      float64
}
