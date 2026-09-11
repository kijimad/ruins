package uicore

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
)

// TestStretchNeedsBlend は、伸縮時に軸を混ぜるべきかを返す純関数の分岐を固定する。
// 等倍または1テクセルは混ぜず、非等倍かつ2テクセル以上だけ混ぜる。
func TestStretchNeedsBlend(t *testing.T) {
	t.Parallel()

	assert.False(t, stretchNeedsBlend(1, 5), "等倍は混ぜない")
	assert.True(t, stretchNeedsBlend(2, 5), "非等倍かつ2テクセル以上は混ぜる")
	assert.False(t, stretchNeedsBlend(2, 1), "1テクセルは混ぜない")
	assert.False(t, stretchNeedsBlend(1, 1), "等倍かつ1テクセルは混ぜない")
}

// TestEbitenCanvas_画像描画がpanicしない は、EbitenCanvas の各画像描画メソッドが
// 等倍・縮小・引き伸ばし・nil・9スライスの各経路で panic せず通ることを固定する。
func TestEbitenCanvas_画像描画がpanicしない(t *testing.T) {
	t.Parallel()

	cv := NewEbitenCanvas(ebiten.NewImage(200, 200))
	img := ebiten.NewImage(16, 16)
	big := ebiten.NewImage(64, 64)

	assert.NotPanics(t, func() {
		cv.DrawImage(image.Pt(10, 10), img)

		cv.DrawImageRect(image.Rect(0, 0, 8, 8), big)   // 収まらず縮小する linear 経路
		cv.DrawImageRect(image.Rect(0, 0, 32, 32), img) // 収まるので等倍経路
		cv.DrawImageRect(image.Rect(0, 0, 8, 8), nil)   // nil は早期に戻る

		cv.DrawImageTintedRect(image.Rect(0, 0, 32, 32), img, color.White) // 引き伸ばして混ぜる経路
		cv.DrawImageTintedRect(image.Rect(0, 0, 16, 16), img, color.White) // 等倍で混ぜない経路
		cv.DrawImageTintedRect(image.Rect(0, 0, 8, 8), nil, color.White)   // nil は早期に戻る

		cv.DrawNineSlice(image.Rect(0, 0, 48, 48), img, [3]int{4, 8, 4}, [3]int{4, 8, 4})
		cv.DrawNineSlice(image.Rect(0, 0, 48, 48), nil, [3]int{4, 8, 4}, [3]int{4, 8, 4}) // nil は早期に戻る
	})
}
