package overworld

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countStoredItems は収納に入っているアイテムの数を返す
func countStoredItems(world w.World) int {
	q := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	n := 0
	for q.Next() {
		n++
	}
	return n
}

// countFieldItems は床に置かれたアイテムの数を返す
func countFieldItems(world w.World) int {
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	n := 0
	for q.Next() {
		n++
	}
	return n
}

func TestSpawnFieldLoot_床に戦利品が出る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	rng := rand.New(rand.NewPCG(1, 0x4))

	pos := consts.Coord[consts.Tile]{X: 0, Y: 0}
	require.NoError(t, spawnFieldLoot(world, "healing_item", pos, rng))
	assert.Positive(t, countFieldItems(world), "loot group から床にアイテムが1つ以上出る")
}

func TestSpawnFieldLoot_未存在グループはエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	rng := rand.New(rand.NewPCG(1, 0x4))
	pos := consts.Coord[consts.Tile]{X: 0, Y: 0}
	require.Error(t, spawnFieldLoot(world, "no_such_group", pos, rng))
}

// 家は荒らされ率を持つので、押入れを複数置いて数える
func TestPopulateStorageLoot_収納家具に戦利品が入る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	rng := rand.New(rand.NewPCG(1, 0x4))

	for i := range 10 {
		closet, err := lifecycle.SpawnProp(world, "closet", consts.Tile(i), 0)
		require.NoError(t, err)
		require.NoError(t, populateStorageLoot(world, closet, "closet", "house_closet", "house", rng))
	}
	assert.Positive(t, countStoredItems(world), "収納家具の押入れに戦利品が入る")

	before := countStoredItems(world)
	candle, err := lifecycle.SpawnProp(world, "candle", 1, 1)
	require.NoError(t, err)
	require.NoError(t, populateStorageLoot(world, candle, "candle", "", "house", rng))
	assert.Equal(t, before, countStoredItems(world), "収納でない蝋燭は戦利品を入れない")
}

func TestPopulateStorageLoot_未登録の施設はエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	closet, err := lifecycle.SpawnProp(world, "closet", 0, 0)
	require.NoError(t, err)
	require.Error(t, populateStorageLoot(world, closet, "closet", "house_closet", "no_such", rand.New(rand.NewPCG(1, 0x4))))
}

func TestPopulateStorageLoot_表の無い収納家具はエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	closet, err := lifecycle.SpawnProp(world, "closet", 0, 0)
	require.NoError(t, err)
	require.Error(t, populateStorageLoot(world, closet, "closet", "", "house", rand.New(rand.NewPCG(1, 0x4))))
}

func TestPopulateStorageLoot_施設の荒らされ率で空のまま残す(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		fac       string
		wantEmpty int
	}{
		{"店は8割が空", "store", 80},
		{"骨董品店は5割が空", "antique", 50},
		{"倉庫は4割が空", "depot", 40},
		{"オフィスは4割が空", "office", 40},
		{"家は3割が空", "house", 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			rng := rand.New(rand.NewPCG(1, 0x4))
			const n = 400
			empty := 0
			for i := range n {
				ent, err := lifecycle.SpawnProp(world, "goods_shelf", consts.Tile(i), 0)
				require.NoError(t, err)
				require.NoError(t, populateStorageLoot(world, ent, "goods_shelf", "store_goods", tt.fac, rng))
				if len(query.GetStorageItems(world, ent)) == 0 {
					empty++
				}
			}
			assert.InDelta(t, tt.wantEmpty, empty*100/n, 6, "空の割合が施設の荒らされ率に近い")
		})
	}
}

func TestSpawnInteriorBeing(t *testing.T) {
	t.Parallel()

	t.Run("施設の敵の表から1体を指定の位置へ置く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		table, err := urbanEnemyTableFor(world.Resources.RawMaster, "store")
		require.NoError(t, err)
		members := map[string]bool{}
		for _, e := range table.Entries {
			members[e.Id] = true
		}
		pos := consts.Coord[consts.Tile]{X: 3, Y: 4}
		require.NoError(t, spawnFacilityBeing(world, "store", raw.MinDanger, pos, rand.New(rand.NewPCG(1, 0x5))))

		var got []string
		q := ecs.NewFilter2[gc.FactionEnemy, gc.GridElement](world.ECS).Query()
		for q.Next() {
			_, ge := q.Get()
			assert.Equal(t, pos, ge.Coord)
			got = append(got, query.GetEntityID(q.Entity(), world))
		}
		require.Len(t, got, 1)
		assert.True(t, members[got[0]], "店の敵の表から引く")
	})

	t.Run("未登録の施設はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		require.Error(t, spawnFacilityBeing(world, "no_such", raw.MinDanger, consts.Coord[consts.Tile]{X: 3, Y: 4}, rand.New(rand.NewPCG(1, 0x5))))
	})
}
