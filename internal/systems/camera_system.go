package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

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
	world.Manager.Join(
		world.Components.Player,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		playerGridElement = world.Components.GridElement.Get(entity).(*gc.GridElement)
	}))

	// カメラのズーム処理と追従処理
	world.Manager.Join(
		world.Components.Camera,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		camera := world.Components.Camera.Get(entity).(*gc.Camera)
		cameraGridElement := world.Components.GridElement.Get(entity).(*gc.GridElement)

		// カメラの追従（プレイヤー位置に更新）
		if playerGridElement != nil {
			cameraGridElement.X = playerGridElement.X
			cameraGridElement.Y = playerGridElement.Y
		}

		// ズーム率変更
		// 参考: https://ebitengine.org/ja/examples/isometric.html
		var scrollY float64
		if ebiten.IsKeyPressed(ebiten.KeyC) || ebiten.IsKeyPressed(ebiten.KeyPageDown) {
			scrollY = -0.25
		} else if ebiten.IsKeyPressed(ebiten.KeyE) || ebiten.IsKeyPressed(ebiten.KeyPageUp) {
			scrollY = 0.25
		} else {
			_, scrollY = ebiten.Wheel()
			if scrollY < -1 {
				scrollY = -1
			} else if scrollY > 1 {
				scrollY = 1
			}
		}
		camera.ScaleTo += scrollY * (camera.ScaleTo / 7)

		// Clamp target zoom level.
		if camera.ScaleTo < 0.8 {
			camera.ScaleTo = 0.8
		} else if camera.ScaleTo > 10 {
			camera.ScaleTo = 10
		}

		// Smooth zoom transition.
		cfg := config.MustGet()
		if cfg.DisableAnimation {
			// アニメーション無効時は即座にズーム
			camera.Scale = camera.ScaleTo
		} else {
			// 通常時はスムーズにズーム
			div := 10.0
			if camera.ScaleTo > camera.Scale {
				camera.Scale += (camera.ScaleTo - camera.Scale) / div
			} else if camera.ScaleTo < camera.Scale {
				camera.Scale -= (camera.Scale - camera.ScaleTo) / div
			}
		}
	}))
	return nil
}
