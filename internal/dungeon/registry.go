package dungeon

import "github.com/kijimaD/ruins/internal/mapplanner"

// 全ステージのマスタ定義
var (
	// DungeonDebug はデバッグ用ダンジョン定義
	DungeonDebug = &DungeonDefinition{
		name:        "デバッグ",
		totalFloors: 99,
		enemyTable:  "forest",
		itemTable:   "forest",
		baseTemp:    10,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
		},
	}

	// DungeonDebugTown は街用NPCと収納箱をテンプレートで固定配置するデバッグ用定義。
	// 敵・アイテムテーブルを空にして、共通の敵配置プランナーを自然に空振りさせ、敵を湧かせない
	DungeonDebugTown = &DungeonDefinition{
		name:        "デバッグ街",
		totalFloors: 1,
		enemyTable:  "",
		itemTable:   "",
		baseTemp:    20,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeDebugTown, Weight: 1},
		},
	}

	// DungeonForest は森ダンジョン定義
	DungeonForest = &DungeonDefinition{
		name:        "亡者の森",
		description: "凍りついた森に、かつて猟師たちが分け入った。\n戻った者は少ない。冷気が骨まで届く。",
		imageKey:    "forest1",
		totalFloors: 20,
		enemyTable:  "forest",
		itemTable:   "forest",
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
		enemyTable:  "cave",
		itemTable:   "cave",
		baseTemp:    5, // 寒い
		bossPlanner: &mapplanner.PlannerTypeBossFloor,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeCave, Weight: 6},
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 1},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 2},
		},
	}

	// DungeonOverworld はオーバーワールド帯を表す定義。
	// フロアを作り直さず帯をスライドさせ続ける。ダンジョン専用フィールドを持たない別の型。
	// 帯形状 24x24 のチャンクを横7枚・縦9枚並べる。1チャンク=1建物=地図の1マスの縮尺。
	// 1建物=24タイル四方を基準にし、建物・部屋を1棟まるごと歩けるサイズまで広げる。
	// 縦9レーンで北/中央/南のルート選択が生まれる。
	// この形状はマスタの設定で、RunSeed だけがプレイごとに変わる。
	DungeonOverworld = NewOverworldDefinition("Overworld", 0, 24, 24, 7, 9)

	// DungeonRuins は廃墟ダンジョン定義
	DungeonRuins = &DungeonDefinition{
		name:        "忘却の廃都",
		description: "古代の都市が、そのまま凍りついている。\n誰が何を忘れたのか、もう誰も知らない。",
		imageKey:    "city1",
		totalFloors: 20,
		enemyTable:  "ruins_area",
		itemTable:   "ruins_area",
		baseTemp:    15, // やや快適
		bossPlanner: &mapplanner.PlannerTypeBossFloor,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeSmallRoom, Weight: 4},
			{PlannerType: mapplanner.PlannerTypeRuins, Weight: 3},
			{PlannerType: mapplanner.PlannerTypeBigRoom, Weight: 2},
		},
	}

	// DungeonCubeInterior は移動拠点キューブの内部の定義。選択画面にも地上の入口にも出さない
	// 内部用ステージなので internalDefinitions に置く。実体の生成は enterCube の SwapTo が
	// テンプレートから行い、この定義のフロア生成やテーブルは使わない。定義を持たせる目的は、
	// 他ステージと同じく名前で解決でき、復帰時のスプラッシュ表示に名前を使えるようにすること。
	// name は gc.NewCubeInteriorStage().Name と一致させる。ずれると復帰で定義解決に失敗する。
	DungeonCubeInterior = &DungeonDefinition{
		name:        "Cube interior",
		description: "移動拠点キューブの内部",
		totalFloors: 1,
		baseTemp:    15, // 内部はシェルター。要調整
		// 内部のレイアウトはこの planner のテンプレート。実際の生成は enterCube の SwapTo が
		// 同じテンプレートで行うので、この pool は spawnFloor 経由では引かれない。他の定義と形を
		// そろえ、レイアウトの出所を示すために持つ
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeCubeInterior, Weight: 1},
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
// オーバーワールドやデバッグなどの内部用の定義は含まない。
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

// internalDefinitions は選択画面に表示しない内部用の定義
var internalDefinitions = []StageDefinition{
	DungeonDebug,
	DungeonDebugTown,
	DungeonOverworld,
	DungeonCubeInterior,
}

// GetStageDefinition は名前からステージ定義のマスタを取得する。
func GetStageDefinition(name string) (StageDefinition, bool) {
	// 内部用の定義を先にチェックする
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
