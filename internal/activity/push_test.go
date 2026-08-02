package activity_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecuteMoveAction_コントロールパネルへ歩き込むと開く は、パネルが NPC への話しかけと同じく
// 移動キーの歩き込みだけで開くことを検証する。Config を ActivationWayOnCollision にし、bump 経路の
// switch に case を繋いだ回帰。
func TestExecuteMoveAction_コントロールパネルへ歩き込むと開く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5}, consts.PushCostBase)

	// プレイヤーの右隣に BlockPass のコントロールパネルを置く
	panel := world.ECS.NewEntity()
	world.Components.GridElement.Add(panel, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 6, Y: 5}})
	world.Components.BlockPass.Add(panel, &gc.BlockPass{})
	world.Components.Interactable.Add(panel, &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionCubePanel}})

	require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionRight))

	// 歩き込みでパネルを開くリクエストが出る
	req := lifecycle.ConsumeStateChange(world)
	require.NotNil(t, req, "歩き込みで状態変更リクエストが出る")
	_, ok := req.Payload.(gc.OpenCubePanel)
	assert.True(t, ok, "リクエストはコントロールパネルを開く")

	// パネルは BlockPass。開くだけで、そのタイルへは移動しない
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord, "パネルを開いても移動はしない")
}

// addCube は指定座標に押せるキューブを作る
func addCube(t *testing.T, world w.World, coord consts.Coord[consts.Tile]) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.Fixed.Add(e, &gc.Fixed{})
	world.Components.BlockPass.Add(e, &gc.BlockPass{})
	world.Components.Pushable.Add(e, &gc.Pushable{})
	return e
}

// addPusher は指定座標・APのプレイヤーを作る
func addPusher(t *testing.T, world w.World, coord consts.Coord[consts.Tile], ap int) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.Player.Add(e, &gc.Player{})
	world.Components.TurnBased.Add(e, &gc.TurnBased{AP: gc.IntPool{Max: ap, Current: ap}})
	return e
}

func addWall(t *testing.T, world w.World, coord consts.Coord[consts.Tile]) {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.GridElement.Add(e, &gc.GridElement{Coord: coord})
	world.Components.BlockPass.Add(e, &gc.BlockPass{})
}

func TestPushBehavior_総重量とパーティAPで決まる複数ターンをかけて押す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// 空のキューブの PushCost は基準 1000。AP 500 なので所要2ターン
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase/2)

	result, err := activity.Execute(activity.NewPushBehavior(cube, gc.DirectionRight), player, world)
	require.NoError(t, err)
	assert.Equal(t, gc.ActivityStateRunning, result.State, "重いので初回では完了せず継続する")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "開始だけではまだ動かない")

	// 1ターン目
	activity.ProcessContinuousActivities(world)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "1ターンでは足りず動かない")

	// 2ターン目でキューブだけが1タイル前進する。押し手は追随しない
	activity.ProcessContinuousActivities(world)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord, "2ターンでキューブが前進する")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 4, Y: 5}, world.Components.GridElement.Get(player).Coord, "押し手は追随せず元位置に留まる")
}

// addInteriorWeight はキューブ内部ステージの床へ重量物を1つ据える。押しコストに乗るかの検証に使う。
func addInteriorWeight(t *testing.T, world w.World, mg consts.Milligram) {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.Weight.Add(e, &gc.Weight{Milligram: mg})
	world.Components.LocationOnField.Add(e, &gc.LocationOnField{})
	world.Components.StageBound.Add(e, &gc.StageBound{Key: gc.NewCubeInteriorStage()})
}

// TestPushBehavior_内部に置いた物の重量が押しターンに乗る は、内部の総重量が実際の押しコストへ
// 反映されることを検証する。空なら1ターンで押し切れるAPでも、内部に重量物があると所要ターンが
// 増えて初回では完了しない。キューブと内部のリンク解決が壊れて総重量が常に0になる回帰を防ぐ。
func TestPushBehavior_内部に置いた物の重量が押しターンに乗る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	// 空のキューブなら PushCostBase ちょうどのAPで1ターンで押し切れる
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)
	// 内部に 8kg を据えると PushCost が基準を超え、同じAPでは1ターンで足りなくなる
	addInteriorWeight(t, world, 8*consts.MilligramPerKg)

	result, err := activity.Execute(activity.NewPushBehavior(cube, gc.DirectionRight), player, world)
	require.NoError(t, err)
	assert.Equal(t, gc.ActivityStateRunning, result.State, "内部の重量ぶん所要ターンが増え、初回では完了しない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "重くて1ターンでは動かない")
}

func TestPushBehavior_プレイヤーが行けない先へは押せない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)
	addWall(t, world, consts.Coord[consts.Tile]{X: 6, Y: 5})

	_, err := activity.Execute(activity.NewPushBehavior(cube, gc.DirectionRight), player, world)
	require.Error(t, err, "押し先が壁なら押せない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "押せなければキューブは動かない")
}

func TestPushBehavior_APが無ければ押せない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5}, 0)

	_, err := activity.Execute(activity.NewPushBehavior(cube, gc.DirectionRight), player, world)
	require.Error(t, err, "APが無ければ押せない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord)
}

func TestExecuteMoveAction_キューブへの移動は押しになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// プレイヤーの右隣にキューブ。AP は1タイルぶん足りる
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)

	// 右へ移動入力すると、歩行でなく押しアクティビティが始まる。
	// AP がちょうど足りるので初回ステップで押し切り、Execute 内で即時完了する
	require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionRight))

	// キューブだけが前進し、押し手は追随しない
	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord)
	assert.Equal(t, consts.Coord[consts.Tile]{X: 4, Y: 5}, world.Components.GridElement.Get(player).Coord, "押し手は追随せず元位置に留まる")
}

// TestExecuteMoveAction_押しの次の入力で空いたタイルへ進む は、押しでキューブが抜けた後、同じ方向を
// もう一度入力するとプレイヤーが空いたタイルへ普通に一歩進むことを検証する。方向を押し続けると
// 押しと移動が交互に起きてキューブが進む、案Aの体感を固定する。
func TestExecuteMoveAction_押しの次の入力で空いたタイルへ進む(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase*2)

	// 1回目: 押し。AP がちょうど足りるので初回ステップで完了し、キューブが {6,5} へ抜け、プレイヤーは {4,5} のまま
	require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionRight))
	require.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord)
	require.Equal(t, consts.Coord[consts.Tile]{X: 4, Y: 5}, world.Components.GridElement.Get(player).Coord)

	// 2回目: 前は空いたので通常移動。プレイヤーが {5,5} へ進みキューブに再隣接する
	require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionRight))
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(player).Coord, "空いたタイルへ一歩進む")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord, "キューブは動かない")
}

func TestPullBehavior_壁際のキューブを引き出せる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	// キューブの西は壁。押しでは西へ動かせない。東に立つプレイヤーが引いて剥がす
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	addWall(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5})
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 6, Y: 5}, consts.PushCostBase)

	// AP がちょうど足りるので初回ステップで引き切り、Execute 内で即時完了する
	_, err := activity.Execute(activity.NewPullBehavior(cube), player, world)
	require.NoError(t, err)

	assert.Equal(t, consts.Coord[consts.Tile]{X: 6, Y: 5}, world.Components.GridElement.Get(cube).Coord, "キューブはプレイヤーの元タイルへ引かれる")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 7, Y: 5}, world.Components.GridElement.Get(player).Coord, "プレイヤーは1タイル後退する")
}

func TestPullBehavior_後退スペースが無ければ引けない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	player := addPusher(t, world, consts.Coord[consts.Tile]{X: 6, Y: 5}, consts.PushCostBase)
	addWall(t, world, consts.Coord[consts.Tile]{X: 7, Y: 5}) // 後退先を塞ぐ

	_, err := activity.Execute(activity.NewPullBehavior(cube), player, world)
	require.Error(t, err, "後退スペースが無ければ引けない")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "引けないのでキューブは動かない")
}

func TestExecuteMoveAction_押し先が壁ならエラーにせず何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube := addCube(t, world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	addPusher(t, world, consts.Coord[consts.Tile]{X: 4, Y: 5}, consts.PushCostBase)
	addWall(t, world, consts.Coord[consts.Tile]{X: 6, Y: 5}) // 押し先を塞ぐ

	// 壁に歩き込むのと同じく、押せないときは致命エラーでなく no-op になる
	require.NoError(t, activity.ExecuteMoveAction(world, gc.DirectionRight))
	assert.Equal(t, consts.Coord[consts.Tile]{X: 5, Y: 5}, world.Components.GridElement.Get(cube).Coord, "押せないのでキューブは動かない")
}
