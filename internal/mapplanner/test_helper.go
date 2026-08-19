package mapplanner

import (
	"github.com/kijimaD/ruins/internal/oapi"
)

// CreateTestRawMaster はテスト用の oapi.Raws インスタンスを作成する
func CreateTestRawMaster() *oapi.Raws {
	// テスト用の基本的なタイルデータを定義
	testTiles := []oapi.Tile{
		{Id: "wall", Name: "wall", BlockPass: true},
		{Id: "floor", Name: "floor", BlockPass: false},
		{Id: "dirt", Name: "dirt", BlockPass: false},
		{Id: "void", Name: "void", BlockPass: true},
		{Id: "bridge_a", Name: "bridge_a", BlockPass: false},
		{Id: "bridge_b", Name: "bridge_b", BlockPass: false},
		{Id: "bridge_c", Name: "bridge_c", BlockPass: false},
		{Id: "bridge_d", Name: "bridge_d", BlockPass: false},
	}

	// テスト用のアイテムテーブルを定義
	testItemTables := []oapi.ItemTable{
		{
			Id:   "normal",
			Name: "normal",
			Entries: []oapi.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
				{Id: "ores", Weight: 0.5, MinDepth: 3, MaxDepth: 40},
			},
		},
		{
			Id:   "cave",
			Name: "cave",
			Entries: []oapi.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
				{Id: "ores", Weight: 0.6, MinDepth: 3, MaxDepth: 25},
			},
		},
		{
			Id:   "forest",
			Name: "forest",
			Entries: []oapi.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 15},
			},
		},
		{
			Id:   "ruins",
			Name: "ruins",
			Entries: []oapi.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 15},
				{Id: "ores", Weight: 0.8, MinDepth: 3, MaxDepth: 20},
			},
		},
	}

	// テスト用の敵テーブルを定義
	testEnemyTables := []oapi.EnemyTable{
		{
			Id:   "normal",
			Name: "normal",
			Entries: []oapi.EnemyTableEntry{
				{Id: "slime", Weight: 1.2, MinDepth: 1, MaxDepth: 10, Pack: "1d1"},
				{Id: "fireball", Weight: 1.0, MinDepth: 1, MaxDepth: 20, Pack: "1d1"},
				{Id: "light_tank", Weight: 0.8, MinDepth: 10, MaxDepth: 50, Pack: "1d1"},
			},
		},
		{
			Id:   "cave",
			Name: "cave",
			Entries: []oapi.EnemyTableEntry{
				{Id: "slime", Weight: 1.0, MinDepth: 1, MaxDepth: 8, Pack: "1d1"},
				{Id: "fireball", Weight: 1.0, MinDepth: 1, MaxDepth: 15, Pack: "1d1"},
				{Id: "light_tank", Weight: 0.6, MinDepth: 8, MaxDepth: 25, Pack: "1d1"},
			},
		},
		{
			Id:   "forest",
			Name: "forest",
			Entries: []oapi.EnemyTableEntry{
				{Id: "slime", Weight: 1.2, MinDepth: 1, MaxDepth: 12, Pack: "1d1"},
				{Id: "fireball", Weight: 1.0, MinDepth: 1, MaxDepth: 15, Pack: "1d1"},
				{Id: "light_tank", Weight: 0.5, MinDepth: 10, MaxDepth: 20, Pack: "1d1"},
			},
		},
		{
			Id:   "ruins",
			Name: "ruins",
			Entries: []oapi.EnemyTableEntry{
				{Id: "slime", Weight: 0.9, MinDepth: 1, MaxDepth: 10, Pack: "1d1"},
				{Id: "fireball", Weight: 0.8, MinDepth: 1, MaxDepth: 20, Pack: "1d1"},
				{Id: "light_tank", Weight: 1.0, MinDepth: 5, MaxDepth: 30, Pack: "1d1"},
				{Id: "ash_idol", Weight: 0.7, MinDepth: 15, MaxDepth: 35, Pack: "1d1"},
			},
		},
	}

	// テスト用のアイテムグループを定義
	testItemGroups := []oapi.ItemGroup{
		{
			Id:      "healing_items",
			Name:    "healing_items",
			Subtype: oapi.Distribution,
			Entries: []oapi.ItemGroupEntry{
				{Id: "healing_potion", Weight: 1.0, Pack: "1d3"},
				{Id: "antidote", Weight: 0.5, Pack: "1d1"},
			},
		},
		{
			Id:      "ores",
			Name:    "ores",
			Subtype: oapi.Collection,
			Entries: []oapi.ItemGroupEntry{
				{Id: "obsidian", Weight: 50, Pack: "1d2"},
				{Id: "silver_shard", Weight: 30, Pack: "1d1"},
			},
		},
	}

	// テスト用のアイテム定義（スタック判定に必要）

	return &oapi.Raws{
		Tiles: &testTiles,
		Items: &[]oapi.Item{
			{Id: "healing_potion", Name: "healing_potion", Description: "restores HP"},
			{Id: "antidote", Name: "antidote", Description: "cures poison"},
			{Id: "obsidian", Name: "obsidian", Description: "a black stone"},
			{Id: "silver_shard", Name: "silver_shard", Description: "silver_shard"},
			{Id: "herb", Name: "herb", Description: "herb"},
			{Id: "wooden_sword", Name: "wooden_sword", Description: "a wooden sword"},
		},
		ItemGroups:  &testItemGroups,
		ItemTables:  &testItemTables,
		EnemyTables: &testEnemyTables,
	}
}
