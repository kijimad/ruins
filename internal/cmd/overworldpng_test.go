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

func TestRunOverworldPNG_同じseedなら同じ画像を書き出しseedと出力先を表示する(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	names := []string{"a.png", "b.png"}
	images := make([][]byte, 0, len(names))
	for _, name := range names {
		out := filepath.Join(dir, name)
		var buf bytes.Buffer
		app := overworldPNGApp()
		app.Writer = &buf
		require.NoError(t, app.Run(context.Background(), []string{"overworldpng", "--seed", "7", "--north", "1", "--south", "0", "--tile-px", "2", "--out", out}))
		assert.Equal(t, "seed: 7\nout: "+out+"\n", buf.String())

		bs, err := os.ReadFile(out)
		require.NoError(t, err)
		images = append(images, bs)
	}
	assert.Equal(t, images[0], images[1])

	img, err := png.Decode(bytes.NewReader(images[0]))
	require.NoError(t, err)
	// 地上は 9 列 × 24 タイル、行は開始の行と北の1行の2行。左と上に座標の余白を足す
	assert.Equal(t, image.Rect(0, 0, gridMarginLeft+9*24*2, gridMarginTop+2*24*2), img.Bounds())
}

func TestRunOverworldPNG_負の行数はエラー(t *testing.T) {
	t.Parallel()

	err := overworldPNGApp().Run(context.Background(), []string{"overworldpng", "--north", "-1", "--out", filepath.Join(t.TempDir(), "x.png")})
	require.ErrorIs(t, err, errOverworldRange)
}

func TestWithChunkGrid_余白を足しチャンクの境目に線を引く(t *testing.T) {
	t.Parallel()

	m := image.NewRGBA(image.Rect(0, 0, 40, 60))
	img := withChunkGrid(m, 20, 30, 2, 2, 1)
	assert.Equal(t, image.Rect(0, 0, gridMarginLeft+40, gridMarginTop+60), img.Bounds())
	// 透明な地図の上に半透明の白い線を重ねる。座標の文字の届かない、チャンクの下の方で確かめる
	y := gridMarginTop + 55
	assert.Equal(t, color.RGBA{R: 160, G: 160, B: 160, A: 160}, img.RGBAAt(gridMarginLeft+20, y), "列の境目")
	assert.Equal(t, color.RGBA{}, img.RGBAAt(gridMarginLeft+10, y), "チャンクの中")
}
