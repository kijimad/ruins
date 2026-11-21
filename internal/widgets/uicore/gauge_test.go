package uicore_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

func TestGauge_Draw(t *testing.T) {
	t.Parallel()
	fill := ebiten.NewImage(1, 4)
	tests := []struct {
		name  string
		ratio float64
		width int
		want  []image.Rectangle
	}{
		{name: "比率ぶんを塗る", ratio: 0.5, width: 20, want: []image.Rectangle{image.Rect(1, 1, 10, 5)}},
		{name: "残りわずかでも1pxは塗る", ratio: 0.01, width: 20, want: []image.Rectangle{image.Rect(1, 1, 2, 5)}},
		{name: "1を超える比率は枠に収める", ratio: 1.5, width: 20, want: []image.Rectangle{image.Rect(1, 1, 19, 5)}},
		{name: "0は塗らない", ratio: 0, width: 20, want: nil},
		{name: "負の比率は塗らない", ratio: -0.5, width: 20, want: nil},
		{name: "枠の内側に幅が無ければ塗らない", ratio: 0.5, width: 2, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := &uicore.Gauge{Fill: fill, Ratio: tt.ratio, Radius: 1}
			g.Layout(image.Rect(0, 0, tt.width, 6))
			cv := &recordCanvas{}
			g.Draw(cv)
			assert.Equal(t, tt.want, cv.tinted)
			assert.Equal(t, []image.Rectangle{image.Rect(0, 0, tt.width, 6)}, cv.roundedStrokes)
		})
	}
}

func TestFitScale(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dst  image.Rectangle
		size image.Point
		want float64
	}{
		{name: "収まる画像は等倍", dst: image.Rect(0, 0, 20, 21), size: image.Pt(20, 20), want: 1},
		{name: "高さが足りなければ高さに合わせて縮める", dst: image.Rect(0, 0, 40, 10), size: image.Pt(20, 20), want: 0.5},
		{name: "幅が足りなければ幅に合わせて縮める", dst: image.Rect(0, 0, 5, 40), size: image.Pt(20, 20), want: 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.InDelta(t, tt.want, uicore.FitScale(tt.dst, tt.size), 1e-9)
		})
	}
}
