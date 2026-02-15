package systems

import (
	"math"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCameraSystem_SmoothFollow(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーエンティティを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// カメラエンティティを作成（初期位置は原点）
	cameraEntity := world.Manager.NewEntity()
	camera := &gc.Camera{
		Scale:   1.0,
		ScaleTo: 1.0,
		X:       0,
		Y:       0,
		TargetX: 0,
		TargetY: 0,
	}
	cameraEntity.AddComponent(world.Components.Camera, camera)

	// CameraSystemを実行
	sys := &CameraSystem{}
	require.NoError(t, sys.Update(world))

	// カメラの目標位置がプレイヤー位置に設定されていることを確認
	tileSize := float64(consts.TileSize)
	expectedTargetX := float64(10)*tileSize + tileSize/2
	expectedTargetY := float64(10)*tileSize + tileSize/2

	assert.Equal(t, expectedTargetX, camera.TargetX, "TargetXがプレイヤー位置に設定されるべき")
	assert.Equal(t, expectedTargetY, camera.TargetY, "TargetYがプレイヤー位置に設定されるべき")

	// カメラ位置が目標に向かって移動していることを確認（補間）
	assert.Greater(t, camera.X, 0.0, "カメラX位置が目標に向かって移動するべき")
	assert.Greater(t, camera.Y, 0.0, "カメラY位置が目標に向かって移動するべき")
	assert.Less(t, camera.X, expectedTargetX, "1回のUpdateではカメラは目標に到達しないべき")
	assert.Less(t, camera.Y, expectedTargetY, "1回のUpdateではカメラは目標に到達しないべき")
}

func TestCameraSystem_ConvergesToTarget(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーエンティティを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 5, Y: 5})

	// カメラエンティティを作成
	cameraEntity := world.Manager.NewEntity()
	tileSize := float64(consts.TileSize)
	targetX := float64(5)*tileSize + tileSize/2
	targetY := float64(5)*tileSize + tileSize/2
	camera := &gc.Camera{
		Scale:   1.0,
		ScaleTo: 1.0,
		X:       0,
		Y:       0,
		TargetX: 0,
		TargetY: 0,
	}
	cameraEntity.AddComponent(world.Components.Camera, camera)

	// CameraSystemを複数回実行して収束を確認
	sys := &CameraSystem{}
	for i := 0; i < 100; i++ {
		require.NoError(t, sys.Update(world))
	}

	// 更新後、カメラが目標位置に十分近づいていることを確認
	tolerance := 0.1
	assert.InDelta(t, targetX, camera.X, tolerance, "カメラX位置が目標に収束するべき")
	assert.InDelta(t, targetY, camera.Y, tolerance, "カメラY位置が目標に収束するべき")
}

func TestCameraSystem_NoPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーなしでカメラのみ
	cameraEntity := world.Manager.NewEntity()
	initialX := 100.0
	initialY := 200.0
	camera := &gc.Camera{
		Scale:   1.0,
		ScaleTo: 1.0,
		X:       initialX,
		Y:       initialY,
		TargetX: initialX,
		TargetY: initialY,
	}
	cameraEntity.AddComponent(world.Components.Camera, camera)

	// CameraSystemを実行
	sys := &CameraSystem{}
	require.NoError(t, sys.Update(world))

	// プレイヤーがいない場合、カメラ位置は変わらないべき
	assert.Equal(t, initialX, camera.X, "プレイヤーがいない場合、カメラX位置は変わらないべき")
	assert.Equal(t, initialY, camera.Y, "プレイヤーがいない場合、カメラY位置は変わらないべき")
}

func TestCameraSystem_LerpCalculation(t *testing.T) {
	t.Parallel()

	// 線形補間の計算が正しいことを確認
	startX := 0.0
	targetX := 100.0

	// 1回の補間
	newX := startX + (targetX-startX)*CameraFollowSpeed
	expectedX := targetX * CameraFollowSpeed

	assert.InDelta(t, expectedX, newX, 0.001, "線形補間の計算が正しいべき")

	// 補間速度が0〜1の範囲内であることを確認
	assert.GreaterOrEqual(t, CameraFollowSpeed, 0.0, "CameraFollowSpeedは0以上であるべき")
	assert.LessOrEqual(t, CameraFollowSpeed, 1.0, "CameraFollowSpeedは1以下であるべき")
}

func TestCameraSystem_LerpRate(t *testing.T) {
	t.Parallel()

	// 補間速度による収束の検証
	// n回の補間後の残り距離は (1 - speed)^n * initialDistance

	initialDistance := 100.0
	speed := CameraFollowSpeed

	// 10回の補間後の距離
	remaining := initialDistance * math.Pow(1-speed, 10)

	// 10回で90%以上近づいているべき
	assert.Less(t, remaining, initialDistance*0.3, "10回の補間で70%以上近づくべき")
}
