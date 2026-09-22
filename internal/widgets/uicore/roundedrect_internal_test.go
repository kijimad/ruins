package uicore

import (
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
