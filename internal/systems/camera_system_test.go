package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCameraSystem_SnapsToPlayerPosition(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// カメラの初期位置は原点
	cameraEntity := world.World.NewEntity()
	camera := &gc.Camera{
		Scale:   1.0,
		ScaleTo: 1.0,
	}
	world.Components.Camera.Add(cameraEntity, camera)

	sys := &CameraSystem{}
	require.NoError(t, sys.Update(world))

	tileSize := float64(consts.TileSize)
	expectedX := float64(10)*tileSize + tileSize/2
	expectedY := float64(10)*tileSize + tileSize/2

	assert.Equal(t, expectedX, camera.X, "1回のUpdateでカメラがプレイヤー位置にスナップする")
	assert.Equal(t, expectedY, camera.Y, "1回のUpdateでカメラがプレイヤー位置にスナップする")
}

func TestCameraSystem_NoPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	initialX := 100.0
	initialY := 200.0
	cameraEntity := world.World.NewEntity()
	camera := &gc.Camera{
		Scale:   1.0,
		ScaleTo: 1.0,
		X:       initialX,
		Y:       initialY,
		TargetX: initialX,
		TargetY: initialY,
	}
	world.Components.Camera.Add(cameraEntity, camera)

	sys := &CameraSystem{}
	require.NoError(t, sys.Update(world))

	assert.Equal(t, initialX, camera.X, "プレイヤーがいない場合、カメラ位置は変わらない")
	assert.Equal(t, initialY, camera.Y, "プレイヤーがいない場合、カメラ位置は変わらない")
}

func TestCameraSystem_FollowsPlayerMovement(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	cameraEntity := world.World.NewEntity()
	camera := &gc.Camera{Scale: 1.0, ScaleTo: 1.0}
	world.Components.Camera.Add(cameraEntity, camera)

	sys := &CameraSystem{}
	require.NoError(t, sys.Update(world))

	tileSize := float64(consts.TileSize)

	// プレイヤーを移動させる
	grid := world.Components.GridElement.Get(player)
	grid.X = 8
	grid.Y = 3

	require.NoError(t, sys.Update(world))

	expectedX := float64(8)*tileSize + tileSize/2
	expectedY := float64(3)*tileSize + tileSize/2
	assert.Equal(t, expectedX, camera.X, "移動後にカメラが即座に追従する")
	assert.Equal(t, expectedY, camera.Y, "移動後にカメラが即座に追従する")
}
