package cmd

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// spriteCellSize はスプライトシートの1マスの辺の長さ
const spriteCellSize = 32

// outFlag は書き出し先を受け取るフラグの名前
const outFlag = "out"

// errNoImageArg は切り出す画像のパスが渡されないときに返す
var errNoImageArg = errors.New("image path is required")

// CmdCutSprite は画像を32ピクセル角のマスに切り、単体のスプライトとして書き出すサブコマンド
var CmdCutSprite = &cli.Command{
	Name:      "cutsprite",
	Usage:     "cut an image into 32x32 sprites",
	ArgsUsage: "<image.png>",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "prefix", Usage: "file name prefix of the sprites", Required: true},
		&cli.StringFlag{Name: outFlag, Usage: "output directory", Value: "assets/file/textures/single"},
	},
	Action: runCutSprite,
}

func runCutSprite(_ context.Context, cmd *cli.Command) error {
	if !cmd.Args().Present() {
		return errNoImageArg
	}
	f, err := os.Open(cmd.Args().First())
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	img, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode png: %w", err)
	}
	cells := cutCells(img, spriteCellSize)
	for _, c := range cells {
		name := fmt.Sprintf("%s_%d_%d_.png", cmd.String("prefix"), c.row, c.col)
		if err := writePNG(filepath.Join(cmd.String(outFlag), name), c.img); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(cmd.Writer, "wrote %d sprites\n", len(cells))
	return err
}

// spriteCell は切り出した1マス
type spriteCell struct {
	row, col int
	img      *image.NRGBA
}

// cutCells は img を size ピクセル角のマスに切り、すべての画素が透明なマスを除いて行と列の順に返す。
// 端に余った size に満たない部分は捨てる。
func cutCells(img image.Image, size int) []spriteCell {
	b := img.Bounds()
	var out []spriteCell
	for row := range b.Dy() / size {
		for col := range b.Dx() / size {
			r := image.Rect(b.Min.X+col*size, b.Min.Y+row*size, b.Min.X+(col+1)*size, b.Min.Y+(row+1)*size)
			cell := image.NewNRGBA(image.Rect(0, 0, size, size))
			draw.Draw(cell, cell.Bounds(), img, r.Min, draw.Src)
			if transparent(cell) {
				continue
			}
			out = append(out, spriteCell{row: row, col: col, img: cell})
		}
	}
	return out
}

func transparent(img *image.NRGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			return false
		}
	}
	return true
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
