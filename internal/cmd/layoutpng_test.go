package cmd

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
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

func TestRenderLayout_マスサイズに応じて境界が変わる(t *testing.T) {
	t.Parallel()

	cells := [][]maptemplate.MapCell{{{Terrain: "floor"}, {Terrain: "wall"}}}
	img := renderLayout(cells, 3)

	assert.Equal(t, image.Rect(0, 0, 6, 3), img.Bounds())
	assert.Equal(t, terrainColor("floor"), img.NRGBAAt(2, 0))
	assert.Equal(t, terrainColor("wall"), img.NRGBAAt(3, 0))
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

// layoutPNGApp は runLayoutPNG を Action に持つ単体の cli.Command を返す。
// パッケージ変数 CmdLayoutPNG を直接 Run すると実行ごとの状態が共有されるため、テストごとに新規に作る。
func layoutPNGApp() *cli.Command {
	return &cli.Command{
		Name:      "layoutpng",
		ArgsUsage: "<template name>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: outFlag, Value: "layout.png"},
			&cli.Uint64Flag{Name: "seed", Value: 1},
		},
		Action: runLayoutPNG,
	}
}

func TestRunLayoutPNG_テンプレート名がないとエラー(t *testing.T) {
	t.Parallel()

	err := layoutPNGApp().Run(context.Background(), []string{"layoutpng"})
	require.ErrorIs(t, err, errNoLayoutArg)
}

func TestRunLayoutPNG_実データのレイアウトをPNGと凡例に書き出す(t *testing.T) {
	t.Parallel()

	loader := maptemplate.NewTemplateLoader()
	require.NoError(t, loader.RegisterAllChunks([]string{"levels/chunks", "levels/facilities", "levels/layouts"}))
	require.NoError(t, loader.RegisterAllPalettes([]string{"levels/palettes"}))
	_, _, wantCells, err := loader.LoadTemplateByName("10x10_small_room", 1)
	require.NoError(t, err)

	outPath := filepath.Join(t.TempDir(), "out.png")
	var buf bytes.Buffer
	app := layoutPNGApp()
	app.Writer = &buf

	err = app.Run(context.Background(), []string{"layoutpng", "--out", outPath, "--seed", "1", "10x10_small_room"})
	require.NoError(t, err)

	f, err := os.Open(outPath)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	got, err := png.Decode(f)
	require.NoError(t, err)
	want := renderLayout(wantCells, layoutCellSize)
	require.Equal(t, want.Bounds(), got.Bounds())
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			require.Equal(t, want.NRGBAAt(x, y), color.NRGBAModel.Convert(got.At(x, y)))
		}
	}

	var legend bytes.Buffer
	require.NoError(t, writeLegend(&legend, wantCells))
	assert.Equal(t, legend.String(), buf.String())
}
