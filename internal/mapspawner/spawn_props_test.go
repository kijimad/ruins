package mapspawner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDoorPlan は扉の向き判定を検証するための3x3マップを持つMetaPlanを生成する。
// WWW
// WDW
// WWW
// 中央がドアで左右が壁のため、縦向きと判定される
func newTestDoorPlan(world w.World) *mapplanner.MetaPlan {
	tiles := make([]rawschema.Tile, 9)
	for i := range tiles {
		tiles[i] = rawschema.Tile{BlockPass: true}
	}
	tiles[4] = rawschema.Tile{BlockPass: false}

	return &mapplanner.MetaPlan{
		Level:     gc.Level{TileWidth: 3, TileHeight: 3},
		RawMaster: &world.Resources.RawMaster,
		Tiles:     tiles,
	}
}

func TestSpawnProps_未知のprops名はエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.Props = []mapplanner.PropsSpec{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}, Name: "存在しないprops"},
	}

	err := spawnProps(world, plan, 0, 0)
	var notFound raw.KeyNotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.Equal(t, raw.KeyNotFoundError{Key: "存在しないprops", Collection: "Props"}, notFound)
}

func TestSpawnProps_有効なpropsをオフセット込みで生成する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.Props = []mapplanner.PropsSpec{
		{Coord: consts.Coord[consts.Tile]{X: 2, Y: 3}, Name: "alarm_clock"},
	}

	err := spawnProps(world, plan, 10, 20)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	found := false
	for query.Next() {
		g := world.Components.GridElement.Get(query.Entity())
		if g.X == 12 && g.Y == 23 {
			found = true
		}
	}
	assert.True(t, found, "propsはオフセット込みの座標に生成される")
}

func TestSpawnProps_扉は向きを設定して閉じた状態で生成される(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestDoorPlan(world)
	plan.Props = []mapplanner.PropsSpec{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}, Name: "door"},
	}

	err := spawnProps(world, plan, 0, 0)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.Door](world.ECS).Query()
	found := false
	for query.Next() {
		entity := query.Entity()
		door := world.Components.Door.Get(entity)
		assert.Equal(t, gc.DoorOrientationVertical, door.Orientation, "左右が壁なら縦向きになる")
		assert.False(t, door.IsOpen, "生成直後は閉じている")
		assert.True(t, world.Components.BlockPass.Has(entity), "閉じた扉はBlockPassを持つ")
		found = true
	}
	assert.True(t, found, "扉エンティティが生成される")
}

func TestSpawnProps_収納propはルートテーブルからアイテムを収納する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	plan := newTestSpawnPlan(world)
	plan.RNG = rand.New(rand.NewPCG(1, 1))
	plan.Danger = 1
	plan.Props = []mapplanner.PropsSpec{
		{Coord: consts.Coord[consts.Tile]{X: 4, Y: 4}, Name: "wooden_crate"},
	}

	err := spawnProps(world, plan, 0, 0)
	require.NoError(t, err)

	query := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	count := 0
	for query.Next() {
		count++
	}
	assert.Positive(t, count, "収納アイテムが1つ以上生成される")
}

func TestSpawnProps_棚の向きを地図を作るときに決めて付ける(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	tiles := make([]rawschema.Tile, 12)
	for i := range tiles {
		tiles[i] = rawschema.Tile{BlockPass: i < 4}
	}
	plan := &mapplanner.MetaPlan{
		Level:     gc.Level{TileWidth: 4, TileHeight: 3},
		RawMaster: &world.Resources.RawMaster,
		Tiles:     tiles,
		Props: []mapplanner.PropsSpec{
			{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}, Name: "book_shelf_2"},
			{Coord: consts.Coord[consts.Tile]{X: 2, Y: 1}, Name: "book_shelf_2"},
		},
	}
	require.NoError(t, spawnProps(world, plan, 0, 0))

	var shelves []ecs.Entity
	q := ecs.NewFilter1[gc.Pose](world.ECS).Query()
	for q.Next() {
		shelves = append(shelves, q.Entity())
	}
	require.Len(t, shelves, 2)
	for _, e := range shelves {
		assert.Equal(t, gc.Pose{Facing: consts.South, Anchor: gc.AnchorBack}, *world.Components.Pose.Get(e))
	}
}

func TestPlanFacings_同じマスの置物はどれか1つが塞げば塞がりに数える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	tiles := make([]rawschema.Tile, 9)
	// 重なりの順を入れ替えても、樽が塞ぐので本棚は南を向く
	for _, order := range [][]string{{"barrel", "soil_crack"}, {"soil_crack", "barrel"}} {
		plan := &mapplanner.MetaPlan{
			Level:     gc.Level{TileWidth: 3, TileHeight: 3},
			RawMaster: &world.Resources.RawMaster,
			Tiles:     tiles,
			Props: []mapplanner.PropsSpec{
				{Coord: consts.Coord[consts.Tile]{X: 1, Y: 0}, Name: order[0]},
				{Coord: consts.Coord[consts.Tile]{X: 1, Y: 0}, Name: order[1]},
				{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}, Name: "book_shelf_2"},
			},
		}
		got, err := planPoses(plan)
		require.NoError(t, err)
		assert.Equal(t, map[consts.Coord[consts.Tile]]gc.Pose{{X: 1, Y: 1}: {Facing: consts.South, Anchor: gc.AnchorBack}}, got, "%v", order)
	}
}
