package states

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/input"
	gs "github.com/kijimaD/ruins/internal/systems"
	w "github.com/kijimaD/ruins/internal/world"
)

// dungeon3D は実験的なローポリ3D表示の状態と操作をまとめる。DungeonState から3D固有の
// フィールドとロジックを切り出し、2Dの本流を汚さず、将来まるごと採否を判断しやすくする。
type dungeon3D struct {
	// enabled は3D描画への切り替え。F3 でトグルする
	enabled bool
	// sys は描画とカメラの状態。初回利用時に構築する
	sys *gs.Render3DSystem
	// orient はカメラと移動キーの向き。0-7 が45度刻み。Z/C で回す
	orient int
	// dragging と lastCurY は右ドラッグでの見回し（Pitch）に使う
	dragging bool
	lastCurY int
}

// toggle は3D描画の有効・無効を切り替える。
func (d *dungeon3D) toggle() { d.enabled = !d.enabled }

// ensure は描画システムを遅延構築する。
func (d *dungeon3D) ensure() {
	if d.sys == nil {
		d.sys = gs.NewRender3DSystem()
	}
}

// update はカメラ操作の入力を処理する。3D有効時のみ呼ぶ。
// Z/C で45度ずつ回し、右ドラッグで見回し、ホイールでズームする。
func (d *dungeon3D) update(kb input.KeyboardInput) {
	d.ensure()
	// 回転は英字キー Z/C。日本語キーボードでも位置が同じで安全。記号 [ ] は JIS でズレる
	if kb.IsKeyJustPressed(ebiten.KeyC) {
		d.rotate(1)
	}
	if kb.IsKeyJustPressed(ebiten.KeyZ) {
		d.rotate(-1)
	}
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
	d.orient = (d.orient + delta + 8) % 8
	if d.sys != nil {
		d.sys.Yaw = float64(d.orient) * (math.Pi / 4)
	}
}

// moveDir は移動方向をカメラの向きへ合わせる。押されたキーの画面意図を、カメラの向きで
// world ベクトルへ回し、最寄りの8方向へスナップする。画面奥は forward、画面右は right の
// 実カメラ値 forward=(-sin y, cos y) / right=(-cos y, -sin y) に一致させる。
func (d *dungeon3D) moveDir(base gc.Direction) gc.Direction {
	su, sr := base.ScreenIntent()
	y := float64(d.orient) * (math.Pi / 4)
	wx := -su*math.Sin(y) - sr*math.Cos(y)
	wy := su*math.Cos(y) - sr*math.Sin(y)
	return gc.SnapWorldVec(wx, wy)
}

// draw は3Dシーンと操作ヒントを screen へ描く。
func (d *dungeon3D) draw(world w.World, screen *ebiten.Image) error {
	d.ensure()
	if err := d.sys.Draw(world, screen); err != nil {
		return err
	}
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("3D orient=%d   Z/C rotate  wheel zoom  right-drag look  F3 back to 2D", d.orient), 8, 8)
	return nil
}
