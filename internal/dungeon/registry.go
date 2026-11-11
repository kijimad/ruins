package dungeon

import (
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// 内部用のステージのマスタ定義。遺跡の入口が指すダンジョンは raw の dungeons の行が持つ
var (
	// DungeonDebug はデバッグ用ダンジョン定義
	DungeonDebug = &DungeonDefinition{
		name:        "Debug",
		displayName: "Debug",
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
		name:        "Debug town",
		displayName: "Debug town",
		enemyTable:  "",
		itemTable:   "",
		baseTemp:    20,
		plannerPool: []PlannerWeight{
			{PlannerType: mapplanner.PlannerTypeDebugTown, Weight: 1},
		},
	}

	// DungeonOverworld はオーバーワールド帯を表す定義。
	// フロアを作り直さず帯をスライドさせ続ける。ダンジョン専用フィールドを持たない別の型。
	// 帯形状 24x24 のチャンクを横7枚・縦9枚並べる。1チャンク=1建物=地図の1マスの縮尺。
	// 1建物=24タイル四方を基準にし、建物・部屋を1棟まるごと歩けるサイズまで広げる。
	// 縦9レーンで北/中央/南のルート選択が生まれる。
	// この形状はマスタの設定で、RunSeed だけがプレイごとに変わる。
	DungeonOverworld = NewOverworldDefinition("Overworld", 0, 24, 24, 7, 9)
)

// plannerTypes は raw のダンジョンの間取りの種類が指すプランナー
var plannerTypes = map[rawschema.DungeonPlanner]mapplanner.PlannerType{
	rawschema.DungeonPlannerSmallRoom: mapplanner.PlannerTypeSmallRoom,
	rawschema.DungeonPlannerBigRoom:   mapplanner.PlannerTypeBigRoom,
	rawschema.DungeonPlannerCave:      mapplanner.PlannerTypeCave,
	rawschema.DungeonPlannerRuins:     mapplanner.PlannerTypeRuins,
	rawschema.DungeonPlannerForest:    mapplanner.PlannerTypeForest,
}

// GetAllDungeons は raw の dungeons の行から、遺跡の入口が指す全ダンジョン定義を行の順に返す。
// オーバーワールドやデバッグなどの内部用の定義は含まない。
func GetAllDungeons(raws rawschema.Raws) []*DungeonDefinition {
	rows := raw.PtrSlice(raws.Dungeons)
	defs := make([]*DungeonDefinition, 0, len(rows))
	for i := range rows {
		defs = append(defs, newDungeonDefinition(&rows[i]))
	}
	return defs
}

// newDungeonDefinition は raw の dungeons の1行からダンジョン定義を組み立てる
func newDungeonDefinition(row *rawschema.Dungeon) *DungeonDefinition {
	pool := make([]PlannerWeight, 0, len(row.Planners))
	for _, p := range row.Planners {
		pool = append(pool, PlannerWeight{PlannerType: plannerTypes[p.Id], Weight: p.Weight})
	}
	return &DungeonDefinition{
		name:        row.Id,
		displayName: i18n.Msgid(row.Name),
		enemyTable:  row.EnemyTable,
		itemTable:   row.ItemTable,
		baseTemp:    row.BaseTemp,
		plannerPool: pool,
	}
}

// internalDefinitions は選択画面に表示しない内部用の定義
var internalDefinitions = []StageDefinition{
	DungeonDebug,
	DungeonDebugTown,
	DungeonOverworld,
}

// BaseTemperatureFor は名前のステージ定義の基本気温を返す。未登録は0。
// ステージ生成時に StageField.BaseTemp へ写し、気温計算が dungeon 登録表を引かずに済むようにする
func BaseTemperatureFor(raws rawschema.Raws, name string) int {
	if def, ok := GetStageDefinition(raws, name); ok {
		return def.BaseTemperature()
	}
	return 0
}

// GetStageDefinition は名前からステージ定義のマスタを取得する。ダンジョンは raw の行の id で引く。
func GetStageDefinition(raws rawschema.Raws, name string) (StageDefinition, bool) {
	// 内部用の定義を先にチェックする
	for _, k := range internalDefinitions {
		if k.Name() == name {
			return k, true
		}
	}
	rows := raw.PtrSlice(raws.Dungeons)
	for i := range rows {
		if rows[i].Id == name {
			return newDungeonDefinition(&rows[i]), true
		}
	}
	return nil, false
}
