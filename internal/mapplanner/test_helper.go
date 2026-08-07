package mapplanner

import (
	"github.com/kijimaD/ruins/internal/oapi"
)

// CreateTestRawMaster はテスト用の oapi.Raws インスタンスを作成する
func CreateTestRawMaster() *oapi.Raws {
	// テスト用の基本的なタイルデータを定義
	testTiles := []oapi.Tile{
		{Name: "wall", BlockPass: true},
		{Name: "floor", BlockPass: false},
		{Name: "dirt", BlockPass: false},
		{Name: "void", BlockPass: true},
		{Name: "bridge_a", BlockPass: false},
		{Name: "bridge_b", BlockPass: false},
		{Name: "bridge_c", BlockPass: false},
		{Name: "bridge_d", BlockPass: false},
	}

	// テスト用のアイテムテーブルを定義
	testItemTables := []oapi.ItemTable{
		{
			Id:   "normal",
			Name: "normal",
			Entries: []oapi.ItemTableEntry{
				{GroupName: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
				{GroupName: "ores", Weight: 0.5, MinDepth: 3, MaxDepth: 40},
			},
		},
		{
			Id:   "cave",
			Name: "cave",
			Entries: []oapi.ItemTableEntry{
				{GroupName: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 20},
				{GroupName: "ores", Weight: 0.6, MinDepth: 3, MaxDepth: 25},
			},
		},
		{
			Id:   "forest",
			Name: "forest",
			Entries: []oapi.ItemTableEntry{
				{GroupName: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 15},
			},
		},
		{
			Id:   "ruins",
			Name: "ruins",
			Entries: []oapi.ItemTableEntry{
				{GroupName: "healing_items", Weight: 1.0, MinDepth: 1, MaxDepth: 15},
				{GroupName: "ores", Weight: 0.8, MinDepth: 3, MaxDepth: 20},
			},
		},
	}

	// テスト用の敵テーブルを定義
	testEnemyTables := []oapi.EnemyTable{
		{
			Id:   "normal",
			Name: "normal",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: "slime", Weight: 1.2, MinDepth: 1, MaxDepth: 10, Pack: "1d1"},
				{EnemyName: "fireball", Weight: 1.0, MinDepth: 1, MaxDepth: 20, Pack: "1d1"},
				{EnemyName: "light_tank", Weight: 0.8, MinDepth: 10, MaxDepth: 50, Pack: "1d1"},
			},
		},
		{
			Id:   "cave",
			Name: "cave",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: "slime", Weight: 1.0, MinDepth: 1, MaxDepth: 8, Pack: "1d1"},
				{EnemyName: "fireball", Weight: 1.0, MinDepth: 1, MaxDepth: 15, Pack: "1d1"},
				{EnemyName: "light_tank", Weight: 0.6, MinDepth: 8, MaxDepth: 25, Pack: "1d1"},
			},
		},
		{
			Id:   "forest",
			Name: "forest",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: "slime", Weight: 1.2, MinDepth: 1, MaxDepth: 12, Pack: "1d1"},
				{EnemyName: "fireball", Weight: 1.0, MinDepth: 1, MaxDepth: 15, Pack: "1d1"},
				{EnemyName: "light_tank", Weight: 0.5, MinDepth: 10, MaxDepth: 20, Pack: "1d1"},
			},
		},
		{
			Id:   "ruins",
			Name: "ruins",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: "slime", Weight: 0.9, MinDepth: 1, MaxDepth: 10, Pack: "1d1"},
				{EnemyName: "fireball", Weight: 0.8, MinDepth: 1, MaxDepth: 20, Pack: "1d1"},
				{EnemyName: "light_tank", Weight: 1.0, MinDepth: 5, MaxDepth: 30, Pack: "1d1"},
				{EnemyName: "ash_idol", Weight: 0.7, MinDepth: 15, MaxDepth: 35, Pack: "1d1"},
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
				{ItemName: "healing_potion", Weight: 1.0, Pack: "1d3"},
				{ItemName: "antidote", Weight: 0.5, Pack: "1d1"},
			},
		},
		{
			Id:      "ores",
			Name:    "ores",
			Subtype: oapi.Collection,
			Entries: []oapi.ItemGroupEntry{
				{ItemName: "obsidian", Weight: 50, Pack: "1d2"},
				{ItemName: "silver_shard", Weight: 30, Pack: "1d1"},
			},
		},
	}

	// テスト用のアイテム定義（Stackable判定に必要）
	stackableTrue := true

	return &oapi.Raws{
		Tiles: &testTiles,
		Items: &[]oapi.Item{
			{Id: "healing_potion", Name: "healing_potion", Description: "restores HP", Stackable: &stackableTrue},
			{Id: "antidote", Name: "antidote", Description: "cures poison", Stackable: &stackableTrue},
			{Id: "obsidian", Name: "obsidian", Description: "a black stone", Stackable: &stackableTrue},
			{Id: "silver_shard", Name: "silver_shard", Description: "silver_shard", Stackable: &stackableTrue},
			{Id: "herb", Name: "herb", Description: "herb", Stackable: &stackableTrue},
			{Id: "wooden_sword", Name: "wooden_sword", Description: "a wooden sword"},
		},
		ItemGroups:  &testItemGroups,
		ItemTables:  &testItemTables,
		EnemyTables: &testEnemyTables,
	}
}
