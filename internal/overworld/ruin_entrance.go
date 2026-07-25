package overworld

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// 遺跡入口は帯全域に点在する階層ステージへの進入口。触れて Enter で潜り、上り階段で
// 地上へ戻る。往復の結線は進入時にポータル機構が戻り先を焼き込むため、配置側は入口 prop と
// 遺跡定義名を決定的に置くだけでよい。

// ruinPlacement は遺跡入口のリージョン配置。小集落と市街地の中間の疎らさで置く。
// チャンクは20タイルなので、Spacing 4 で80タイルに1つの体感密度になる。
// 開始付近にも歩いて到達できる入口が必ず出る
var ruinPlacement = Placement{Spacing: 4, Separation: 1, Salt: ruinSalt}

// ruinEntranceFeature は遺跡入口の feature 実装。
type ruinEntranceFeature struct{}

// place は当選チャンクの中心付近へ遺跡入口を置く。進入先の遺跡定義は登録済み一覧から
// チャンク座標のシードで決定的に選ぶ。開始チャンクには driver が歩いて届く入口を別途
// 置くため、ここでは重複を避けてスキップする。
func (ruinEntranceFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], rows consts.Chunk, g chunkGeom) error {
	if !ruinPlacement.At(runSeed, c, rows) {
		return nil
	}
	defs := dungeon.GetAllDungeons()
	if len(defs) == 0 {
		return nil
	}
	rng := rand.New(rand.NewPCG(ChunkSeed2D(runSeed^ruinSalt, c.X, c.Y), 0))
	def := defs[rng.IntN(len(defs))]
	pos := consts.Coord[consts.Tile]{X: g.offsetX + g.chunkW/2, Y: g.offsetY + g.chunkH/2}
	if _, err := lifecycle.SpawnDungeonEntrance(world, pos.X, pos.Y, def.Name()); err != nil {
		return fmt.Errorf("遺跡入口の配置に失敗 (x=%d, y=%d): %w", c.X, c.Y, err)
	}
	return nil
}
