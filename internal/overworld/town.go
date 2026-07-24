package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// townSpot は集落中心からの相対座標に置くエンティティの定義。
type townSpot struct {
	name string
	dx   consts.Tile
	dy   consts.Tile
}

// villageNPCs は村に配置する会話NPC。会話 InteractionTalk で店(商人)・雇用(酒場の主人)・
// 合成(怪しい科学者)を開く。小集落は無状態の補給地で、stash となる収納は置かない。
// フィールドにアイテムを残さない方針のため、seed からの決定的再生成と整合する。
var villageNPCs = []townSpot{
	{"商人", -2, -1},
	{"酒場の主人", -2, 1},
	{"怪しい科学者", -4, 0},
}

// hamletNPCs は一軒家に配置する会話NPC。行商の拠点という位置づけで商人だけがいる。
var hamletNPCs = []townSpot{
	{"商人", -2, -1},
}

// villageProps と hamletProps は集落の生活感を出す prop。NPC の座標と重ねない。
var (
	villageProps = []townSpot{
		{"bonfire", 2, -2},
		{"bench", 3, 1},
		{"wooden_sign", 0, -3},
	}
	hamletProps = []townSpot{
		{"bonfire", 2, -2},
		{"crate", 1, 2},
	}
)

// settlementSalt は小集落の配置と規模抽選の相関を他地物と切る。
const settlementSalt = 0x5e77

// settlementPlacement は小集落のリージョン配置。おおよそ Spacing チャンクに1つ当選する。
// チャンクは50タイルあるため、Spacing 5 で250タイルに1つの体感密度になる。
var settlementPlacement = Placement{Spacing: 5, Separation: 1, Salt: settlementSalt}

// settlementFeature は小集落の feature 実装。開始チャンクは特例で必ず当選し、
// 新規ゲームの開始点に交易・雇用・合成の必須サービスを保証する。
type settlementFeature struct{}

func (settlementFeature) place(world w.World, runSeed uint64, c, start worldstream.ChunkCoord, rows consts.Chunk, g chunkGeom) error {
	if c != start {
		if !settlementPlacement.At(runSeed, c, rows) {
			return nil
		}
		// 市街地と重なった当選は市街地へ譲る。安全な補給地が危険地帯の中に出るのを防ぐ
		if _, _, ok := urbanAnchorOf(runSeed, c, rows); ok {
			return nil
		}
	}
	center := consts.Coord[consts.Tile]{X: g.offsetX + g.chunkW/2, Y: g.offsetY + g.chunkH/2}
	return spawnTown(world, center, settlementIsVillage(runSeed, c, start))
}

// settlementIsVillage は集落の規模を決定的に選ぶ。真なら村、偽なら一軒家。
// 開始チャンクは交易・雇用・合成の必須サービスを保証するため必ず村にする。
func settlementIsVillage(runSeed uint64, c, start worldstream.ChunkCoord) bool {
	if c == start {
		return true
	}
	return ChunkSeed2D(runSeed^settlementSalt, c.X, c.Y)%10 < 6
}

// spawnTown は小集落を構成する。center を集落の中心として会話NPCと生活感の prop を
// 近傍へ決定的に配置する。村は全サービスのNPCが揃い、一軒家は商人だけの行商拠点になる。
// 集落はステージでなくオーバーワールドの地物なので、専用の State を持たず prop として常在する。
// 帯への束縛は呼び出し元のチャンク生成が一括で行う。
func spawnTown(world w.World, center consts.Coord[consts.Tile], village bool) error {
	npcs, props := hamletNPCs, hamletProps
	if village {
		npcs, props = villageNPCs, villageProps
	}
	for _, n := range npcs {
		pos := consts.Coord[consts.Tile]{X: center.X + n.dx, Y: center.Y + n.dy}
		if _, err := lifecycle.SpawnNeutralNPC(world, pos, n.name); err != nil {
			return fmt.Errorf("集落NPCの配置に失敗 (%s): %w", n.name, err)
		}
	}
	for _, p := range props {
		if _, err := lifecycle.SpawnProp(world, p.name, center.X+p.dx, center.Y+p.dy); err != nil {
			return fmt.Errorf("集落propの配置に失敗 (%s): %w", p.name, err)
		}
	}
	return nil
}
