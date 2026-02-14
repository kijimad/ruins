package systems

import (
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// CameraFollowSpeed はカメラ追従の速度を制御する
// 値が大きいほど追従が速くなる。0.0〜1.0の範囲で、1.0なら即座に追従する
const CameraFollowSpeed = 0.15

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
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		camera := world.Components.Camera.Get(entity).(*gc.Camera)

		// プレイヤー位置をピクセル座標に変換してカメラの目標位置に設定
		if playerGridElement != nil {
			tileSize := float64(consts.TileSize)
			camera.TargetX = float64(playerGridElement.X)*tileSize + tileSize/2
			camera.TargetY = float64(playerGridElement.Y)*tileSize + tileSize/2
		}

		// カメラ位置を目標位置に向けて滑らかに補間
		camera.X += (camera.TargetX - camera.X) * CameraFollowSpeed
		camera.Y += (camera.TargetY - camera.Y) * CameraFollowSpeed

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

		// ズーム率の範囲制限
		if camera.ScaleTo < 0.8 {
			camera.ScaleTo = 0.8
		} else if camera.ScaleTo > 10 {
			camera.ScaleTo = 10
		}

		// ズームも滑らかに追従
		camera.Scale = camera.ScaleTo
	}))
	return nil
}
