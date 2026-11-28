package aiinput

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanRaidAction(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	cubeAt := at(20, 20)

	t.Run("プレイヤーが見えなければ地図を下ってキューブへ寄る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
		require.NoError(t, err)
		cube, err := lifecycle.SpawnCube(world, cubeAt)
		require.NoError(t, err)
		world.Components.Deployed.Add(cube, &gc.Deployed{})
		raider, err := lifecycle.SpawnEnemy(world, at(20, 26), "stray_dog")
		require.NoError(t, err)
		world.Components.SoloAI.Get(raider).Movement = rawschema.Converge

		rp := newSoloPlanner(newTestRNG())
		act := rp.Plan(world, raider)

		require.NotNil(t, act)
		require.Equal(t, gc.BehaviorMove, act.BehaviorName)
		p, ok := act.Params.(*gc.MoveParams)
		require.True(t, ok)
		assert.Equal(t, at(20, 25), p.Destination.Coord)
	})

	t.Run("キューブの隣ならキューブを殴る", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
		require.NoError(t, err)
		cube, err := lifecycle.SpawnCube(world, cubeAt)
		require.NoError(t, err)
		world.Components.Deployed.Add(cube, &gc.Deployed{})
		raider, err := lifecycle.SpawnEnemy(world, at(20, 21), "stray_dog")
		require.NoError(t, err)
		world.Components.SoloAI.Get(raider).Movement = rawschema.Converge

		rp := newSoloPlanner(newTestRNG())
		act := rp.Plan(world, raider)

		require.NotNil(t, act)
		require.Equal(t, gc.BehaviorMelee, act.BehaviorName)
		p, ok := act.Params.(*gc.MeleeParams)
		require.True(t, ok)
		assert.Equal(t, cube, p.Target)
	})

	t.Run("プレイヤーが見えていればプレイヤーを追う", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		player, err := lifecycle.SpawnPlayer(world, at(20, 30), "ash")
		require.NoError(t, err)
		cube, err := lifecycle.SpawnCube(world, cubeAt)
		require.NoError(t, err)
		world.Components.Deployed.Add(cube, &gc.Deployed{})
		raider, err := lifecycle.SpawnEnemy(world, at(20, 29), "stray_dog")
		require.NoError(t, err)
		world.Components.SoloAI.Get(raider).Movement = rawschema.Converge

		rp := newSoloPlanner(newTestRNG())
		act := rp.Plan(world, raider)

		require.NotNil(t, act)
		require.Equal(t, gc.BehaviorMelee, act.BehaviorName)
		p, ok := act.Params.(*gc.MeleeParams)
		require.True(t, ok)
		assert.Equal(t, player, p.Target)
	})

	t.Run("道の上の仲間は重いので回り込む", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
		require.NoError(t, err)
		cube, err := lifecycle.SpawnCube(world, cubeAt)
		require.NoError(t, err)
		world.Components.Deployed.Add(cube, &gc.Deployed{})
		raider, err := lifecycle.SpawnEnemy(world, at(20, 26), "stray_dog")
		require.NoError(t, err)
		world.Components.SoloAI.Get(raider).Movement = rawschema.Converge
		_, err = lifecycle.SpawnEnemy(world, at(20, 25), "stray_dog")
		require.NoError(t, err)

		rp := newSoloPlanner(newTestRNG())
		act := rp.Plan(world, raider)

		require.NotNil(t, act)
		require.Equal(t, gc.BehaviorMove, act.BehaviorName)
		p, ok := act.Params.(*gc.MoveParams)
		require.True(t, ok)
		assert.Equal(t, at(19, 25), p.Destination.Coord)
	})
}

func TestPlanRaidAction_見えた相手よりキューブが近ければキューブへ向かう(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, at(20, 26), "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, at(20, 20))
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	raider, err := lifecycle.SpawnEnemy(world, at(20, 22), "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge

	rp := newSoloPlanner(newTestRNG())
	act := rp.Plan(world, raider)

	require.NotNil(t, act)
	require.Equal(t, gc.BehaviorMove, act.BehaviorName)
	p, ok := act.Params.(*gc.MoveParams)
	require.True(t, ok)
	assert.Equal(t, at(20, 21), p.Destination.Coord)
}

func TestPlanRaidAction_道の上の扉を殴る(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, at(20, 20))
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	door, err := lifecycle.SpawnDoor(world, at(20, 23), gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	raider, err := lifecycle.SpawnEnemy(world, at(20, 24), "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge
	si := query.GetSpatialIndex(world)
	for y := 21; y <= 30; y++ {
		si.Blocked[gc.GridElement{Coord: at(19, y)}] = true
		si.Blocked[gc.GridElement{Coord: at(21, y)}] = true
	}

	rp := newSoloPlanner(newTestRNG())
	act := rp.Plan(world, raider)

	require.NotNil(t, act)
	require.Equal(t, gc.BehaviorMelee, act.BehaviorName)
	p, ok := act.Params.(*gc.MeleeParams)
	require.True(t, ok)
	assert.Equal(t, door, p.Target)
}

func TestPlanRaidAction_追跡の状態でも相手が見えなければキューブへ戻る(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, at(20, 20))
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	raider, err := lifecycle.SpawnEnemy(world, at(20, 26), "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge
	world.Components.SoloAI.Get(raider).SubState = gc.AIStateChasing

	rp := newSoloPlanner(newTestRNG())
	act := rp.Plan(world, raider)

	require.NotNil(t, act)
	require.Equal(t, gc.BehaviorMove, act.BehaviorName)
	p, ok := act.Params.(*gc.MoveParams)
	require.True(t, ok)
	assert.Equal(t, at(20, 25), p.Destination.Coord)
}

func TestPlanRaidAction_細い通路で前に仲間がいれば待つ(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, at(20, 20))
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	_, err = lifecycle.SpawnEnemy(world, at(20, 23), "stray_dog")
	require.NoError(t, err)
	raider, err := lifecycle.SpawnEnemy(world, at(20, 24), "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge
	si := query.GetSpatialIndex(world)
	for y := 21; y <= 40; y++ {
		si.Blocked[gc.GridElement{Coord: at(19, y)}] = true
		si.Blocked[gc.GridElement{Coord: at(21, y)}] = true
	}

	rp := newSoloPlanner(newTestRNG())
	act := rp.Plan(world, raider)

	require.NotNil(t, act)
	assert.Equal(t, gc.BehaviorWait, act.BehaviorName, "回り道が長ければ、前の仲間を待つ")
}

func TestPlanRaidAction_開けた所では前の仲間の横へ回り込む(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, at(20, 20))
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	raider, err := lifecycle.SpawnEnemy(world, at(20, 24), "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge
	// 手番の初めに作った地図のあとで、先に動いた仲間が真ん前へ入った
	query.GetConvergeDistanceMap(world)
	front, err := lifecycle.SpawnEnemy(world, at(20, 23), "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(front).Movement = rawschema.Converge

	rp := newSoloPlanner(newTestRNG())
	act := rp.Plan(world, raider)

	require.NotNil(t, act)
	require.Equal(t, gc.BehaviorMove, act.BehaviorName)
	p, ok := act.Params.(*gc.MoveParams)
	require.True(t, ok)
	assert.Equal(t, at(19, 23), p.Destination.Coord)
}

func TestCullDistantSolo_襲撃の敵は圏外でも残す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "ash")
	require.NoError(t, err)
	far := consts.Coord[consts.Tile]{X: consts.Tile(activationRadius + 5), Y: 0}
	raider, err := lifecycle.SpawnEnemy(world, far, "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge
	ordinary, err := lifecycle.SpawnEnemy(world, far.Add(consts.Coord[consts.Tile]{Y: 1}), "stray_dog")
	require.NoError(t, err)

	kept, err := cullDistantSolo(world, []ecs.Entity{raider, ordinary})
	require.NoError(t, err)
	assert.Equal(t, []ecs.Entity{raider}, kept)
}

func TestProcessor_襲撃の敵がいる手番だけ距離の地図を作る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	playerAt := consts.Coord[consts.Tile]{X: 2, Y: 2}
	_, err := lifecycle.SpawnPlayer(world, playerAt, "ash")
	require.NoError(t, err)
	enemy, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: 30, Y: 30}, "stray_dog")
	require.NoError(t, err)
	m := query.GetConvergeDistanceMap(world)
	require.True(t, m.Built, "前の手番の地図")

	require.NoError(t, NewProcessor(newTestRNG()).ProcessAll(world))
	assert.False(t, m.Built, "普段の手番は前の地図を捨て、作らない")

	world.Components.SoloAI.Get(enemy).Movement = rawschema.Converge
	require.NoError(t, NewProcessor(newTestRNG()).ProcessAll(world))
	require.True(t, m.Built, "襲撃の敵が引いて作る")
	d, ok := m.Dist(playerAt)
	require.True(t, ok)
	assert.Equal(t, int32(0), d, "キューブが無ければプレイヤーを的にする")
}

func TestPlanConvergeAction_地図のあとで塞がれたマスは飛ばす(t *testing.T) {
	t.Parallel()
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, at(2, 2), "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, at(20, 20))
	require.NoError(t, err)
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	raider, err := lifecycle.SpawnEnemy(world, at(20, 26), "stray_dog")
	require.NoError(t, err)
	world.Components.SoloAI.Get(raider).Movement = rawschema.Converge
	query.GetConvergeDistanceMap(world)
	query.GetSpatialIndex(world).Blocked[gc.GridElement{Coord: at(20, 25)}] = true

	act := newSoloPlanner(newTestRNG()).Plan(world, raider)

	require.NotNil(t, act)
	require.Equal(t, gc.BehaviorMove, act.BehaviorName)
	p, ok := act.Params.(*gc.MoveParams)
	require.True(t, ok)
	assert.Equal(t, at(19, 25), p.Destination.Coord, "殴れる物の無い塞がれたマスは飛ばし、次の隣へ進む")
}
