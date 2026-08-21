package states

import (
	"math"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCameraWorld はカメラを持つプレイヤーが1人いるワールドを返す。
// カメラの向きは ECS が持つので、視点を触るテストは本番の spawn でカメラごと用意する。
func newCameraWorld(t *testing.T) w.World {
	t.Helper()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	return world
}

func TestDungeon3D_rotate(t *testing.T) {
	t.Parallel()

	world := newCameraWorld(t)
	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	d := &dungeon3D{}

	d.rotate(world, 1)
	assert.Equal(t, gc.Orient(1), camera.Orient)
	assert.InDelta(t, math.Pi/4, camera.Yaw(), 1e-9)

	d.rotate(world, -1)
	assert.Equal(t, gc.Orient(0), camera.Orient)

	// 反時計回りは環で7へ回り込む
	d.rotate(world, -1)
	assert.Equal(t, gc.Orient(7), camera.Orient)
	assert.InDelta(t, 7*math.Pi/4, camera.Yaw(), 1e-9)

	// 8回転で一巡して元へ戻る
	for range 8 {
		d.rotate(world, 1)
	}
	assert.Equal(t, gc.Orient(7), camera.Orient)
}

func TestDungeon3D_moveDir(t *testing.T) {
	t.Parallel()

	// orient0: 既定カメラは南から北を見下ろす。画面の上=北で、北上のミニマップと向きが一致する。
	// Up=北・Right=東・Left=西と、キーと地図の向きがそろうことを固定する
	world := newCameraWorld(t)
	camera := query.GetPlayerCamera(world)
	require.NotNil(t, camera)
	d := &dungeon3D{}

	assert.Equal(t, gc.DirectionUp, d.moveDir(world, gc.DirectionUp))
	assert.Equal(t, gc.DirectionRight, d.moveDir(world, gc.DirectionRight))
	assert.Equal(t, gc.DirectionLeft, d.moveDir(world, gc.DirectionLeft))

	// orient2 は90度。カメラが回ると Up は西へ回る
	camera.Orient = 2
	assert.Equal(t, gc.DirectionLeft, d.moveDir(world, gc.DirectionUp))
}

func TestDungeonState_moveDir_delegation(t *testing.T) {
	t.Parallel()

	// 常に3Dカメラの向きへ dungeon3D で委譲する。北上カメラの既定 orient0 では Up=北・Right=東のまま
	world := newCameraWorld(t)
	st := &DungeonState{}
	assert.Equal(t, gc.DirectionUp, st.moveDir(world, gc.DirectionUp))
	assert.Equal(t, gc.DirectionRight, st.moveDir(world, gc.DirectionRight))
}
