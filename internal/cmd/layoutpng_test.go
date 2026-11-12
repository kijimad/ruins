package cmd

import (
	"bytes"
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderLayout_地形と設置物とNPCを塗る(t *testing.T) {
	t.Parallel()

	cells := [][]maptemplate.MapCell{
		{{Terrain: "floor"}, {Terrain: "floor", Prop: "crate"}},
		{{Terrain: "wall"}, {Terrain: "floor", NPC: "slime"}},
	}

	img := renderLayout(cells, 8)

	assert.Equal(t, image.Rect(0, 0, 16, 16), img.Bounds())
	assert.Equal(t, terrainColor("floor"), img.NRGBAAt(0, 0))
	assert.Equal(t, terrainColor("wall"), img.NRGBAAt(0, 8))
	assert.Equal(t, terrainColor("floor"), img.NRGBAAt(8, 0), "設置物はマスの内側だけに描く")
	assert.Equal(t, propColor, img.NRGBAAt(12, 4))
	assert.Equal(t, npcColor, img.NRGBAAt(12, 12))
}

func TestWriteLegend_地形の名前の順に色を書く(t *testing.T) {
	t.Parallel()

	cells := [][]maptemplate.MapCell{{{Terrain: "wall"}, {Terrain: "floor"}, {Terrain: "wall"}}}
	var b bytes.Buffer
	require.NoError(t, writeLegend(&b, cells))

	f, w := terrainColor("floor"), terrainColor("wall")
	want := "#" + hex(f.R) + hex(f.G) + hex(f.B) + " floor\n#" + hex(w.R) + hex(w.G) + hex(w.B) + " wall\n"
	assert.Equal(t, want, b.String())
}

func hex(v uint8) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[v>>4], digits[v&0xf]})
}

func TestRenderLayout_実データのレイアウトを展開して塗れる(t *testing.T) {
	t.Parallel()

	loader := maptemplate.NewTemplateLoader()
	require.NoError(t, loader.RegisterAllChunks([]string{"levels/chunks", "levels/facilities", "levels/layouts"}))
	require.NoError(t, loader.RegisterAllPalettes([]string{"levels/palettes"}))
	_, _, cells, err := loader.LoadTemplateByName("10x10_small_room", 1)
	require.NoError(t, err)

	img := renderLayout(cells, 8)
	assert.Equal(t, image.Rect(0, 0, 80, 80), img.Bounds())
}
