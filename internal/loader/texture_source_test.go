package loader

import (
	"image"
	"image/png"
	"io/fs"
	"path"
	"testing"

	"github.com/kijimaD/ruins/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextureSources_インデックス形式のPNGを置かない は、束にまとめる元の絵が色の表を持つインデックス形式で
// ないことを確かめる。make aseprite はインデックス形式の絵を色を見ずに中身の番号だけで同じ絵とみなして
// 1枚にまとめるので、色違いの絵が黙って同じ色で描かれる
func TestTextureSources_インデックス形式のPNGを置かない(t *testing.T) {
	t.Parallel()

	var checked int
	var paletted []string
	for _, dir := range []string{"tiles", "single", "bg"} {
		root := path.Join("file/textures", dir)
		entries, err := fs.ReadDir(assets.FS, root)
		require.NoError(t, err, root)
		for _, e := range entries {
			if e.IsDir() || path.Ext(e.Name()) != ".png" {
				continue
			}
			name := path.Join(root, e.Name())
			f, err := assets.FS.Open(name)
			require.NoError(t, err, name)
			img, err := png.Decode(f)
			require.NoError(t, f.Close())
			require.NoError(t, err, name)
			if _, ok := img.(*image.Paletted); ok {
				paletted = append(paletted, name)
			}
			checked++
		}
	}
	require.Positive(t, checked, "絵が見つからないと検査が素通りする")
	assert.Empty(t, paletted)
}
