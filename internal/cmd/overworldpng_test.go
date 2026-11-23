package cmd

import (
	"bytes"
	"context"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// overworldPNGApp は runOverworldPNG を Action に持つ単体の cli.Command を返す。
// パッケージ変数 CmdOverworldPNG を直接 Run すると実行ごとの状態が共有されるため、テストごとに新規に作る。
func overworldPNGApp() *cli.Command {
	return &cli.Command{
		Name: "overworldpng",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: outFlag},
			&cli.Uint64Flag{Name: seedFlag},
			&cli.IntFlag{Name: northFlag, Value: 20},
			&cli.IntFlag{Name: southFlag, Value: 1},
			&cli.IntFlag{Name: tilePxFlag, Value: 8},
		},
		Action: runOverworldPNG,
	}
}

func TestRunOverworldPNG_seedの地図をPNGに書きseedと出力先を表示する(t *testing.T) {
	t.Parallel()

	out := filepath.Join(t.TempDir(), "out.png")
	var buf bytes.Buffer
	app := overworldPNGApp()
	app.Writer = &buf
	require.NoError(t, app.Run(context.Background(), []string{"overworldpng", "--seed", "7", "--north", "1", "--south", "0", "--tile-px", "2", "--out", out}))
	assert.Equal(t, "seed: 7\nout: "+out+"\n", buf.String())

	f, err := os.Open(out)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	got, err := png.Decode(f)
	require.NoError(t, err)

	world, err := newOverworldWorld()
	require.NoError(t, err)
	want, err := overworld.RenderNorthward(world, 7, 1, 0, 2)
	require.NoError(t, err)
	assert.Equal(t, want.Bounds(), got.Bounds())
	assert.Equal(t, want.Pix, imageRGBA(got).Pix)
}

// imageRGBA は PNG から読んだ画像を RGBA に写す
func imageRGBA(img image.Image) *image.RGBA {
	out := image.NewRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	return out
}

func TestRunOverworldPNG_範囲の外の引数はエラー(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		args []string
	}{
		{"北が負", []string{"--north", "-1"}},
		{"南が負", []string{"--south", "-1"}},
		{"行の数が上限を超える", []string{"--north", "99", "--south", "1"}},
		{"1タイルの画素が小さすぎる", []string{"--tile-px", "1"}},
		{"1タイルの画素が大きすぎる", []string{"--tile-px", "33"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			args := append([]string{"overworldpng", "--out", filepath.Join(t.TempDir(), "x.png")}, tt.args...)
			require.ErrorIs(t, overworldPNGApp().Run(context.Background(), args), errOverworldRange)
		})
	}
}
