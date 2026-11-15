package mapplanner

import (
	"github.com/kijimaD/ruins/internal/rawschema"
)

// テスト用の間取りの行
var (
	testLayoutTiles = rawschema.LayoutTiles{Floor: "floor", Wall: "wall", Void: "void"}
	testRoomLayout  = rawschema.RoomLayout{
		Id: "small_room", Tiles: testLayoutTiles,
		Rooms: rawschema.RandInt{Base: 4, Spread: 10}, RoomSize: rawschema.RandInt{Base: 2, Spread: 8},
		CorridorWidth: 3, DoorChance: 0.8,
	}
	testBigRoomLayout = rawschema.BigRoomLayout{
		Id: "big_room", Tiles: testLayoutTiles,
		PillarSpacing: rawschema.RandInt{Base: 3, Spread: 4}, ObstacleArea: 30, MazeGapChance: 0.3,
		PlatformRadius: rawschema.RandInt{Base: 2, Spread: 3},
	}
	testCaveLayout = rawschema.CaveLayout{
		Id: "cave", Tiles: testLayoutTiles,
		WallChance: 0.3, Iterations: 3, WallNeighbors: 6, MinRoomTiles: 10, WidenChance: 0.3, StalactiteChance: 0.02,
	}
	testRuinsLayout = rawschema.RuinsLayout{
		Id: "ruins", Tiles: testLayoutTiles,
		Buildings: rawschema.RandInt{Base: 3, Spread: 3}, BuildingWidth: rawschema.RandInt{Base: 8, Spread: 10},
		BuildingHeight: rawschema.RandInt{Base: 6, Spread: 8}, WallGapChance: 0.3, PartitionGapChance: 0.4, PathChance: 0.3,
	}
	testForestLayout = rawschema.ForestLayout{
		Id: "forest", Tiles: rawschema.LayoutTiles{Floor: "dirt", Wall: "wall", Void: "dirt"}, Path: "road",
		Clearings: rawschema.RandInt{Base: 3, Spread: 4}, ClearingSize: rawschema.RandInt{Base: 4, Spread: 6},
		TreeDensity: 0.6, LargeTreeChance: 0.2, WildlifeSpots: rawschema.RandInt{Base: 2, Spread: 4},
	}
)

// CreateTestRawMaster はテスト用の rawschema.Raws インスタンスを作成する
func CreateTestRawMaster() *rawschema.Raws {
	// テスト用の基本的なタイルデータを定義
	autotile := &rawschema.TileSpawn{Autotile: true}
	testTiles := []rawschema.Tile{
		{Id: "wall", Name: "wall", BlockPass: true, Spawn: &rawschema.TileSpawn{Tile: new("dwall"), Autotile: true}},
		{Id: "floor", Name: "floor", BlockPass: false, Spawn: autotile},
		{Id: "road", Name: "road", BlockPass: false, Spawn: autotile},
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
		RoomLayouts:     &[]rawschema.RoomLayout{testRoomLayout},
		BigRoomLayouts:  &[]rawschema.BigRoomLayout{testBigRoomLayout},
		CaveLayouts:     &[]rawschema.CaveLayout{testCaveLayout},
		RuinsLayouts:    &[]rawschema.RuinsLayout{testRuinsLayout},
		ForestLayouts:   &[]rawschema.ForestLayout{testForestLayout},
		TemplateLayouts: &[]rawschema.TemplateLayout{{Id: "town_plaza", Template: "50x20_town_plaza"}, {Id: "debug_town", Template: "20x20_debug_town"}},
	}
}
