package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// resolveEnemyEntries は敵テーブル名とRawMasterから、指定危険度でフィルタリングしたSpawnEntryを返す
func resolveEnemyEntries(rawMaster *oapi.Raws, tableName string, danger int) ([]SpawnEntry, error) {
	// tableName 空はテーブル非設定のプランナー、rawMaster nil は Resources 未設定のワールド。
	// どちらも配置対象が無いだけの正常系なので、error でなく空を返す。呼び出し側は len 0 を no-op として扱う。
	if rawMaster == nil || tableName == "" {
		return nil, nil
	}
	enemyTable, err := raw.GetEnemyTable(*rawMaster, tableName)
	if err != nil {
		return nil, fmt.Errorf("enemy table not found: %s: %w", tableName, err)
	}
	result := make([]SpawnEntry, 0, len(enemyTable.Entries))
	for _, entry := range enemyTable.Entries {
		if danger < entry.MinDanger || danger > entry.MaxDanger {
			continue
		}
		pack, err := consts.ParseDice(entry.Pack)
		if err != nil {
			return nil, fmt.Errorf("enemy table '%s' entry '%s' has invalid pack notation: %w", tableName, entry.Id, err)
		}
		result = append(result, SpawnEntry{
			Name:   entry.Id,
			Weight: entry.Weight,
			Pack:   pack,
		})
	}
	return result, nil
}

// resolveItemSources はアイテムテーブル名と RawMaster から、指定危険度でフィルタリングした参照先グループを返す。
// グループ中身の解決と抽選は draw 時に raw.SelectFromItemGroup が担うので、ここはテーブルの危険度フィルタと
// 参照先グループの収集だけを行う。テーブルから group への参照の実在は raw のロード時検証が担保する。
func resolveItemSources(rawMaster *oapi.Raws, tableName string, danger int) ([]itemGroupRef, error) {
	// tableName 空はテーブル非設定のプランナー、rawMaster nil は Resources 未設定のワールド。
	// どちらも配置対象が無いだけの正常系なので、error でなく空を返す。呼び出し側は len 0 を no-op として扱う。
	if rawMaster == nil || tableName == "" {
		return nil, nil
	}
	itemTable, err := raw.GetItemTable(*rawMaster, tableName)
	if err != nil {
		return nil, fmt.Errorf("item table not found: %s: %w", tableName, err)
	}
	result := make([]itemGroupRef, 0, len(itemTable.Entries))
	for _, entry := range itemTable.Entries {
		if danger < entry.MinDanger || danger > entry.MaxDanger {
			continue
		}
		result = append(result, itemGroupRef{GroupID: entry.Id, Weight: entry.Weight})
	}
	return result, nil
}
