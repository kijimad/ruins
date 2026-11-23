package states_test

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

// phaseStripGap は並べた段階の画像の間の幅
const phaseStripGap = 8

// phaseStrip は段階ごとの画像 frames を、それぞれ背景と違う所に切り詰めて 1/shrink に縮め、左から横一列に並べた PNG にする。
// 背景は各画像の右下の画素の色とみなす
func phaseStrip(t *testing.T, frames [][]byte, shrink int) []byte {
	t.Helper()
	imgs := make([]image.Image, 0, len(frames))
	width, height := phaseStripGap, 0
	for _, f := range frames {
		src, err := png.Decode(bytes.NewReader(f))
		require.NoError(t, err)
		img := shrinkNearest(contentOf(src), shrink)
		imgs = append(imgs, img)
		width += img.Bounds().Dx() + phaseStripGap
		height = max(height, img.Bounds().Dy())
	}
	out := image.NewNRGBA(image.Rect(0, 0, width, height+2*phaseStripGap))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.NRGBA{R: 32, G: 32, B: 36, A: 255}), image.Point{}, draw.Src)
	x := phaseStripGap
	for _, img := range imgs {
		b := img.Bounds()
		draw.Draw(out, image.Rect(x, phaseStripGap, x+b.Dx(), phaseStripGap+b.Dy()), img, b.Min, draw.Src)
		x += b.Dx() + phaseStripGap
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, out))
	return buf.Bytes()
}

// contentOf は img のうち、右下の画素と違う色の画素を全て含む最小の矩形を切り出す。全て同じ色なら img をそのまま返す
func contentOf(img image.Image) image.Image {
	b := img.Bounds()
	bg := color.NRGBAModel.Convert(img.At(b.Max.X-1, b.Max.Y-1))
	box := image.Rectangle{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if color.NRGBAModel.Convert(img.At(x, y)) != bg {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if box.Empty() {
		return img
	}
	out := image.NewNRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
	draw.Draw(out, out.Bounds(), img, box.Min, draw.Src)
	return out
}

// shrinkNearest は img を 1/n に、近い画素を拾って縮める。n が1以下なら img をそのまま返す
func shrinkNearest(img image.Image, n int) image.Image {
	if n <= 1 {
		return img
	}
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()/n, b.Dy()/n))
	for y := range out.Bounds().Dy() {
		for x := range out.Bounds().Dx() {
			out.Set(x, y, img.At(b.Min.X+x*n, b.Min.Y+y*n))
		}
	}
	return out
}
