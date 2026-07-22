package dungeon

import "github.com/kijimaD/ruins/internal/mapplanner"

// 全ステージ種別のマスタ定義
var (
	// DungeonDebug はデバッグ用ダンジョン定義
	DungeonDebug = &DungeonDefinition{
		name:        "デバッグ",
		totalFloors: 99,
		enemyTable:  "森",
		itemTable:   "森",
		baseTemp:    10,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
		},
	}

	// DungeonForest は森ダンジョン定義
	DungeonForest = &DungeonDefinition{
		name:        "亡者の森",
		description: "凍りついた森に、かつて猟師たちが分け入った。\n戻った者は少ない。冷気が骨まで届く。",
		imageKey:    "forest1",
		totalFloors: 20,
		enemyTable:  "森",
		itemTable:   "森",
		baseTemp:    0, // 寒い
		bossPlanner: &mapplanner.PlannerTypeBossFloor,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeForest, Weight: 5},
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 2},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 1},
		},
	}

	// DungeonCave は洞窟ダンジョン定義
	DungeonCave = &DungeonDefinition{
		name:        "灰の洞窟",
		description: "灰色の岩壁に凍晶が脈のように走っている。\n奥に進むほど、静かになる。",
		imageKey:    "cave1",
		totalFloors: 20,
		enemyTable:  "洞窟",
		itemTable:   "洞窟",
		baseTemp:    5, // 寒い
		bossPlanner: &mapplanner.PlannerTypeBossFloor,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeCave, Weight: 6},
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 2},
		},
	}

	// DungeonOverworld はオーバーワールド帯を表す種別。
	// フロアを作り直さず帯をスライドさせ続ける。ダンジョン専用フィールドを持たない別の型。
	// 帯形状 50x50 のチャンクを3枚並べる。この形状はマスタの設定で、RunSeed だけがプレイごとに変わる。
	DungeonOverworld = NewOverworldDefinition("オーバーワールド", 0, 50, 50, 3)

	// DungeonRuins は廃墟ダンジョン定義
	DungeonRuins = &DungeonDefinition{
		name:        "忘却の廃都",
		description: "古代の都市が、そのまま凍りついている。\n誰が何を忘れたのか、もう誰も知らない。",
		imageKey:    "city1",
		totalFloors: 20,
		enemyTable:  "廃墟",
		itemTable:   "廃墟",
		baseTemp:    15, // やや快適
		bossPlanner: &mapplanner.PlannerTypeBossFloor,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 4},
			{PlannerType: mapplanner.PlannerTypeRuins, Weight: 3},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 2},
		},
	}
)

// allDungeons は選択画面に表示する登録済みダンジョンの一覧
var allDungeons = []*DungeonDefinition{
	DungeonForest,
	DungeonCave,
	DungeonRuins,
}

// GetAllDungeons は選択画面に表示する全ダンジョン定義を返す。
// オーバーワールドやデバッグなどの内部用種別は含まない。
func GetAllDungeons() []*DungeonDefinition {
	return allDungeons
}

// GetAllDungeonNames は全ダンジョン名のスライスを返す
func GetAllDungeonNames() []string {
	names := make([]string, len(allDungeons))
	for i := range allDungeons {
		names[i] = allDungeons[i].Name()
	}
	return names
}

// internalDefinitions は選択画面に表示しない内部用の種別
var internalDefinitions = []StageDefinition{
	DungeonDebug,
	DungeonOverworld,
}

// GetStageDefinition は名前からステージ種別のマスタを取得する。
func GetStageDefinition(name string) (StageDefinition, bool) {
	// 内部用種別を先にチェックする
	for _, k := range internalDefinitions {
		if k.Name() == name {
			return k, true
		}
	}
	for _, d := range allDungeons {
		if d.Name() == name {
			return d, true
		}
	}
	return nil, false
}
