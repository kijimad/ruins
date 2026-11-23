package cmd

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/urfave/cli/v3"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	seedFlag   = "seed"
	northFlag  = "north"
	southFlag  = "south"
	tilePxFlag = "tile-px"
)

// errOverworldRange は生成する行の数や1タイルの画素の数が負や0のときに返す
var errOverworldRange = errors.New("north and south must be >= 0 and tile-px must be > 0")

// CmdOverworldPNG は地上の地図を開始の行から北へ長く生成し、チャンクの座標を書き込んだ PNG を書き出すサブコマンド。
// 生成の自然さや面白さを人が目で確かめるための開発用の道具で、書き出した画像はリポジトリに入れない
var CmdOverworldPNG = &cli.Command{
	Name:  "overworldpng",
	Usage: "generate a long overworld region and render it as a png with chunk coordinates",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: outFlag, Usage: "output png path", Value: filepath.Join(os.TempDir(), "overworld.png")},
		&cli.Uint64Flag{Name: seedFlag, Usage: "overworld run seed. random when omitted"},
		&cli.IntFlag{Name: northFlag, Usage: "chunk rows to generate north of the start row", Value: 20},
		&cli.IntFlag{Name: southFlag, Usage: "chunk rows to generate south of the start row", Value: 1},
		&cli.IntFlag{Name: tilePxFlag, Usage: "pixels per tile", Value: 8},
	},
	Action: runOverworldPNG,
}

func runOverworldPNG(_ context.Context, cmd *cli.Command) error {
	seed := cmd.Uint64(seedFlag)
	if !cmd.IsSet(seedFlag) {
		seed = rand.Uint64()
	}
	north, south, tilePx := cmd.Int(northFlag), cmd.Int(southFlag), cmd.Int(tilePxFlag)
	if north < 0 || south < 0 || tilePx <= 0 {
		return fmt.Errorf("%w: north=%d south=%d tile-px=%d", errOverworldRange, north, south, tilePx)
	}
	img, err := renderOverworld(seed, consts.Chunk(north), consts.Chunk(south), tilePx)
	if err != nil {
		return err
	}
	out := cmd.String(outFlag)
	if err := writePNG(out, img); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.Writer, "seed: %d\nout: %s\n", seed, out)
	return err
}

// renderOverworld は地上の定義の帯の形で、開始の行から北へ north 行、南へ south 行のチャンクを生成し、座標を書き込んだ画像にする
func renderOverworld(seed uint64, north, south consts.Chunk, tilePx int) (*image.RGBA, error) {
	world, err := newOverworldWorld()
	if err != nil {
		return nil, err
	}
	chunkW, chunkH, cols, rows := dungeon.DungeonOverworld.BandShape()
	sb := query.EnsureSeamlessBand(world)
	*sb = gc.SeamlessBand{Active: true, RunSeed: seed, ChunkW: chunkW, ChunkH: chunkH, Cols: cols, Rows: rows}
	spawnRow := sb.SpawnChunkY()

	top := spawnRow - north
	gen := overworld.NewChunkGen(world, seed, chunkW, chunkH, cols, mapplanner.PlannerTypeOverworldField)
	for cy := top; cy <= spawnRow+south; cy++ {
		for cx := range cols {
			if err := gen(consts.Coord[consts.Chunk]{X: cx, Y: cy}, cx.Tiles(chunkW), (cy - top).Tiles(chunkH)); err != nil {
				return nil, err
			}
		}
	}
	mapRows := north + south + 1
	area := image.Rect(0, 0, int(cols.Tiles(chunkW)), int(mapRows.Tiles(chunkH)))
	m, err := overworld.RenderRegion(world, area, tilePx)
	if err != nil {
		return nil, err
	}
	return withChunkGrid(m, int(chunkW)*tilePx, int(chunkH)*tilePx, int(cols), int(mapRows), int(north)), nil
}

// newOverworldWorld は地上の生成に要る raw と定義だけを持つ world を作る。描画やシステムは組まない
func newOverworldWorld() (w.World, error) {
	cfg := &config.Config{Profile: config.ProfileDevelopment}
	cfg.ApplyProfileDefaults()
	cfg.LogLevel = "ignore"
	world, err := w.InitWorld(&gc.Components{}, cfg)
	if err != nil {
		return w.World{}, err
	}
	rw, err := loader.LoadRaws()
	if err != nil {
		return w.World{}, err
	}
	world.Resources.RawMaster = rw
	if world.Resources.ConditionDefs, err = raw.NewConditionDefs(rw); err != nil {
		return w.World{}, err
	}
	if world.Resources.SkillDefs, err = raw.NewSkillDefs(rw); err != nil {
		return w.World{}, err
	}
	if world.Resources.MaterialDefs, err = raw.NewMaterialDefs(rw); err != nil {
		return w.World{}, err
	}
	world.Resources.WeatherDefs = raw.NewWeatherDefs(rw)
	world.Resources.TileEnvDefs = raw.NewTileEnvDefs(rw)
	world.Resources.NeedDefs = raw.NewNeedDefs(rw)
	world.Resources.CalendarDefs = raw.NewCalendarDefs(rw)
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	return world, nil
}

const (
	// gridMarginLeft と gridMarginTop は地図の左と上に空ける、行と列の番号を書く余白
	gridMarginLeft = 40
	gridMarginTop  = 18
)

var (
	gridLineColor  = color.NRGBA{R: 255, G: 255, B: 255, A: 160}
	gridLabelColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	gridLabelBack  = color.NRGBA{A: 200}
	gridMarginBack = color.NRGBA{R: 32, G: 32, B: 36, A: 255}
)

// withChunkGrid は地図 m の周りに余白を足し、チャンクの境目に線を引いて、チャンクごとに座標を書く。列は西から A, B, C、
// 行は開始の行を 0 として北へ 1, 2, 3 と数える。一番上の行は north。どのチャンクにも C5 のような座標を書く
func withChunkGrid(m *image.RGBA, chunkPxW, chunkPxH, cols, rows, north int) *image.RGBA {
	b := m.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, gridMarginLeft+b.Dx(), gridMarginTop+b.Dy()))
	draw.Draw(img, img.Bounds(), image.NewUniform(gridMarginBack), image.Point{}, draw.Src)
	draw.Draw(img, b.Add(image.Pt(gridMarginLeft, gridMarginTop)), m, image.Point{}, draw.Src)

	line := image.NewUniform(gridLineColor)
	for c := 0; c <= cols; c++ {
		x := gridMarginLeft + c*chunkPxW
		draw.Draw(img, image.Rect(x, gridMarginTop, x+1, img.Bounds().Dy()), line, image.Point{}, draw.Over)
	}
	for r := 0; r <= rows; r++ {
		y := gridMarginTop + r*chunkPxH
		draw.Draw(img, image.Rect(gridMarginLeft, y, img.Bounds().Dx(), y+1), line, image.Point{}, draw.Over)
	}
	for c := range cols {
		drawGridLabel(img, gridMarginLeft+c*chunkPxW+chunkPxW/2-3, 13, columnName(c), false)
	}
	for r := range rows {
		n := strconv.Itoa(north - r)
		drawGridLabel(img, 4, gridMarginTop+r*chunkPxH+chunkPxH/2+4, n, false)
		for c := range cols {
			drawGridLabel(img, gridMarginLeft+c*chunkPxW+4, gridMarginTop+r*chunkPxH+14, columnName(c)+n, true)
		}
	}
	return img
}

// columnName は西から c 番目の列の名前を A から返す
func columnName(c int) string {
	return string(rune('A' + c))
}

// drawGridLabel は (x, y) を文字の基準線の左端として s を書く。back が立っていれば文字の下に暗い帯を敷く
func drawGridLabel(img *image.RGBA, x, y int, s string, back bool) {
	face := basicfont.Face7x13
	if back {
		r := image.Rect(x-2, y-11, x+len(s)*face.Advance+2, y+3)
		draw.Draw(img, r, image.NewUniform(gridLabelBack), image.Point{}, draw.Over)
	}
	d := font.Drawer{Dst: img, Src: image.NewUniform(gridLabelColor), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(s)
}
