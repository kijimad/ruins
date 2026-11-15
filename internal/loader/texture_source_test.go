package loader

import (
	"encoding/json"
	"image"
	"image/png"
	"path"
	"testing"

	"github.com/kijimaD/ruins/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextureSheets_束の各枠が元の絵と同じ は、make aseprite が束ねた絵の各枠が、同じ名前の元の絵と
// 画素まで一致することを確かめる。束の作り直し忘れや、色違いの絵を1枚にまとめてしまう束ね方を止める
func TestTextureSheets_束の各枠が元の絵と同じ(t *testing.T) {
	t.Parallel()

	for _, sheet := range []string{"tiles", "single", "bg"} {
		t.Run(sheet, func(t *testing.T) {
			t.Parallel()
			atlas := decodePNG(t, path.Join("file/textures/dist", sheet+".png"))
			b, err := assets.FS.ReadFile(path.Join("file/textures/dist", sheet+".json"))
			require.NoError(t, err)
			var data AsepriteJSON
			require.NoError(t, json.Unmarshal(b, &data))
			require.NotEmpty(t, data.Frames, "枠が無いと検査が素通りする")

			var differ []string
			for _, fr := range data.Frames {
				src := decodePNG(t, path.Join("file/textures", sheet, fr.Filename+".png"))
				if !sameFrame(atlas, fr.Frame, src) {
					differ = append(differ, fr.Filename)
				}
			}
			assert.Empty(t, differ)
		})
	}
}

func decodePNG(t *testing.T, name string) image.Image {
	t.Helper()
	f, err := assets.FS.Open(name)
	require.NoError(t, err, name)
	img, err := png.Decode(f)
	require.NoError(t, f.Close())
	require.NoError(t, err, name)
	return img
}

// sameFrame は束 atlas の枠 r の画素が元の絵 src と同じかを返す
func sameFrame(atlas image.Image, r AsepriteRect, src image.Image) bool {
	sb := src.Bounds()
	if sb.Dx() != r.W || sb.Dy() != r.H {
		return false
	}
	for y := range r.H {
		for x := range r.W {
			r1, g1, b1, a1 := atlas.At(r.X+x, r.Y+y).RGBA()
			r2, g2, b2, a2 := src.At(sb.Min.X+x, sb.Min.Y+y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				return false
			}
		}
	}
	return true
}
