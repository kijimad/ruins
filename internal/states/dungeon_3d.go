package states

import (
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// dungeon3D はローポリ3D表示の状態と操作をまとめる。DungeonState から3D固有のものを切り出す。
//
// カメラの向き・見下ろし角・距離は ECS の Camera が持つ。世界を描く Render3DSystem と、
// その上へカーソルやエフェクトを重ねる側が同じ値を読むことで、投影先が一致する。
// ここに残すのはポインタ操作の途中経過だけで、フレームを跨いで意味を持たない。
type dungeon3D struct {
	// sys は描画システム。初回利用時に構築する
	sys *gs.Render3DSystem
	// dragging と lastCurY は右ドラッグでの見回しに使う
	dragging bool
	lastCurY int
}

// ensure は描画システムを遅延構築する。
func (d *dungeon3D) ensure() {
	if d.sys == nil {
		d.sys = gs.NewRender3DSystem()
	}
}

// update はカメラ操作のポインタ入力を処理する。右ドラッグで見回し、ホイールでズームする。
// 45度回転はキー由来なので束縛表の Action として DoAction から rotate へ届く
func (d *dungeon3D) update(world w.World) {
	d.ensure()
	_, cy := ebiten.CursorPosition()
	camera := query.GetPlayerCamera(world)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if d.dragging && camera != nil {
			camera.Pitch = min(max(camera.Pitch+float64(cy-d.lastCurY)*0.01, gc.CameraMinPitch), gc.CameraMaxPitch)
		}
		d.dragging = true
	} else {
		d.dragging = false
	}
	d.lastCurY = cy
	if _, wy := ebiten.Wheel(); wy != 0 && camera != nil {
		camera.Dist = min(max(camera.Dist-wy, gc.CameraMinDist), gc.CameraMaxDist)
	}
}

// rotate はカメラの向きを45度単位で回す。Orient を8状態で巡回させる。
func (d *dungeon3D) rotate(world w.World, delta int) {
	d.ensure()
	camera := query.GetPlayerCamera(world)
	if camera == nil {
		return
	}
	camera.Orient = camera.Orient.Rotated(delta)
}

// moveDir は押されたキーの画面意図を、カメラの向きで world の8方向へ回す。
func (d *dungeon3D) moveDir(world w.World, base gc.Direction) gc.Direction {
	var y float64
	if camera := query.GetPlayerCamera(world); camera != nil {
		y = camera.Yaw()
	}
	return gc.RotateScreenDir(base, y)
}

// draw は3Dシーンを screen へ描く。
func (d *dungeon3D) draw(world w.World, screen *ebiten.Image) error {
	d.ensure()
	return d.sys.Draw(world, screen)
}
