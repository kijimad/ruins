package mapplanner

import (
	"github.com/kijimaD/ruins/internal/rawschema"
)

// CreateTestRawMaster はテスト用の rawschema.Raws インスタンスを作成する
func CreateTestRawMaster() *rawschema.Raws {
	// テスト用の基本的なタイルデータを定義
	autotile := &rawschema.TileSpawn{Autotile: true}
	testTiles := []rawschema.Tile{
		{Id: "wall", Name: "wall", BlockPass: true, Spawn: &rawschema.TileSpawn{Tile: new("dwall"), Autotile: true}},
		{Id: "floor", Name: "floor", BlockPass: false, Spawn: autotile},
		{Id: "dirt", Name: "dirt", BlockPass: false, Spawn: autotile},
		{Id: "void", Name: "void", BlockPass: true, Spawn: &rawschema.TileSpawn{}},
		{Id: "bridge_a", Name: "bridge_a", BlockPass: false, Spawn: autotile},
		{Id: "bridge_b", Name: "bridge_b", BlockPass: false, Spawn: autotile},
		{Id: "bridge_c", Name: "bridge_c", BlockPass: false, Spawn: autotile},
		{Id: "bridge_d", Name: "bridge_d", BlockPass: false, Spawn: autotile},
	}

	// テスト用のアイテムテーブルを定義
	testItemTables := []rawschema.ItemTable{
		{
			Id:   "normal",
			Name: "normal",
			Entries: []rawschema.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
				{Id: "ores", Weight: 0.5, MinDanger: 3, MaxDanger: 40},
			},
		},
		{
			Id:   "cave",
			Name: "cave",
			Entries: []rawschema.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDanger: 1, MaxDanger: 20},
				{Id: "ores", Weight: 0.6, MinDanger: 3, MaxDanger: 25},
			},
		},
		{
			Id:   "forest",
			Name: "forest",
			Entries: []rawschema.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDanger: 1, MaxDanger: 15},
			},
		},
		{
			Id:   "ruins",
			Name: "ruins",
			Entries: []rawschema.ItemTableEntry{
				{Id: "healing_items", Weight: 1.0, MinDanger: 1, MaxDanger: 15},
				{Id: "ores", Weight: 0.8, MinDanger: 3, MaxDanger: 20},
			},
		},
	}

	// テスト用の敵テーブルを定義
	testEnemyTables := []rawschema.EnemyTable{
		{
			Id:   "normal",
			Name: "normal",
			Entries: []rawschema.EnemyTableEntry{
				{Id: "slime", Weight: 1.2, MinDanger: 1, MaxDanger: 10, Pack: "1d1"},
				{Id: "fireball", Weight: 1.0, MinDanger: 1, MaxDanger: 20, Pack: "1d1"},
				{Id: "light_tank", Weight: 0.8, MinDanger: 10, MaxDanger: 50, Pack: "1d1"},
			},
		},
		{
			Id:   "cave",
			Name: "cave",
			Entries: []rawschema.EnemyTableEntry{
				{Id: "slime", Weight: 1.0, MinDanger: 1, MaxDanger: 8, Pack: "1d1"},
				{Id: "fireball", Weight: 1.0, MinDanger: 1, MaxDanger: 15, Pack: "1d1"},
				{Id: "light_tank", Weight: 0.6, MinDanger: 8, MaxDanger: 25, Pack: "1d1"},
			},
		},
		{
			Id:   "forest",
			Name: "forest",
			Entries: []rawschema.EnemyTableEntry{
				{Id: "slime", Weight: 1.2, MinDanger: 1, MaxDanger: 12, Pack: "1d1"},
				{Id: "fireball", Weight: 1.0, MinDanger: 1, MaxDanger: 15, Pack: "1d1"},
				{Id: "light_tank", Weight: 0.5, MinDanger: 10, MaxDanger: 20, Pack: "1d1"},
			},
		},
		{
			Id:   "ruins",
			Name: "ruins",
			Entries: []rawschema.EnemyTableEntry{
				{Id: "slime", Weight: 0.9, MinDanger: 1, MaxDanger: 10, Pack: "1d1"},
				{Id: "fireball", Weight: 0.8, MinDanger: 1, MaxDanger: 20, Pack: "1d1"},
				{Id: "light_tank", Weight: 1.0, MinDanger: 5, MaxDanger: 30, Pack: "1d1"},
				{Id: "ash_idol", Weight: 0.7, MinDanger: 15, MaxDanger: 35, Pack: "1d1"},
			},
		},
	}

	// テスト用のアイテムグループを定義
	testItemGroups := []rawschema.ItemGroup{
		{
			Id:   "healing_items",
			Name: "healing_items",
			Entries: []rawschema.ItemGroupEntry{
				{Id: "healing_potion", Weight: 1.0, Pack: "1d3"},
				{Id: "antidote", Weight: 0.5, Pack: "1d1"},
			},
		},
	}
	testItemCollections := []rawschema.ItemCollection{
		{
			Id:   "ores",
			Name: "ores",
			Entries: []rawschema.ItemCollectionEntry{
				{Id: "obsidian", Chance: 50, Pack: "1d2"},
				{Id: "silver_shard", Chance: 30, Pack: "1d1"},
			},
		},
	}

	// テスト用のアイテム定義（スタック判定に必要）

	return &rawschema.Raws{
		Tiles: &testTiles,
		Items: &[]rawschema.Item{
			{Id: "healing_potion", Name: "healing_potion", Description: "restores HP"},
			{Id: "antidote", Name: "antidote", Description: "cures poison"},
			{Id: "obsidian", Name: "obsidian", Description: "a black stone"},
			{Id: "silver_shard", Name: "silver_shard", Description: "silver_shard"},
			{Id: "herb", Name: "herb", Description: "herb"},
			{Id: "wooden_sword", Name: "wooden_sword", Description: "a wooden sword"},
		},
		ItemGroups:      &testItemGroups,
		ItemCollections: &testItemCollections,
		ItemTables:      &testItemTables,
		EnemyTables:     &testEnemyTables,
	}
}
