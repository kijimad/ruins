package cmd

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCutCells_透明なマスを除いて行と列の順に返す(t *testing.T) {
	t.Parallel()

	// 3列2行に余り5ピクセルを足した画像。(0,1) と (1,0) だけに画素を置く
	img := image.NewNRGBA(image.Rect(0, 0, 3*4+5, 2*4+5))
	red := color.NRGBA{R: 255, A: 255}
	img.Set(4+1, 0, red)
	img.Set(0, 4+2, red)

	got := cutCells(img, 4)

	require.Len(t, got, 2)
	assert.Equal(t, [2]int{0, 1}, [2]int{got[0].row, got[0].col})
	assert.Equal(t, [2]int{1, 0}, [2]int{got[1].row, got[1].col})
	assert.Equal(t, red, got[0].img.NRGBAAt(1, 0))
	assert.Equal(t, image.Rect(0, 0, 4, 4), got[0].img.Bounds())
}
