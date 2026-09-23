package uicore

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

// clampRadius は非公開だが角丸パスが壊れないための要なので白箱で検証する。

func TestClampRadius(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name         string
		radius, w, h int
		want         float32
	}{
		{"半径が短辺の半分以下ならそのまま", 7, 100, 50, 7},
		{"半径が大きすぎたら短辺の半分へ", 100, 10, 10, 5},
		{"幅が短辺なら幅の半分で頭打ち", 10, 8, 30, 4},
		{"高さが短辺なら高さの半分で頭打ち", 10, 30, 6, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, clampRadius(tc.radius, tc.w, tc.h))
		})
	}
}

// roundedFillShape は焼き済み画像をキャッシュから使い回す。命中すれば同じポインタ、キーが違えば
// 別画像を返すことを確かめる。キャッシュが常にミスすると性能が落ちるが golden では気づけない。
func TestRoundedFillShape_同じキーは焼き済み画像を使い回す(t *testing.T) {
	t.Parallel()
	// 他テストと衝突しない一意な寸法を使う
	c := color.RGBA{R: 1, G: 2, B: 3, A: 255}
	a := roundedFillShape(13, 17, 3, c)
	assert.Same(t, a, roundedFillShape(13, 17, 3, c), "同じ寸法・半径・色は同じ画像を返す")
	assert.NotSame(t, a, roundedFillShape(13, 17, 5, c), "半径が違えば別の画像")
	assert.NotSame(t, a, roundedFillShape(13, 17, 3, color.RGBA{R: 9, G: 9, B: 9, A: 255}), "色が違えば別の画像")
}
