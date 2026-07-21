package mapplanner

import (
	"github.com/kijimaD/ruins/internal/oapi"
)

// テストフィクスチャで相互参照する名前。テーブルとグループ、アイテム定義で一致させる必要があり、
// 定数化することでタイポによる参照ずれをコンパイル時に防ぐ。
const (
	fixGroupHealing  = "回復アイテム"
	fixGroupOre      = "鉱石類"
	fixEnemySlime    = "スライム"
	fixEnemyFireball = "火の玉"
	fixEnemyTank     = "軽戦車"
	fixItemSilver    = "銀の欠片"
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
			Name: "通常",
			Entries: []oapi.ItemTableEntry{
				{GroupName: fixGroupHealing, Weight: 1.0, MinDepth: 1, MaxDepth: 20},
				{GroupName: fixGroupOre, Weight: 0.5, MinDepth: 3, MaxDepth: 40},
			},
		},
		{
			Name: "洞窟",
			Entries: []oapi.ItemTableEntry{
				{GroupName: fixGroupHealing, Weight: 1.0, MinDepth: 1, MaxDepth: 20},
				{GroupName: fixGroupOre, Weight: 0.6, MinDepth: 3, MaxDepth: 25},
			},
		},
		{
			Name: "森",
			Entries: []oapi.ItemTableEntry{
				{GroupName: fixGroupHealing, Weight: 1.0, MinDepth: 1, MaxDepth: 15},
			},
		},
		{
			Name: "廃墟",
			Entries: []oapi.ItemTableEntry{
				{GroupName: fixGroupHealing, Weight: 1.0, MinDepth: 1, MaxDepth: 15},
				{GroupName: fixGroupOre, Weight: 0.8, MinDepth: 3, MaxDepth: 20},
			},
		},
	}

	// テスト用の敵テーブルを定義
	testEnemyTables := []oapi.EnemyTable{
		{
			Name: "通常",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: fixEnemySlime, Weight: 1.2, MinDepth: 1, MaxDepth: 10},
				{EnemyName: fixEnemyFireball, Weight: 1.0, MinDepth: 1, MaxDepth: 20},
				{EnemyName: fixEnemyTank, Weight: 0.8, MinDepth: 10, MaxDepth: 50},
			},
		},
		{
			Name: "洞窟",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: fixEnemySlime, Weight: 1.0, MinDepth: 1, MaxDepth: 8},
				{EnemyName: fixEnemyFireball, Weight: 1.0, MinDepth: 1, MaxDepth: 15},
				{EnemyName: fixEnemyTank, Weight: 0.6, MinDepth: 8, MaxDepth: 25},
			},
		},
		{
			Name: "森",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: fixEnemySlime, Weight: 1.2, MinDepth: 1, MaxDepth: 12},
				{EnemyName: fixEnemyFireball, Weight: 1.0, MinDepth: 1, MaxDepth: 15},
				{EnemyName: fixEnemyTank, Weight: 0.5, MinDepth: 10, MaxDepth: 20},
			},
		},
		{
			Name: "廃墟",
			Entries: []oapi.EnemyTableEntry{
				{EnemyName: fixEnemySlime, Weight: 0.9, MinDepth: 1, MaxDepth: 10},
				{EnemyName: fixEnemyFireball, Weight: 0.8, MinDepth: 1, MaxDepth: 20},
				{EnemyName: fixEnemyTank, Weight: 1.0, MinDepth: 5, MaxDepth: 30},
				{EnemyName: "灰の偶像", Weight: 0.7, MinDepth: 15, MaxDepth: 35},
			},
		},
	}

	// テスト用のアイテムグループを定義
	testItemGroups := []oapi.ItemGroup{
		{
			Name:    fixGroupHealing,
			Subtype: oapi.Distribution,
			Entries: []oapi.ItemGroupEntry{
				{ItemName: "回復薬", Weight: 1.0, PackMin: 1, PackMax: 3},
				{ItemName: "毒消し", Weight: 0.5, PackMin: 1, PackMax: 1},
			},
		},
		{
			Name:    fixGroupOre,
			Subtype: oapi.Collection,
			Entries: []oapi.ItemGroupEntry{
				{ItemName: "黒曜石", Weight: 50, PackMin: 1, PackMax: 2},
				{ItemName: fixItemSilver, Weight: 30, PackMin: 1, PackMax: 1},
			},
		},
	}

	// テスト用のアイテム定義（Stackable判定に必要）
	stackableTrue := true
	testItems := []oapi.Item{
		{Name: "回復薬", Description: "HPを回復する", Stackable: &stackableTrue},
		{Name: "毒消し", Description: "毒を回復する", Stackable: &stackableTrue},
		{Name: "黒曜石", Description: "黒い石", Stackable: &stackableTrue},
		{Name: fixItemSilver, Description: fixItemSilver, Stackable: &stackableTrue},
		{Name: "薬草", Description: "薬草", Stackable: &stackableTrue},
		{Name: "木刀", Description: "木製の刀"},
	}

	return &oapi.Raws{
		Tiles:       &testTiles,
		Items:       &testItems,
		ItemGroups:  &testItemGroups,
		ItemTables:  &testItemTables,
		EnemyTables: &testEnemyTables,
	}
}
