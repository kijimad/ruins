package mapspawner

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/oapi"
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
// 中央がドアで左右が壁のため、縦向きと判定されるはず
func newTestDoorPlan(world w.World) *mapplanner.MetaPlan {
	tiles := make([]oapi.Tile, 9)
	for i := range tiles {
		tiles[i] = oapi.Tile{BlockPass: true}
	}
	tiles[4] = oapi.Tile{BlockPass: false}

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
	require.Error(t, err)
	assert.ErrorContains(t, err, "存在しないprops")
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
	plan.Depth = 1
	plan.Props = []mapplanner.PropsSpec{
		{Coord: consts.Coord[consts.Tile]{X: 4, Y: 4}, Name: "木箱"},
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
