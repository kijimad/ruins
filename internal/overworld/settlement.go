package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// relSpot は基準点からの相対座標に置くエンティティの定義。ランドマークの原点や散布の起点を基準に、
// prop をずらして配置するのに使う。
type relSpot struct {
	name string
	dx   consts.Tile
	dy   consts.Tile
}

// settlementFeature は小集落の feature 実装。当選チャンクに村または一軒家を置く。
type settlementFeature struct{}

func (settlementFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	if !settlementPlacement.At(runSeed, c, cols) {
		return nil
	}
	// 市街地と重なった当選は市街地へ譲る。安全な補給地が危険地帯の中に出るのを防ぐ
	if _, _, _, ok := urbanRegionOf(runSeed, c, cols); ok {
		return nil
	}
	center := consts.Coord[consts.Tile]{X: g.offsetX + g.chunkW/2, Y: g.offsetY + g.chunkH/2}
	return spawnSettlement(world, center, settlementAt(world.Resources.RawMaster, runSeed, c))
}

// settlementAt は集落の種類を raw の settlements の行から重みの比で選ぶ。純粋に (runSeed, 座標) の
// 関数で、地図の分類と生成の両方がこれを使うので種類の見た目と実体が一致する。
// 行は読み込み時に1つ以上あり、重みは1以上であることを検証している。
func settlementAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk]) rawschema.Settlement {
	rows := raw.PtrSlice(raws.Settlements)
	total := 0
	for i := range rows {
		total += int(rows[i].Weight)
	}
	roll := int(ChunkSeed2D(runSeed^settlementSalt, c.X, c.Y) % uint64(total))
	for i := range rows {
		roll -= int(rows[i].Weight)
		if roll < 0 {
			return rows[i]
		}
	}
	return rows[len(rows)-1]
}

// spawnSettlement は小集落を構成する。center を集落の中心として、行の住人と生活感の prop を
// 近傍へ決定的に配置する。住人と prop の座標は重ねない。
// 集落はステージでなくオーバーワールドの地物なので、専用の State を持たず prop として常在する。
// 帯への束縛は呼び出し元のチャンク生成が一括で行う。
func spawnSettlement(world w.World, center consts.Coord[consts.Tile], s rawschema.Settlement) error {
	for _, n := range s.Members {
		pos := consts.Coord[consts.Tile]{X: center.X + consts.Tile(n.Dx), Y: center.Y + consts.Tile(n.Dy)}
		if _, err := lifecycle.SpawnNeutralNPC(world, pos, n.Name); err != nil {
			return fmt.Errorf("failed to place settlement NPC (%s): %w", n.Name, err)
		}
	}
	for _, p := range s.Props {
		if _, err := lifecycle.SpawnProp(world, p.Name, center.X+consts.Tile(p.Dx), center.Y+consts.Tile(p.Dy)); err != nil {
			return fmt.Errorf("failed to place settlement prop (%s): %w", p.Name, err)
		}
	}
	return nil
}
