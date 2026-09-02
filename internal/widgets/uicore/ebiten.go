package uicore

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// EbitenCanvas は Canvas を ebiten の描画で満たす本番実装。
type EbitenCanvas struct {
	screen *ebiten.Image
}

var _ Canvas = (*EbitenCanvas)(nil)

// NewEbitenCanvas は描画先スクリーンを与えて Canvas を作る。
func NewEbitenCanvas(screen *ebiten.Image) *EbitenCanvas {
	return &EbitenCanvas{screen: screen}
}

// FillRect は EbitenCanvas を実装する。
func (e *EbitenCanvas) FillRect(r image.Rectangle, c color.Color) {
	vector.FillRect(e.screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c, false)
}

// StrokeRect は EbitenCanvas を実装する。
func (e *EbitenCanvas) StrokeRect(r image.Rectangle, width int, c color.Color) {
	vector.StrokeRect(e.screen, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), float32(width), c, false)
}

// DrawText は EbitenCanvas を実装する。pos を左上として1行を描く。
func (e *EbitenCanvas) DrawText(pos image.Point, s string, face text.Face, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	op.ColorScale.ScaleWithColor(c)
	text.Draw(e.screen, s, face, op)
}

// DrawImage は EbitenCanvas を実装する。pos を左上として画像を描く。
func (e *EbitenCanvas) DrawImage(pos image.Point, img *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(pos.X), float64(pos.Y))
	e.screen.DrawImage(img, op)
}

// DrawImageRect は EbitenCanvas を実装する。dst に収まるよう縦横比を保って縮小し、左寄せ・縦中央で描く。
func (e *EbitenCanvas) DrawImageRect(dst image.Rectangle, img *ebiten.Image) {
	if img == nil {
		return
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw <= 0 || ih <= 0 || dst.Dx() <= 0 || dst.Dy() <= 0 {
		return
	}
	scale := math.Min(math.Min(float64(dst.Dx())/float64(iw), float64(dst.Dy())/float64(ih)), 1)
	dh := float64(ih) * scale
	// 左寄せ・縦中央。アイコンとキーキャップを列の左に揃え、行の高さの中央へ置く
	ox := float64(dst.Min.X)
	oy := float64(dst.Min.Y) + (float64(dst.Dy())-dh)/2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(ox, oy)
	// 収まりきらない画像だけがここで縮む。端数倍率の nearest はテクセル境界の画素が
	// 揺れるので linear にする。一覧のアイコンは表示サイズで渡ってくるため等倍で通り、
	// この分岐には入らない
	if scale != 1 {
		op.Filter = ebiten.FilterLinear
	}
	e.screen.DrawImage(img, op)
}

// DrawImageTintedRect は EbitenCanvas を実装する。img を dst いっぱいに引き伸ばし tint を掛けて描く。
func (e *EbitenCanvas) DrawImageTintedRect(dst image.Rectangle, img *ebiten.Image, tint color.Color) {
	if img == nil {
		return
	}
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 || dst.Dx() <= 0 || dst.Dy() <= 0 {
		return
	}
	sx, sy := float64(dst.Dx())/float64(b.Dx()), float64(dst.Dy())/float64(b.Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(float64(dst.Min.X), float64(dst.Min.Y))
	op.ColorScale.ScaleWithColor(tint)
	// 伸ばす軸がどれも2テクセル以上あるときだけ linear にする。判定の理由は stretchNeedsBlend を参照
	if stretchNeedsBlend(sx, b.Dx()) && stretchNeedsBlend(sy, b.Dy()) {
		op.Filter = ebiten.FilterLinear
	}
	e.screen.DrawImage(img, op)
}

// stretchNeedsBlend はその軸を混ぜるべきかを返す。端数倍率の nearest はテクセル境界の
// 画素が揺れるので、混ぜられる軸は linear で混ぜたい。等倍なら境界をまたがず混ぜる必要が無い。
// 1テクセルしか無い軸は混ぜる相手が画像の外側の透明しかなく、端がぼやけるだけなので混ぜない。
func stretchNeedsBlend(scale float64, srcLen int) bool {
	return scale != 1 && srcLen > 1
}

// DrawNineSlice は EbitenCanvas を実装する。ソースを縦横3分割し、四隅は原寸、辺は片方向、
// 中央は両方向へ伸ばして9セルを dst へ描く。
func (e *EbitenCanvas) DrawNineSlice(dst image.Rectangle, img *ebiten.Image, bx, by [3]int) {
	if img == nil {
		return
	}
	// ソースの分割境界
	sx := [4]int{0, bx[0], bx[0] + bx[1], bx[0] + bx[1] + bx[2]}
	sy := [4]int{0, by[0], by[0] + by[1], by[0] + by[1] + by[2]}
	// dst の分割境界。四隅は原寸、中央は残りを埋める
	midW := max(dst.Dx()-bx[0]-bx[2], 0)
	midH := max(dst.Dy()-by[0]-by[2], 0)
	dx := [4]int{dst.Min.X, dst.Min.X + bx[0], dst.Min.X + bx[0] + midW, dst.Max.X}
	dy := [4]int{dst.Min.Y, dst.Min.Y + by[0], dst.Min.Y + by[0] + midH, dst.Max.Y}

	for r := range 3 {
		for c := range 3 {
			sw, sh := sx[c+1]-sx[c], sy[r+1]-sy[r]
			dw, dh := dx[c+1]-dx[c], dy[r+1]-dy[r]
			if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
				continue
			}
			sub, ok := img.SubImage(image.Rect(sx[c], sy[r], sx[c+1], sy[r+1])).(*ebiten.Image)
			if !ok {
				continue
			}
			ssx, ssy := float64(dw)/float64(sw), float64(dh)/float64(sh)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(ssx, ssy)
			op.GeoM.Translate(float64(dx[c]), float64(dy[r]))
			// 四隅は原寸で通り、伸びるのは辺と中央だけ。端数倍率の nearest は
			// テクセル境界の画素が揺れるので linear にする
			if ssx != 1 || ssy != 1 {
				op.Filter = ebiten.FilterLinear
			}
			e.screen.DrawImage(sub, op)
		}
	}
}
