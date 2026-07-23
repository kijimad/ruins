package mapspawner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSpawnPlan は spawnNPCs/spawnItems/spawnProps を直接呼ぶためのテスト用MetaPlanを生成する。
// RawMaster は world 共有のものをそのまま使う。
func newTestSpawnPlan(world w.World) *mapplanner.MetaPlan {
	return &mapplanner.MetaPlan{
		Level:     gc.Level{TileWidth: 10, TileHeight: 10},
		RawMaster: &world.Resources.RawMaster,
	}
}

func TestSpawnNPCs_未知のNPC名はエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.NPCs = []mapplanner.NPCSpec{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}, Name: "存在しないNPC"},
	}

	err := spawnNPCs(world, plan, 0, 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "存在しないNPC")
}

func TestSpawnNPCs_中立NPCを生成する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.NPCs = []mapplanner.NPCSpec{
		{Coord: consts.Coord[consts.Tile]{X: 3, Y: 4}, Name: "商人"},
	}

	err := spawnNPCs(world, plan, 0, 0)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.FactionNeutral](world.ECS).Query()
	count := 0
	for query.Next() {
		count++
	}
	assert.Equal(t, 1, count, "中立NPCが1体生成される")
}

func TestSpawnNPCs_敵NPCを生成する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.NPCs = []mapplanner.NPCSpec{
		{Coord: consts.Coord[consts.Tile]{X: 2, Y: 2}, Name: "光虫"},
	}

	err := spawnNPCs(world, plan, 0, 0)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.FactionEnemy](world.ECS).Query()
	count := 0
	bossCount := 0
	for query.Next() {
		count++
		if world.Components.Boss.Has(query.Entity()) {
			bossCount++
		}
	}
	assert.Equal(t, 1, count, "敵NPCが1体生成される")
	assert.Equal(t, 0, bossCount, "ボスフラグは付かない")
}

func TestSpawnNPCs_ボス敵NPCにBossコンポーネントが付く(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.NPCs = []mapplanner.NPCSpec{
		{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}, Name: "凍晶化した猟師"},
	}

	err := spawnNPCs(world, plan, 0, 0)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.FactionEnemy](world.ECS).Query()
	bossCount := 0
	for query.Next() {
		if world.Components.Boss.Has(query.Entity()) {
			bossCount++
		}
	}
	assert.Equal(t, 1, bossCount, "isBoss=trueのNPCにはBossコンポーネントが付く")
}

func TestSpawnNPCs_オフセットが座標に加算される(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.NPCs = []mapplanner.NPCSpec{
		{Coord: consts.Coord[consts.Tile]{X: 2, Y: 3}, Name: "商人"},
	}

	err := spawnNPCs(world, plan, 100, 200)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	found := false
	for query.Next() {
		g := world.Components.GridElement.Get(query.Entity())
		if g.X == 102 && g.Y == 203 {
			found = true
		}
	}
	assert.True(t, found, "NPC座標はオフセット分ずれる")
}

func TestSpawnItems_個数0以下はエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.Items = []mapplanner.ItemSpec{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}, Name: "木刀", Count: 0},
	}

	err := spawnItems(world, plan, 0, 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "count=0")
}

func TestSpawnItems_未知のアイテム名はエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.Items = []mapplanner.ItemSpec{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}, Name: "存在しないアイテム", Count: 1},
	}

	err := spawnItems(world, plan, 0, 0)
	require.Error(t, err)
	assert.ErrorContains(t, err, "存在しないアイテム")
}

func TestSpawnItems_有効なアイテムを生成する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.Items = []mapplanner.ItemSpec{
		{Coord: consts.Coord[consts.Tile]{X: 4, Y: 4}, Name: "木刀", Count: 1},
	}

	err := spawnItems(world, plan, 10, 20)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	found := false
	for query.Next() {
		g := world.Components.GridElement.Get(query.Entity())
		if g.X == 14 && g.Y == 24 {
			found = true
		}
	}
	assert.True(t, found, "アイテムはオフセット込みの座標に生成される")
}

func TestPopulateStorageLoot_未知のルートテーブルはエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.RNG = rand.New(rand.NewPCG(1, 1))
	plan.Depth = 1

	tableName := "存在しないテーブル"
	propRaw := oapi.Prop{
		Storage: &oapi.StorageRaw{LootTableName: &tableName},
	}
	storageEntity := world.ECS.NewEntity()

	err := populateStorageLoot(world, plan, storageEntity, propRaw)
	require.Error(t, err)
	assert.ErrorContains(t, err, "存在しないテーブル")
}

func TestPopulateStorageLoot_ルートテーブルからアイテムを収納する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.RNG = rand.New(rand.NewPCG(1, 1))
	plan.Depth = 1

	propRaw, err := raw.GetProp(world.Resources.RawMaster, "木箱")
	require.NoError(t, err)
	require.NotNil(t, propRaw.Storage)

	storageEntity := world.ECS.NewEntity()

	err = populateStorageLoot(world, plan, storageEntity, propRaw)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	count := 0
	for query.Next() {
		loc := world.Components.LocationInStorage.Get(query.Entity())
		if loc.Owner == storageEntity {
			count++
		}
	}
	assert.GreaterOrEqual(t, count, 1, "ルート数の下限以上のアイテムが収納される")
	assert.LessOrEqual(t, count, 3, "ルート数の上限以下のアイテムが収納される")
}

func TestPopulateStorageLoot_最小数が最大数を超える場合は最大数に丸める(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.RNG = rand.New(rand.NewPCG(1, 1))
	plan.Depth = 1

	tableName := "廃墟"
	var lootMin, lootMax int32 = 9, 2
	propRaw := oapi.Prop{
		Storage: &oapi.StorageRaw{
			LootTableName: &tableName,
			LootCountMin:  &lootMin,
			LootCountMax:  &lootMax,
		},
	}
	storageEntity := world.ECS.NewEntity()

	err := populateStorageLoot(world, plan, storageEntity, propRaw)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	count := 0
	for query.Next() {
		loc := world.Components.LocationInStorage.Get(query.Entity())
		if loc.Owner == storageEntity {
			count++
		}
	}
	assert.Equal(t, 2, count, "countMinがcountMaxを超える場合はcountMaxに丸められる")
}
