package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// getCamera は単一のカメラコンポーネントを返す。存在しなければ nil。
// カメラはシングルトン的に1つだけ存在する前提で、各所のクエリ定型を集約する
func getCamera(world w.World) *gc.Camera {
	var camera *gc.Camera
	cameraQuery := ecs.NewFilter1[gc.Camera](world.ECS).Query()
	for cameraQuery.Next() {
		camera = world.Components.Camera.Get(cameraQuery.Entity())
	}
	return camera
}

// CameraSystem はカメラの追従とズーム処理を行う
type CameraSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys CameraSystem) String() string {
	return "CameraSystem"
}

// Update はカメラの追従とズーム処理を行う
// w.Updater interfaceを実装
func (sys *CameraSystem) Update(world w.World) error {
	var playerGridElement *gc.GridElement

	// プレイヤー位置を取得
	playerQuery := ecs.NewFilter2[gc.Player, gc.GridElement](world.ECS).Query()
	for playerQuery.Next() {
		entity := playerQuery.Entity()
		playerGridElement = world.Components.GridElement.Get(entity)
	}

	// カメラのズーム処理と追従処理
	camera := getCamera(world)
	if camera == nil {
		return nil
	}

	// プレイヤー位置をピクセル座標に変換してカメラの目標位置に設定
	if playerGridElement != nil {
		camera.Target = consts.TileCenterToWorld(playerGridElement.Coord)
	}

	// カメラ位置を目標位置に即座にスナップする
	camera.Pos = camera.Target

	// ズーム率変更
	// 参考: https://ebitengine.org/ja/examples/isometric.html
	var scrollY float64
	switch {
	case ebiten.IsKeyPressed(ebiten.KeyC) || ebiten.IsKeyPressed(ebiten.KeyPageDown):
		scrollY = -0.25
	case ebiten.IsKeyPressed(ebiten.KeyE) || ebiten.IsKeyPressed(ebiten.KeyPageUp):
		scrollY = 0.25
	default:
		_, scrollY = ebiten.Wheel()
		if scrollY < -1 {
			scrollY = -1
		} else if scrollY > 1 {
			scrollY = 1
		}
	}
	camera.ScaleTo += scrollY * (camera.ScaleTo / 7)

	// ズーム率の範囲制限
	if camera.ScaleTo < 0.8 {
		camera.ScaleTo = 0.8
	} else if camera.ScaleTo > 10 {
		camera.ScaleTo = 10
	}

	// ズームも滑らかに追従
	camera.Scale = camera.ScaleTo
	return nil
}
