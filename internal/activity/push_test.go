package activity_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addCube は指定座標に押せるキューブを作る
func addCube(world w.World, coord consts.Coord[consts.Tile]) ecs.Entity {
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.Prop.Add(e, &gc.Prop{})
	world.Components.BlockPass.Add(e, &gc.BlockPass{})
	world.Components.Pushable.Add(e, &gc.Pushable{})
	return e
}

// addPusher は指定座標・APのプレイヤーを作る
func addPusher(world w.World, coord consts.Coord[consts.Tile], ap int) ecs.Entity {
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.Player.Add(e, &gc.Player{})
	world.Components.TurnBased.Add(e, &gc.TurnBased{AP: gc.IntPool{Max: ap, Current: ap}})
	return e
}

func addWall(world w.World, coord consts.Coord[consts.Tile]) ecs.Entity {
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.BlockPass.Add(e, &gc.BlockPass{})
	return e
}

func TestPushActivity_総重量とパーティAPで決まる複数ターンをかけて押す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 空のキューブの PushCost は基準 1000。AP 500 なので所要2ターン
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase/2)

	result, err := activity.Execute(activity.NewPushActivity(cube, gc.DirectionRight), player, world)
	require.NoError(t, err)
	assert.Equal(t, gc.ActivityStateRunning, result.State, "重いので初回では完了せず継続する")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "開始だけではまだ動かない")

	// 1ターン目
	activity.ProcessContinuousActivities(world)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "1ターンでは足りず動かない")

	// 2ターン目で1タイル前進し、押し手も追随する
	activity.ProcessContinuousActivities(world)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord, "2ターンでキューブが前進する")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord, "押し手はキューブの空けたタイルへ追随する")
}

func TestPushActivity_プレイヤーが行けない先へは押せない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)
	addWall(world, consts.Coord[consts.Tile]{X: 6, Y: 5})

	_, err := activity.Execute(activity.NewPushActivity(cube, gc.DirectionRight), player, world)
	require.Error(t, err, "押し先が壁なら押せない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "押せなければキューブは動かない")
}

func TestPushActivity_APが無ければ押せない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, 0)

	_, err := activity.Execute(activity.NewPushActivity(cube, gc.DirectionRight), player, world)
	require.Error(t, err, "APが無ければ押せない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord)
}
