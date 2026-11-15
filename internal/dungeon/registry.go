package dungeon

import (
	"github.com/kijimaD/ruins/internal/i18n"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

// デバッグで名前を指して入るダンジョンの行の id
const (
	// DebugID は小部屋だけのデバッグ用ダンジョン
	DebugID = "debug"
	// DebugTownID は街用NPCと収納箱を絵で固定配置したデバッグ用ダンジョン
	DebugTownID = "debug_town"
)

// 内部用のステージのマスタ定義
var (
	// DungeonOverworld はオーバーワールド帯を表す定義。
	// フロアを作り直さず帯をスライドさせ続ける。ダンジョン専用フィールドを持たない別の型。
	// 帯形状 24x24 のチャンクを横7枚・縦9枚並べる。1チャンク=1建物=地図の1マスの縮尺。
	// 1建物=24タイル四方を基準にし、建物・部屋を1棟まるごと歩けるサイズまで広げる。
	// 縦9レーンで北/中央/南のルート選択が生まれる。
	// この形状はマスタの設定で、RunSeed だけがプレイごとに変わる。
	DungeonOverworld = NewOverworldDefinition("Overworld", 0, 24, 24, 7, 9)
)

// GetAllDungeons は raw の dungeons の行から、一覧に出す全ダンジョン定義を行の順に返す。
// オーバーワールドの定義と hidden の行は含まない。
func GetAllDungeons(raws rawschema.Raws) []*DungeonDefinition {
	rows := raw.PtrSlice(raws.Dungeons)
	defs := make([]*DungeonDefinition, 0, len(rows))
	for i := range rows {
		if rows[i].Hidden != nil && *rows[i].Hidden {
			continue
		}
		defs = append(defs, newDungeonDefinition(&rows[i]))
	}
	return defs
}

// newDungeonDefinition は raw の dungeons の1行からダンジョン定義を組み立てる
func newDungeonDefinition(row *rawschema.Dungeon) *DungeonDefinition {
	pool := make([]PlannerWeight, 0, len(row.Planners))
	for _, p := range row.Planners {
		pool = append(pool, PlannerWeight{PlannerType: mapplanner.LayoutPlannerType(p.Id), Weight: p.Weight})
	}
	return &DungeonDefinition{
		name:        row.Id,
		displayName: i18n.Msgid(row.Name),
		enemyTable:  row.EnemyTable,
		itemTable:   row.ItemTable,
		enemyCount:  row.EnemyCount,
		itemCount:   row.ItemCount,
		baseTemp:    row.BaseTemp,
		plannerPool: pool,
	}
}

// internalDefinitions は選択画面に表示しない内部用の定義
var internalDefinitions = []StageDefinition{
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
