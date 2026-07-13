package dungeon

import "github.com/kijimaD/ruins/internal/mapplanner"

// 全ダンジョン定義
var (
	// DungeonTown は拠点用ダンジョン定義
	DungeonTown = Definition{
		Name:            "晶営地",
		TotalFloors:     1,
		EnemyTableName:  "",
		ItemTableName:   "",
		BaseTemperature: 0, // デバッグ用
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeTown, Weight: 1},
		},
	}

	// DungeonMarket はマクロ移動の集落（マーケット）ノード用の定義。商人で交易し帰還ゲートで道中へ戻る
	DungeonMarket = Definition{
		Name:            "集落",
		TotalFloors:     1,
		EnemyTableName:  "",
		ItemTableName:   "",
		BaseTemperature: 0,
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeMarket, Weight: 1},
		},
	}

	// DungeonDebug はデバッグ用ダンジョン定義
	DungeonDebug = Definition{
		Name:            "デバッグ",
		TotalFloors:     99,
		EnemyTableName:  "森",
		ItemTableName:   "森",
		BaseTemperature: 10,
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
		},
	}

	// DungeonForest は森ダンジョン定義
	DungeonForest = Definition{
		Name:            "亡者の森",
		Description:     "凍りついた森に、かつて猟師たちが分け入った。\n戻った者は少ない。冷気が骨まで届く。",
		ImageKey:        "forest1",
		TotalFloors:     20,
		EnemyTableName:  "森",
		ItemTableName:   "森",
		BaseTemperature: 0, // 寒い
		BossPlannerType: &mapplanner.PlannerTypeBossFloor,
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeForest, Weight: 5},
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 2},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 1},
		},
	}

	// DungeonCave は洞窟ダンジョン定義
	DungeonCave = Definition{
		Name:            "灰の洞窟",
		Description:     "灰色の岩壁に凍晶が脈のように走っている。\n奥に進むほど、静かになる。",
		ImageKey:        "cave1",
		TotalFloors:     20,
		EnemyTableName:  "洞窟",
		ItemTableName:   "洞窟",
		BaseTemperature: 5, // 寒い
		BossPlannerType: &mapplanner.PlannerTypeBossFloor,
		PlannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeCave, Weight: 6},
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 2},
		},
	}

	// DungeonRuins は廃墟ダンジョン定義
	DungeonRuins = Definition{
		Name:            "忘却の廃都",
		Description:     "古代の都市が、そのまま凍りついている。\n誰が何を忘れたのか、もう誰も知らない。",
		ImageKey:        "city1",
		TotalFloors:     20,
		EnemyTableName:  "廃墟",
		ItemTableName:   "廃墟",
		BaseTemperature: 15, // やや快適
		BossPlannerType: &mapplanner.PlannerTypeBossFloor,
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

// GetAllDungeonNames は全ダンジョン名のスライスを返す
func GetAllDungeonNames() []string {
	names := make([]string, len(allDungeons))
	for i := range allDungeons {
		names[i] = allDungeons[i].Name
	}
	return names
}

// internalDungeons は選択画面に表示しない内部用ダンジョン定義
var internalDungeons = []Definition{
	DungeonTown,
	DungeonMarket,
	DungeonDebug,
}

// GetDungeon は名前からダンジョン定義を取得する
func GetDungeon(name string) (Definition, bool) {
	// 内部用定義を先にチェック
	for i := range internalDungeons {
		if internalDungeons[i].Name == name {
			return internalDungeons[i], true
		}
	}
	for i := range allDungeons {
		if allDungeons[i].Name == name {
			return allDungeons[i], true
		}
	}
	return Definition{}, false
}
