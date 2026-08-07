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
			return nil, fmt.Errorf("enemy table '%s' entry '%s' has invalid pack notation: %w", tableName, entry.EnemyName, err)
		}
		result = append(result, SpawnEntry{
			Name:   entry.EnemyName,
			Weight: entry.Weight,
			Pack:   pack,
		})
	}
	return result, nil
}

// resolveItemSources はアイテムテーブル名とRawMasterから、指定深度でフィルタリングしたItemSourceを返す
func resolveItemSources(rawMaster *oapi.Raws, tableName string, depth int) ([]ItemSource, error) {
	if rawMaster == nil || tableName == "" {
		return nil, nil
	}
	itemTable, err := raw.GetItemTable(*rawMaster, tableName)
	if err != nil {
		return nil, fmt.Errorf("item table not found: %s: %w", tableName, err)
	}
	result := make([]ItemSource, 0, len(itemTable.Entries))
	for _, entry := range itemTable.Entries {
		if int32(depth) < entry.MinDepth || int32(depth) > entry.MaxDepth {
			continue
		}
		group, err := raw.GetItemGroup(*rawMaster, entry.GroupName)
		if err != nil {
			return nil, fmt.Errorf("item group not found: %s: %w", entry.GroupName, err)
		}
		spawnEntries := make([]SpawnEntry, len(group.Entries))
		for i, ge := range group.Entries {
			pack, err := consts.ParseDice(ge.Pack)
			if err != nil {
				return nil, fmt.Errorf("item group '%s' entry '%s' has invalid pack notation: %w", entry.GroupName, ge.ItemName, err)
			}
			spawnEntries[i] = SpawnEntry{
				Name:   ge.ItemName,
				Weight: ge.Weight,
				Pack:   pack,
			}
		}
		result = append(result, ItemSource{
			Weight:  entry.Weight,
			Subtype: ItemGroupSubtype(group.Subtype),
			Entries: spawnEntries,
		})
	}
	return result, nil
}
