package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// resolveEnemyEntries は敵テーブル名とRawMasterから、指定深度でフィルタリングしたSpawnEntryを返す
func resolveEnemyEntries(rawMaster *oapi.Raws, tableName string, depth int) ([]SpawnEntry, error) {
	if rawMaster == nil || tableName == "" {
		return nil, nil
	}
	enemyTable, err := raw.GetEnemyTable(*rawMaster, tableName)
	if err != nil {
		return nil, fmt.Errorf("enemy table not found: %s: %w", tableName, err)
	}
	result := make([]SpawnEntry, 0, len(enemyTable.Entries))
	for _, entry := range enemyTable.Entries {
		if int32(depth) < entry.MinDepth || int32(depth) > entry.MaxDepth {
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

// resolveItemSources はアイテムテーブル名と RawMaster から、指定深度でフィルタリングした参照先グループを返す。
// グループ中身の解決と抽選は draw 時に raw.SelectFromItemGroup が担うので、ここはテーブルの深度フィルタと
// 参照先グループの収集だけを行う。テーブルから group への参照の実在は raw のロード時検証が担保する。
func resolveItemSources(rawMaster *oapi.Raws, tableName string, depth int) ([]itemGroupRef, error) {
	if rawMaster == nil || tableName == "" {
		return nil, nil
	}
	itemTable, err := raw.GetItemTable(*rawMaster, tableName)
	if err != nil {
		return nil, fmt.Errorf("item table not found: %s: %w", tableName, err)
	}
	result := make([]itemGroupRef, 0, len(itemTable.Entries))
	for _, entry := range itemTable.Entries {
		if int32(depth) < entry.MinDepth || int32(depth) > entry.MaxDepth {
			continue
		}
		result = append(result, itemGroupRef{GroupID: entry.Id, Weight: entry.Weight})
	}
	return result, nil
}
