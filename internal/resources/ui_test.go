package resources

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHexToColor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		hex  string
		want color.NRGBA
	}{
		{"白", "ffffff", color.NRGBA{R: 255, G: 255, B: 255, A: 255}},
		{"黒", "000000", color.NRGBA{R: 0, G: 0, B: 0, A: 255}},
		{"赤", "ff0000", color.NRGBA{R: 255, G: 0, B: 0, A: 255}},
		{"緑", "00ff00", color.NRGBA{R: 0, G: 255, B: 0, A: 255}},
		{"青", "0000ff", color.NRGBA{R: 0, G: 0, B: 255, A: 255}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := hexToColor(tc.hex)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHexToColor_不正な16進数文字列はpanicする(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		hexToColor("zzzzzz")
	})
}
