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

func addWall(world w.World, coord consts.Coord[consts.Tile]) {
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.BlockPass.Add(e, &gc.BlockPass{})
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

// addInteriorWeight はキューブ内装ステージへ重量物を1つ据える。押しコストに乗るかの検証に使う。
func addInteriorWeight(world w.World, mg consts.Milligram) {
	e := world.ECS.NewEntity()
	world.Components.Weight.Add(e, &gc.Weight{Milligram: mg})
	world.Components.StageBound.Add(e, &gc.StageBound{Key: gc.NewCubeInteriorStage()})
}

// TestPushActivity_内装に置いた物の重量が押しターンに乗る は、内装の総重量が実際の押しコストへ
// 反映されることを検証する。空なら1ターンで押し切れるAPでも、内装に重量物があると所要ターンが
// 増えて初回では完了しない。キューブと内装のリンク解決が壊れて総重量が常に0になる回帰を防ぐ。
func TestPushActivity_内装に置いた物の重量が押しターンに乗る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	// 空のキューブなら PushCostBase ちょうどのAPで1ターンで押し切れる
	player := addPusher(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)
	// 内装に 8kg を据えると PushCost が基準を超え、同じAPでは1ターンで足りなくなる
	addInteriorWeight(world, 8*consts.MilligramPerKg)

	result, err := activity.Execute(activity.NewPushActivity(cube, gc.DirectionRight), player, world)
	require.NoError(t, err)
	assert.Equal(t, gc.ActivityStateRunning, result.State, "内装の重量ぶん所要ターンが増え、初回では完了しない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "重くて1ターンでは動かない")
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

func TestExecuteMoveAction_キューブへの移動は押しになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// プレイヤーの右隣にキューブ。AP は1タイルぶん足りる
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)

	// 右へ移動入力すると、歩行でなく押しアクティビティが始まる
	require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionRight))

	// AP がちょうど足りるので1ターンで押し切れ、キューブと押し手が前進する
	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord)
}

func TestPullActivity_壁際のキューブを引き出せる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// キューブの西は壁。押しでは西へ動かせない。東に立つプレイヤーが引いて剥がす
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	addWall(world, consts.Coord[consts.Tile]{X: 4, Y: 5})
	player := addPusher(world, consts.Coord[consts.Tile]{X: 6, Y: 5}, consts.PushCostBase)

	_, err := activity.Execute(activity.NewPullActivity(cube), player, world)
	require.NoError(t, err)

	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord, "キューブはプレイヤーの元タイルへ引かれる")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 7, Y: 5}, world.Components.GridElement.Get(player).Coord, "プレイヤーは1タイル後退する")
}

func TestPullActivity_後退スペースが無ければ引けない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(world, consts.Coord[consts.Tile]{X: 6, Y: 5}, consts.PushCostBase)
	addWall(world, consts.Coord[consts.Tile]{X: 7, Y: 5}) // 後退先を塞ぐ

	_, err := activity.Execute(activity.NewPullActivity(cube), player, world)
	require.Error(t, err, "後退スペースが無ければ引けない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "引けないのでキューブは動かない")
}

func TestExecuteMoveAction_押し先が壁ならエラーにせず何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	addPusher(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)
	addWall(world, consts.Coord[consts.Tile]{X: 6, Y: 5}) // 押し先を塞ぐ

	// 壁に歩き込むのと同じく、押せないときは致命エラーでなく no-op になる
	require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionRight))
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "押せないのでキューブは動かない")
}
