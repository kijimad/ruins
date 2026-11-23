package cmd

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/urfave/cli/v3"
)

const (
	seedFlag   = "seed"
	northFlag  = "north"
	southFlag  = "south"
	tilePxFlag = "tile-px"
)

const (
	// maxOverworldRows は1枚に生成するチャンクの行の上限。チャンクを1つ置くたびに world の全タイルを見てオートタイルを
	// 計算し直すので、時間は行の数のおよそ2乗で伸びる。上限の近くでは数十秒と1GB を超えるメモリを使う
	maxOverworldRows = 100
	// minTilePx と maxTilePx は1タイルの画素の数の範囲。小さすぎると座標の文字がチャンクを埋め、スプライトより大きくしても細かくならない
	minTilePx = 2
	maxTilePx = 32
)

// errOverworldRange は生成する行の数や1タイルの画素の数が範囲の外のときに返す
var errOverworldRange = errors.New("north, south or tile-px out of range")

// CmdOverworldPNG は地上の地図を開始の行から北へ長く生成し、チャンクの座標を書き込んだ PNG を書き出すサブコマンド。
// 生成の自然さや面白さを人が目で確かめるための開発用の道具で、書き出した画像はリポジトリに入れない
var CmdOverworldPNG = &cli.Command{
	Name:  "overworldpng",
	Usage: "generate a long overworld region and render it as a png with chunk coordinates",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: outFlag, Usage: "output png path", Value: filepath.Join(os.TempDir(), "overworld.png")},
		&cli.Uint64Flag{Name: seedFlag, Usage: "overworld run seed. random when omitted"},
		&cli.IntFlag{Name: northFlag, Usage: "chunk rows to generate north of the start row. time grows roughly with the square of the rows", Value: 20},
		&cli.IntFlag{Name: southFlag, Usage: "chunk rows to generate south of the start row", Value: 1},
		&cli.IntFlag{Name: tilePxFlag, Usage: "pixels per tile, 2 to 32", Value: 8},
	},
	Action: runOverworldPNG,
}

func runOverworldPNG(_ context.Context, cmd *cli.Command) error {
	seed := cmd.Uint64(seedFlag)
	if !cmd.IsSet(seedFlag) {
		seed = rand.Uint64()
	}
	north, south, tilePx := cmd.Int(northFlag), cmd.Int(southFlag), cmd.Int(tilePxFlag)
	if north < 0 || south < 0 || north+south+1 > maxOverworldRows || tilePx < minTilePx || tilePx > maxTilePx {
		return fmt.Errorf("%w: north and south must be >= 0 with north+south+1 <= %d, tile-px must be %d to %d: north=%d south=%d tile-px=%d",
			errOverworldRange, maxOverworldRows, minTilePx, maxTilePx, north, south, tilePx)
	}
	world, err := newOverworldWorld()
	if err != nil {
		return err
	}
	img, err := overworld.RenderNorthward(world, seed, consts.Chunk(north), consts.Chunk(south), tilePx)
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
