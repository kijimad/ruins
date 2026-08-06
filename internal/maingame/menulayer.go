package maingame

import "github.com/hajimehoshi/ebiten/v2"

// menuLayer はメニュー群を1枚のオフスクリーンへ平坦化してから、世界へ一度だけ透過合成する層。
// 透明度を層合成の一度きりに集約する。これにより、メニューが重なっても世界の減衰が二重にかからず、
// 下のメニューが上のメニューを透けて見えることもない。パネルを層の中で不透明に描くほど、重なりは
// 上が下を上書きし、下メニューは隠れる。
type menuLayer struct {
	img   *ebiten.Image // 画面サイズのオフスクリーン。毎フレーム透明クリアして使い回す
	lastW int
	lastH int
}

// Begin は画面サイズ分のオフスクリーンを用意し、透明にクリアして返す。
// サイズが変わったら作り直す。
func (m *menuLayer) Begin(width, height int) *ebiten.Image {
	if m.img == nil || m.lastW != width || m.lastH != height {
		m.img = ebiten.NewImage(width, height)
		m.lastW = width
		m.lastH = height
	}
	m.img.Clear()

	return m.img
}

// Composite は層を dst へ大域アルファで一度だけ重ねる。透明度はこの一度の合成だけが持つ。
func (m *menuLayer) Composite(dst *ebiten.Image, alpha float64) {
	if m.img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(m.img, op)
}
