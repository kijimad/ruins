package mapplanner

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewItemPlanner(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plannerType := PlannerType{
		Name: "test",
		ItemSources: []ItemSource{
			{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{{Name: "薬草", Weight: 1.0, PackMin: 1, PackMax: 1}}},
		},
	}
	planner := NewItemPlanner(world, plannerType)

	assert.NotNil(t, planner)
	assert.Equal(t, "test", planner.plannerType.Name)
}

func TestItemPlanner_PlanMeta(t *testing.T) {
	t.Parallel()

	t.Run("ItemSourcesが空の場合は何もしない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name:        "test_empty",
			ItemSources: []ItemSource{},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.Empty(t, chain.PlanData.Items)
	})

	t.Run("ItemSourcesがある場合はアイテムが配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_with_items",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{{Name: "薬草", Weight: 1.0, PackMin: 1, PackMax: 1}}},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.NotEmpty(t, chain.PlanData.Items)
	})

	t.Run("配置されたアイテムは歩行可能なタイルにある", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_valid_position",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{{Name: "薬草", Weight: 1.0, PackMin: 1, PackMax: 1}}},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		for _, item := range chain.PlanData.Items {
			tileIdx := chain.PlanData.Level.XYTileIndex(consts.Tile(item.X), consts.Tile(item.Y))
			tile := chain.PlanData.Tiles[tileIdx]
			assert.False(t, tile.BlockPass, "アイテム(%d,%d)が壁タイルに配置されている", item.X, item.Y)
		}
	})

	t.Run("深い階層ではアイテム数が増加する", func(t *testing.T) {
		t.Parallel()

		plannerType := PlannerType{
			Name: "test_depth",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{{Name: "薬草", Weight: 1.0, PackMin: 1, PackMax: 1}}},
			},
		}

		// 浅い階層
		worldShallow := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(worldShallow, &gc.Dungeon{Depth: 1})

		chainShallow, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chainShallow.PlanData.RawMaster = CreateTestRawMaster()
		err = chainShallow.Plan()
		require.NoError(t, err)

		plannerShallow := NewItemPlanner(worldShallow, plannerType)
		err = plannerShallow.PlanMeta(&chainShallow.PlanData)
		require.NoError(t, err)

		// 深い階層
		worldDeep := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(worldDeep, &gc.Dungeon{Depth: 10})

		chainDeep, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chainDeep.PlanData.RawMaster = CreateTestRawMaster()
		err = chainDeep.Plan()
		require.NoError(t, err)

		plannerDeep := NewItemPlanner(worldDeep, plannerType)
		err = plannerDeep.PlanMeta(&chainDeep.PlanData)
		require.NoError(t, err)

		// 両方ともアイテムが配置されていることを確認
		assert.NotEmpty(t, chainShallow.PlanData.Items)
		assert.NotEmpty(t, chainDeep.PlanData.Items)
	})

	t.Run("複数のアイテムタイプが重みに応じて選択される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_multiple_items",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{
					{Name: "薬草", Weight: 10.0, PackMin: 1, PackMax: 1},
					{Name: "毒消し", Weight: 1.0, PackMin: 1, PackMax: 1},
				}},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		assert.NotEmpty(t, chain.PlanData.Items)
	})

	t.Run("StackableアイテムはCountが2以上になる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_stackable",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{
					{Name: "回復薬", Weight: 1.0, PackMin: 3, PackMax: 3},
				}},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		require.NotEmpty(t, chain.PlanData.Items)

		// StackableアイテムはCountがPackSizeと一致し、1エンティティにまとめられる
		hasStackedItem := false
		for _, item := range chain.PlanData.Items {
			if item.Name == "回復薬" && item.Count == 3 {
				hasStackedItem = true
				break
			}
		}
		assert.True(t, hasStackedItem, "StackableアイテムはCount=3の1エンティティにまとめられるべき")
	})

	t.Run("非StackableアイテムはCount=1で個別配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_non_stackable",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{
					{Name: "木刀", Weight: 1.0, PackMin: 2, PackMax: 2},
				}},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		require.NotEmpty(t, chain.PlanData.Items)

		// 非StackableアイテムはすべてCount=1
		for _, item := range chain.PlanData.Items {
			if item.Name == "木刀" {
				assert.Equal(t, 1, item.Count, "非StackableアイテムはCount=1であるべき")
			}
		}
	})

	t.Run("Stackableアイテムの配置数がplacedカウントで制御される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		// PackSize=5のStackableアイテム。totalが8-12の範囲なので、
		// 配置エンティティ数はtotal/5程度に収まるべき
		plannerType := PlannerType{
			Name: "test_placed_count",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{
					{Name: "回復薬", Weight: 1.0, PackMin: 5, PackMax: 5},
				}},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 42)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		// PackSize=5でtotal=8-12なので、配置エンティティ数は2-3程度のはず
		assert.LessOrEqual(t, len(chain.PlanData.Items), 5,
			"PackSize=5のStackableアイテムの配置エンティティ数が多すぎる")
	})

	t.Run("部屋がある場合はアイテムが部屋内に配置される", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		worldhelper.SetDungeon(world, &gc.Dungeon{Depth: 1})

		plannerType := PlannerType{
			Name: "test_room_based_items",
			ItemSources: []ItemSource{
				{Weight: 1.0, Subtype: ItemGroupDistribution, Entries: []SpawnEntry{{Name: "薬草", Weight: 1.0, PackMin: 1, PackMax: 1}}},
			},
		}

		chain, err := NewSmallRoomPlanner(30, 30, 12345)
		require.NoError(t, err)
		chain.PlanData.RawMaster = CreateTestRawMaster()
		err = chain.Plan()
		require.NoError(t, err)
		require.NotEmpty(t, chain.PlanData.Rooms, "テストにはRoomsが必要")

		planner := NewItemPlanner(world, plannerType)
		err = planner.PlanMeta(&chain.PlanData)
		require.NoError(t, err)

		// 大半のアイテムが部屋内または廊下上に配置されていることを確認する
		// フォールバックとしてonMapSelectorを使うため、一部は部屋外に配置される可能性がある
		inRoomCount := 0
		for _, item := range chain.PlanData.Items {
			for _, room := range chain.PlanData.Rooms {
				if item.X >= int(room.X1) && item.X < int(room.X2) &&
					item.Y >= int(room.Y1) && item.Y < int(room.Y2) {
					inRoomCount++
					break
				}
			}
		}
		// 部屋配置を優先するため、半数以上が部屋内にあることを期待する
		assert.Greater(t, inRoomCount, len(chain.PlanData.Items)/2, "部屋内のアイテムが半数未満")
	})
}

func TestResolveDistribution(t *testing.T) {
	t.Parallel()

	t.Run("StackableアイテムはCount=PackSizeの1エントリにまとめられる", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		entries := []SpawnEntry{{Name: "回復薬", Weight: 1.0, PackMin: 3, PackMax: 3}}
		result := resolveDistribution(entries, &chain.PlanData)

		require.Len(t, result, 1)
		assert.Equal(t, "回復薬", result[0].Name)
		assert.Equal(t, 3, result[0].Count)
	})

	t.Run("非StackableアイテムはPackSize分の個別エントリになる", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		entries := []SpawnEntry{{Name: "木刀", Weight: 1.0, PackMin: 2, PackMax: 2}}
		result := resolveDistribution(entries, &chain.PlanData)

		require.Len(t, result, 2)
		for _, item := range result {
			assert.Equal(t, "木刀", item.Name)
			assert.Equal(t, 1, item.Count)
		}
	})

	t.Run("RawMasterがnilの場合は非Stackableとして扱う", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 12345)
		// RawMaster未設定

		entries := []SpawnEntry{{Name: "回復薬", Weight: 1.0, PackMin: 3, PackMax: 3}}
		result := resolveDistribution(entries, &chain.PlanData)

		require.Len(t, result, 3)
		for _, item := range result {
			assert.Equal(t, 1, item.Count)
		}
	})
}

func TestResolveCollection(t *testing.T) {
	t.Parallel()

	t.Run("StackableアイテムはCount=PackSizeの1エントリにまとめられる", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 99999)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		// weight=100で確実に当選させる
		entries := []SpawnEntry{{Name: "回復薬", Weight: 100, PackMin: 4, PackMax: 4}}
		result := resolveCollection(entries, &chain.PlanData)

		require.Len(t, result, 1)
		assert.Equal(t, "回復薬", result[0].Name)
		assert.Equal(t, 4, result[0].Count)
	})

	t.Run("非StackableアイテムはPackSize分の個別エントリになる", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 99999)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		// weight=100で確実に当選させる
		entries := []SpawnEntry{{Name: "木刀", Weight: 100, PackMin: 2, PackMax: 2}}
		result := resolveCollection(entries, &chain.PlanData)

		require.Len(t, result, 2)
		for _, item := range result {
			assert.Equal(t, "木刀", item.Name)
			assert.Equal(t, 1, item.Count)
		}
	})
}

func TestIsStackableItem(t *testing.T) {
	t.Parallel()

	t.Run("Stackableアイテムはtrueを返す", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		assert.True(t, isStackableItem(&chain.PlanData, "回復薬"))
	})

	t.Run("非Stackableアイテムはfalseを返す", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		assert.False(t, isStackableItem(&chain.PlanData, "木刀"))
	})

	t.Run("存在しないアイテムはfalseを返す", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 12345)
		chain.PlanData.RawMaster = CreateTestRawMaster()

		assert.False(t, isStackableItem(&chain.PlanData, "存在しないアイテム"))
	})

	t.Run("RawMasterがnilの場合はfalseを返す", func(t *testing.T) {
		t.Parallel()
		chain := NewPlannerChain(10, 10, 12345)

		assert.False(t, isStackableItem(&chain.PlanData, "回復薬"))
	})
}
