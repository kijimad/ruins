package overworld

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

// settlementFeature は小集落の feature 実装。当選チャンクの中央に、集落の行が指す施設の絵を1つ置く。
type settlementFeature struct{}

func (settlementFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	// 区域や墜落現場と重なった当選は譲る。地物の優先度は分類が一元管理する
	if chunkTypeAt(world.Resources.RawMaster, runSeed, c, cols, spawnRowOf(world)) != chunkSettlement {
		return nil
	}
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return fmt.Errorf("failed to load facility templates: %w", err)
	}
	fac := settlementAt(world.Resources.RawMaster, runSeed, c).Facility
	seed := consts.Mix2D(runSeed^placementOf(world.Resources.RawMaster, rawschema.PointFeatureKindSettlement).Salt, c.X, c.Y)
	// seed は settlementAt が種類を選ぶ値と同じだが、ストリーム 0 で絵の抽選に分ける。spawnFacility は戦利品と敵に 0x4 と 0x5 を使う
	rng := rand.New(rand.NewPCG(seed, 0))
	cells, err := planFacility(loader, fac, rng.Uint64(), rng)
	if err != nil {
		return fmt.Errorf("failed to plan settlement %q: %w", fac, err)
	}
	o := centerOrigin(g, cells)
	if _, _, err := spawnFacility(world, g, cells, o.X, o.Y, fac, urbanDangerAt(world, c), seed); err != nil {
		return fmt.Errorf("failed to place settlement %q: %w", fac, err)
	}
	return nil
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
	roll := int(consts.Mix2D(runSeed^placementOf(raws, rawschema.PointFeatureKindSettlement).Salt, c.X, c.Y) % uint64(total))
	for i := range rows {
		roll -= int(rows[i].Weight)
		if roll < 0 {
			return rows[i]
		}
	}
	return rows[len(rows)-1]
}
