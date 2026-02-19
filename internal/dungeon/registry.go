package dungeon

import "github.com/kijimaD/ruins/internal/mapplanner"

// 全ダンジョン定義
var (
	// DungeonTown は街用ダンジョン定義
	DungeonTown = Definition{
		Name:           "街",
		TotalFloors:    1,
		EnemyTableName: "",
		ItemTableName:  "",
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeTown, Weight: 1},
		},
	}

	// DungeonDebug はデバッグ用ダンジョン定義
	DungeonDebug = Definition{
		Name:           "デバッグ",
		TotalFloors:    99,
		EnemyTableName: "森",
		ItemTableName:  "森",
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
		},
	}

	DungeonForest = Definition{
		Name:           "亡者の森",
		TotalFloors:    10,
		EnemyTableName: "森",
		ItemTableName:  "森",
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeForest, Weight: 5},
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 2},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 1},
		},
	}

	DungeonCave = Definition{
		Name:           "灰の洞窟",
		TotalFloors:    15,
		EnemyTableName: "洞窟",
		ItemTableName:  "洞窟",
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeCave, Weight: 6},
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 2},
		},
	}

	DungeonRuins = Definition{
		Name:           "忘却の廃都",
		TotalFloors:    20,
		EnemyTableName: "廃墟",
		ItemTableName:  "廃墟",
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 4},
			{PlannerType: mapplanner.PlannerTypeRuins, Weight: 3},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 2},
		},
	}
)

// allDungeons は登録済みダンジョンの一覧
var allDungeons = []Definition{
	DungeonForest,
	DungeonCave,
	DungeonRuins,
}

// GetAllDungeons は全ダンジョン定義を返す
func GetAllDungeons() []Definition {
	return allDungeons
}

// internalDungeons は選択画面に表示しない内部用ダンジョン定義
var internalDungeons = []Definition{
	DungeonTown,
	DungeonDebug,
}

// GetDungeon は名前からダンジョン定義を取得する
func GetDungeon(name string) (Definition, bool) {
	// 内部用定義を先にチェック
	for _, d := range internalDungeons {
		if d.Name == name {
			return d, true
		}
	}
	for _, d := range allDungeons {
		if d.Name == name {
			return d, true
		}
	}
	return Definition{}, false
}
