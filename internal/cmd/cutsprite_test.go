package cmd

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
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

func TestWritePNG_書き出した画像を読み戻せる(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	red := color.NRGBA{R: 255, A: 255}
	img.Set(1, 0, red)

	path := filepath.Join(t.TempDir(), "out.png")
	require.NoError(t, writePNG(path, img))

	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	got, err := png.Decode(f)
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 2, 2), got.Bounds())
	r, g, b, a := got.At(1, 0).RGBA()
	assert.Equal(t, [4]uint32{0xffff, 0, 0, 0xffff}, [4]uint32{r, g, b, a})
}

func TestWritePNG_作成できないパスはエラー(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	path := filepath.Join(t.TempDir(), "no-such-dir", "out.png")

	err := writePNG(path, img)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

// cutSpriteApp は runCutSprite を Action に持つ単体の cli.Command を返す。
// パッケージ変数 CmdCutSprite を直接 Run すると実行ごとの状態が共有されるため、テストごとに新規に作る。
func cutSpriteApp() *cli.Command {
	return &cli.Command{
		Name:      "cutsprite",
		ArgsUsage: "<image.png>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "prefix", Usage: "file name prefix of the sprites", Required: true},
			&cli.StringFlag{Name: outFlag, Usage: "output directory", Value: "assets/file/textures/single"},
		},
		Action: runCutSprite,
	}
}

func TestRunCutSprite_画像引数がないとエラー(t *testing.T) {
	t.Parallel()

	err := cutSpriteApp().Run(context.Background(), []string{"cutsprite", "--prefix", "p"})
	require.ErrorIs(t, err, errNoImageArg)
}

func TestRunCutSprite_存在しない画像はエラー(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.png")
	err := cutSpriteApp().Run(context.Background(), []string{"cutsprite", "--prefix", "p", path})
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestRunCutSprite_不正なpngはエラー(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "broken.png")
	require.NoError(t, os.WriteFile(path, []byte("not a png"), 0o644))

	err := cutSpriteApp().Run(context.Background(), []string{"cutsprite", "--prefix", "p", path})
	var fe png.FormatError
	require.ErrorAs(t, err, &fe)
}

func TestRunCutSprite_透明なマスを除いてスプライトを書き出す(t *testing.T) {
	t.Parallel()

	// 64x32の画像を32角で切ると2マス。左は不透明、右は未設定のまま透明に残す
	src := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	red := color.NRGBA{R: 255, A: 255}
	for y := range 32 {
		for x := range 32 {
			src.Set(x, y, red)
		}
	}

	srcPath := filepath.Join(t.TempDir(), "sheet.png")
	f, err := os.Create(srcPath)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, src))
	require.NoError(t, f.Close())

	outDir := t.TempDir()
	var buf bytes.Buffer
	app := cutSpriteApp()
	app.Writer = &buf

	err = app.Run(context.Background(), []string{"cutsprite", "--prefix", "chip", "--out", outDir, srcPath})
	require.NoError(t, err)
	assert.Equal(t, "wrote 1 sprites\n", buf.String())

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "chip_0_0_.png", entries[0].Name())
}
