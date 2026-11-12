package cmd

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"io"
	"sort"

	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/urfave/cli/v3"
)

// layoutCellSize はレイアウトの PNG の1マスの辺の長さ
const layoutCellSize = 8

// errNoLayoutArg はレイアウトの名前が渡されないときに返す
var errNoLayoutArg = errors.New("layout name is required")

// CmdLayoutPNG はレイアウトのテンプレートを展開し、マスを色で塗った PNG を書き出すサブコマンド
var CmdLayoutPNG = &cli.Command{
	Name:      "layoutpng",
	Usage:     "render a layout template as a png",
	ArgsUsage: "<template name>",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: outFlag, Usage: "output png path", Value: "layout.png"},
		&cli.Uint64Flag{Name: "seed", Usage: "seed to pick variants and placements", Value: 1},
	},
	Action: runLayoutPNG,
}

func runLayoutPNG(_ context.Context, cmd *cli.Command) error {
	if !cmd.Args().Present() {
		return errNoLayoutArg
	}
	loader := maptemplate.NewTemplateLoader()
	if err := loader.RegisterAllChunks([]string{"levels/chunks", "levels/facilities", "levels/layouts"}); err != nil {
		return err
	}
	if err := loader.RegisterAllPalettes([]string{"levels/palettes"}); err != nil {
		return err
	}
	_, _, cells, err := loader.LoadTemplateByName(cmd.Args().First(), cmd.Uint64("seed"))
	if err != nil {
		return err
	}
	if err := writePNG(cmd.String(outFlag), renderLayout(cells, layoutCellSize)); err != nil {
		return err
	}
	return writeLegend(cmd.Writer, cells)
}

var (
	propColor = color.NRGBA{R: 40, G: 40, B: 40, A: 255}
	npcColor  = color.NRGBA{R: 220, G: 40, B: 40, A: 255}
)

// renderLayout はマスを地形の色で塗る。設置物はマスの内側の四角、NPC はマスの中央の点で示す。
func renderLayout(cells [][]maptemplate.MapCell, size int) *image.NRGBA {
	width := 0
	for _, row := range cells {
		width = max(width, len(row))
	}
	img := image.NewNRGBA(image.Rect(0, 0, width*size, len(cells)*size))
	for y, row := range cells {
		for x, c := range row {
			r := image.Rect(x*size, y*size, (x+1)*size, (y+1)*size)
			draw.Draw(img, r, image.NewUniform(terrainColor(c.Terrain)), image.Point{}, draw.Src)
			if c.Prop != "" {
				draw.Draw(img, r.Inset(size/4), image.NewUniform(propColor), image.Point{}, draw.Src)
			}
			if c.NPC != "" {
				draw.Draw(img, r.Inset(size*3/8), image.NewUniform(npcColor), image.Point{}, draw.Src)
			}
		}
	}
	return img
}

// terrainColor は地形の名前から決まる明るめの色を返す。同じ名前はいつも同じ色になる。
func terrainColor(name string) color.NRGBA {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	v := h.Sum32()
	return color.NRGBA{R: 96 + uint8(v%128), G: 96 + uint8((v>>8)%128), B: 96 + uint8((v>>16)%128), A: 255}
}

// writeLegend は地形の名前と色の対応を、名前の順に書く。
func writeLegend(w io.Writer, cells [][]maptemplate.MapCell) error {
	seen := map[string]bool{}
	for _, row := range cells {
		for _, c := range row {
			seen[c.Terrain] = true
		}
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		c := terrainColor(n)
		if _, err := fmt.Fprintf(w, "#%02x%02x%02x %s\n", c.R, c.G, c.B, n); err != nil {
			return err
		}
	}
	return nil
}
