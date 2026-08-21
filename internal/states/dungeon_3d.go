package states

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
)

// dungeon3D はローポリ3D表示の状態と操作をまとめる。DungeonState から3D固有のものを切り出す。
type dungeon3D struct {
	// sys は描画とカメラの状態。初回利用時に構築する
	sys *gs.Render3DSystem
	// orient はカメラと移動キーの向き。0-7 が45度刻み。回転 Action で回す
	orient int
	// dragging と lastCurY は右ドラッグでの見回し（Pitch）に使う
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
func (d *dungeon3D) update() {
	d.ensure()
	_, cy := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if d.dragging {
			d.sys.Pitch = max(0.15, min(1.45, d.sys.Pitch+float64(cy-d.lastCurY)*0.01))
		}
		d.dragging = true
	} else {
		d.dragging = false
	}
	d.lastCurY = cy
	if _, wy := ebiten.Wheel(); wy != 0 {
		d.sys.Dist = max(3, min(25, d.sys.Dist-wy))
	}
}

// rotate はカメラの向きを45度単位で回す。orient を8状態で巡回し、カメラ Yaw を同期する。
func (d *dungeon3D) rotate(delta int) {
	d.ensure()
	d.orient = (d.orient + delta + 8) % 8
	d.sys.Yaw = float64(d.orient) * (math.Pi / 4)
}

// moveDir は押されたキーの画面意図を、カメラの向きで world ベクトルへ回し最寄りの8方向へスナップする。
func (d *dungeon3D) moveDir(base gc.Direction) gc.Direction {
	su, sr := base.ScreenIntent()
	y := float64(d.orient) * (math.Pi / 4)
	// 南から北を見下ろすカメラに合わせる。画面奥 forward=(-sin y, -cos y)、画面右 right=(cos y, -sin y)、
	// world = su*forward + sr*right
	wx := -su*math.Sin(y) + sr*math.Cos(y)
	wy := -su*math.Cos(y) - sr*math.Sin(y)
	return gc.SnapWorldVec(wx, wy)
}

// draw は3Dシーンを screen へ描く。
func (d *dungeon3D) draw(world w.World, screen *ebiten.Image) error {
	d.ensure()
	return d.sys.Draw(world, screen)
}
