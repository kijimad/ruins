package screeneffect

import "github.com/hajimehoshi/ebiten/v2"

// AlphaLayer は複数の描画を1枚のオフスクリーンへ平坦化してから、描画先へ大域アルファで一度だけ
// 合成する層。透明度を最後の一度の合成に集約する。層の中で重ねた描画は不透明に上書きし合い、
// 合成先に対する透過は重ねた枚数によらず一定になる。
type AlphaLayer struct {
	img   *ebiten.Image // 描画先と同じサイズのオフスクリーン。毎フレーム透明クリアして使い回す
	lastW int
	lastH int
}

// Begin は幅高さ分のオフスクリーンを用意し、透明にクリアして返す。サイズが変わったら作り直す。
func (l *AlphaLayer) Begin(width, height int) *ebiten.Image {
	if l.img == nil || l.lastW != width || l.lastH != height {
		l.img = ebiten.NewImage(width, height)
		l.lastW = width
		l.lastH = height
	}
	l.img.Clear()

	return l.img
}

// Composite は層を dst へ大域アルファで一度だけ重ねる。透明度はこの一度の合成だけが持つ。
func (l *AlphaLayer) Composite(dst *ebiten.Image, alpha float64) {
	if l.img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(l.img, op)
}
