package overworld

import (
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// 遺跡入口は帯全域に点在する階層ステージへの進入口。触れて Enter で潜り、上り階段で
// 地上へ戻る。往復の結線は進入時にポータル機構が戻り先を焼き込むため、配置側は入口 prop と
// 遺跡定義名を決定的に置くだけでよい。

// dungeonEntranceFeature は遺跡入口の feature 実装。
type dungeonEntranceFeature struct{}

// errNoEntranceDungeon は遺跡の入口が引けるダンジョンの行が無いときのエラー。raw の検証が1つ以上を保証する
var errNoEntranceDungeon = errors.New("no dungeon to enter from a ruins entrance")

// place は当選チャンクの中心付近へ遺跡入口を置く。進入先のダンジョンは raw の行の入口の重みから
// チャンク座標のシードで決定的に選ぶ。開始チャンクには driver が歩いて届く入口を別途
// 置くため、ここでは重複を避けてスキップする。
func (dungeonEntranceFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	p := placementOf(world.Resources.RawMaster, rawschema.PointFeatureKindDungeonEntrance)
	if !p.At(runSeed, c, cols) {
		return nil
	}
	rows := raw.PtrSlice(world.Resources.RawMaster.Dungeons)
	rng := rand.New(rand.NewPCG(ChunkSeed2D(runSeed^p.Salt, c.X, c.Y), 0))
	i, ok := consts.Pick(rng, rows, func(d rawschema.Dungeon) float64 { return float64(d.EntranceWeight) })
	if !ok {
		return errNoEntranceDungeon
	}
	pos := consts.Coord[consts.Tile]{X: g.offsetX + g.chunkW/2, Y: g.offsetY + g.chunkH/2}
	if _, err := lifecycle.SpawnDungeonEntrance(world, pos.X, pos.Y, rows[i].Id); err != nil {
		return fmt.Errorf("failed to place ruins entrance (x=%d, y=%d): %w", c.X, c.Y, err)
	}
	return nil
}
