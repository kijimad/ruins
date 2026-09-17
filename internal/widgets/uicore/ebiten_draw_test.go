package uicore_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
	"github.com/stretchr/testify/assert"
)

// pixelAlpha は screen 上の1点のアルファ値を返す。
func pixelAlpha(screen *ebiten.Image, x, y int) byte {
	b := screen.Bounds()
	pix := make([]byte, b.Dx()*b.Dy()*4)
	screen.ReadPixels(pix)
	i := ((y-b.Min.Y)*b.Dx() + (x - b.Min.X)) * 4
	return pix[i+3]
}

func TestEbitenCanvas_FillTriangle_三角形の内側が塗られ外側は塗られない(t *testing.T) {
	t.Parallel()
	screen := ebiten.NewImage(50, 50)
	cv := uicore.NewEbitenCanvas(screen)

	cv.FillTriangle([2]float32{5, 5}, [2]float32{45, 5}, [2]float32{25, 45}, color.White)

	assert.Positive(t, countOpaque(screen), "三角形の内部が塗られて不透明画素が出る")
	assert.Zero(t, pixelAlpha(screen, 0, 0), "三角形の外側は塗られない")
}

func TestEbitenCanvas_DrawImage_posを左上として描く(t *testing.T) {
	t.Parallel()
	src := ebiten.NewImage(10, 10)
	src.Fill(color.White)
	screen := ebiten.NewImage(50, 50)
	cv := uicore.NewEbitenCanvas(screen)

	cv.DrawImage(image.Pt(20, 20), src)

	assert.Positive(t, pixelAlpha(screen, 25, 25), "指定位置に画像が描かれる")
	assert.Zero(t, pixelAlpha(screen, 5, 5), "指定位置より外は描かれない")
}

func TestEbitenCanvas_DrawImageRect(t *testing.T) {
	t.Parallel()

	t.Run("nilなら何もしない", func(t *testing.T) {
		t.Parallel()
		screen := ebiten.NewImage(50, 50)
		cv := uicore.NewEbitenCanvas(screen)

		cv.DrawImageRect(image.Rect(0, 0, 50, 50), nil)

		assert.Zero(t, countOpaque(screen))
	})

	t.Run("矩形の幅が0なら何もしない", func(t *testing.T) {
		t.Parallel()
		src := ebiten.NewImage(10, 10)
		src.Fill(color.White)
		screen := ebiten.NewImage(50, 50)
		cv := uicore.NewEbitenCanvas(screen)

		cv.DrawImageRect(image.Rect(10, 10, 10, 40), src)

		assert.Zero(t, countOpaque(screen))
	})

	t.Run("収まる画像は拡大せず等倍で左寄せ縦中央に置く", func(t *testing.T) {
		t.Parallel()
		src := ebiten.NewImage(10, 10)
		src.Fill(color.White)
		screen := ebiten.NewImage(50, 50)
		cv := uicore.NewEbitenCanvas(screen)

		// dst(40x40) は画像(10x10)より十分大きいので scale は1に頭打ちになり、等倍で置かれる。
		// 左寄せ・縦中央なので oy=(40-10)/2=15
		cv.DrawImageRect(image.Rect(0, 0, 40, 40), src)

		assert.Positive(t, pixelAlpha(screen, 5, 15+5), "左寄せ・縦中央の位置に等倍で描かれる")
		assert.Zero(t, pixelAlpha(screen, 5, 35), "拡大されないので dst 下部は空のまま")
	})

	t.Run("収まらない画像は縦横比を保って縮小し左寄せ縦中央に置く", func(t *testing.T) {
		t.Parallel()
		src := ebiten.NewImage(100, 50) // 横長
		src.Fill(color.White)
		screen := ebiten.NewImage(60, 60)
		cv := uicore.NewEbitenCanvas(screen)

		// scale = min(40/100, 40/50, 1) = 0.4。dh=50*0.4=20、oy=(40-20)/2=10
		cv.DrawImageRect(image.Rect(0, 0, 40, 40), src)

		assert.Positive(t, pixelAlpha(screen, 5, 10+5), "縮小された位置に描かれる")
		assert.Zero(t, pixelAlpha(screen, 5, 45), "dst の外へはみ出さない")
	})
}

func TestEbitenCanvas_DrawImageTintedRect(t *testing.T) {
	t.Parallel()

	t.Run("nilなら何もしない", func(t *testing.T) {
		t.Parallel()
		screen := ebiten.NewImage(50, 50)
		cv := uicore.NewEbitenCanvas(screen)

		cv.DrawImageTintedRect(image.Rect(0, 0, 50, 50), nil, color.White)

		assert.Zero(t, countOpaque(screen))
	})

	t.Run("矩形の幅が0なら何もしない", func(t *testing.T) {
		t.Parallel()
		src := ebiten.NewImage(4, 4)
		src.Fill(color.White)
		screen := ebiten.NewImage(50, 50)
		cv := uicore.NewEbitenCanvas(screen)

		cv.DrawImageTintedRect(image.Rect(10, 10, 10, 40), src, color.White)

		assert.Zero(t, countOpaque(screen))
	})

	t.Run("dstいっぱいに引き伸ばして塗る", func(t *testing.T) {
		t.Parallel()
		src := ebiten.NewImage(2, 2)
		src.Fill(color.White)
		screen := ebiten.NewImage(50, 50)
		cv := uicore.NewEbitenCanvas(screen)

		cv.DrawImageTintedRect(image.Rect(5, 5, 45, 45), src, color.NRGBA{R: 255, A: 255})

		assert.Positive(t, pixelAlpha(screen, 10, 10), "dst の内側は塗られる")
		assert.Positive(t, pixelAlpha(screen, 40, 40), "dst の隅近くまで引き伸ばされる")
		assert.Zero(t, pixelAlpha(screen, 2, 2), "dst の外は塗られない")
	})
}

func TestEbitenCanvas_DrawNineSlice(t *testing.T) {
	t.Parallel()

	t.Run("nilなら何もしない", func(t *testing.T) {
		t.Parallel()
		screen := ebiten.NewImage(50, 50)
		cv := uicore.NewEbitenCanvas(screen)

		cv.DrawNineSlice(image.Rect(0, 0, 50, 50), nil, [3]int{2, 2, 2}, [3]int{2, 2, 2})

		assert.Zero(t, countOpaque(screen))
	})

	t.Run("9セルへ分割してdstいっぱいに描く", func(t *testing.T) {
		t.Parallel()
		src := ebiten.NewImage(9, 9)
		src.Fill(color.White)
		screen := ebiten.NewImage(60, 60)
		cv := uicore.NewEbitenCanvas(screen)

		cv.DrawNineSlice(image.Rect(5, 5, 55, 55), src, [3]int{3, 3, 3}, [3]int{3, 3, 3})

		assert.Positive(t, pixelAlpha(screen, 6, 6), "四隅が原寸で描かれる")
		assert.Positive(t, pixelAlpha(screen, 30, 30), "中央も引き伸ばされて塗られる")
		assert.Zero(t, pixelAlpha(screen, 1, 1), "dst の外は塗られない")
	})

	t.Run("境界幅が0のスライスは描かずスキップする", func(t *testing.T) {
		t.Parallel()
		src := ebiten.NewImage(9, 9)
		src.Fill(color.White)
		screen := ebiten.NewImage(60, 60)
		cv := uicore.NewEbitenCanvas(screen)

		// 左右の境界幅を0にすると左右列のソース・dst幅が0になりスキップされる
		assert.NotPanics(t, func() {
			cv.DrawNineSlice(image.Rect(5, 5, 55, 55), src, [3]int{0, 9, 0}, [3]int{0, 9, 0})
		})
		assert.Positive(t, countOpaque(screen), "中央セルは描かれる")
	})
}
