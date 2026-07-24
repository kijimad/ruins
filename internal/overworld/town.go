package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// townNPCs は小集落に配置する会話NPCの定義名と、集落中心からの相対座標。
// 会話 InteractionTalk で店(商人)・雇用(酒場の主人)・合成(怪しい科学者)を開く。
// 小集落は無状態の補給地で、stash となる収納は置かない。フィールドにアイテムを残さない
// 方針のため、seed からの決定的再生成と整合する。
var townNPCs = []struct {
	name string
	dx   consts.Tile
	dy   consts.Tile
}{
	{"商人", -2, -1},
	{"酒場の主人", -2, 1},
	{"怪しい科学者", -4, 0},
}

// settlementPlacement は小集落のリージョン配置。おおよそ Spacing チャンクに1つ当選する。
var settlementPlacement = Placement{Spacing: 8, Separation: 2, Salt: 0x5e77}

// settlementFeature は小集落の feature 実装。開始チャンクは特例で必ず当選し、
// 新規ゲームの開始点に交易・雇用・合成の必須サービスを保証する。
type settlementFeature struct{}

func (settlementFeature) place(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error {
	if !settlementPlacement.At(runSeed, c, rows) && c != start {
		return nil
	}
	center := consts.Coord[consts.Tile]{X: g.offsetX + g.chunkW/2, Y: g.offsetY + g.chunkH/2}
	return spawnTown(world, center)
}

// spawnTown は小集落を構成する。center を集落の中心として会話NPCを近傍へ決定的に配置する。
// 集落はステージでなくオーバーワールドの地物なので、専用の State を持たず prop として常在する。
// 帯への束縛は呼び出し元のチャンク生成が一括で行う。
func spawnTown(world w.World, center consts.Coord[consts.Tile]) error {
	for _, n := range townNPCs {
		pos := consts.Coord[consts.Tile]{X: center.X + n.dx, Y: center.Y + n.dy}
		if _, err := lifecycle.SpawnNeutralNPC(world, pos, n.name); err != nil {
			return fmt.Errorf("集落NPCの配置に失敗 (%s): %w", n.name, err)
		}
	}
	return nil
}
