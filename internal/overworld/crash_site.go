package overworld

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
)

// spawnChunkOf は帯の列数 cols と開始の行 spawnRow から開始チャンクを返す
func spawnChunkOf(cols, spawnRow consts.Chunk) consts.Coord[consts.Chunk] {
	return consts.Coord[consts.Chunk]{X: cols / 2, Y: spawnRow}
}

// crashSiteAt は開始チャンク c に置く墜落現場の施設の id を、候補の重みで引く
func crashSiteAt(raws rawschema.Raws, runSeed uint64, c consts.Coord[consts.Chunk]) string {
	sites := raw.PtrSlice(raws.CrashSites)
	i, ok := consts.Pick(consts.HashRoll(consts.Mix2D(runSeed^crashSiteSalt, c.X, c.Y)), sites, func(s rawschema.CrashSite) float64 { return float64(s.Weight) })
	if !ok {
		// スキーマが重みを1以上に縛り、TestRealData_HasCrashSites が実 raw の候補の存在を固定する
		panic("crashSiteAt: no crash site to pick")
	}
	return sites[i].Id
}

// crashSiteFeature は宇宙船が墜落した開始の現場の feature 実装
type crashSiteFeature struct{}

// place は開始チャンクに墜落現場の絵を1枚、回転も反転もせずにチャンクの真ん中へ置く。
// 絵の東側をキューブの置き場所として空けているので、向きを変えない
func (crashSiteFeature) place(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], cols consts.Chunk, g chunkGeom) error {
	raws := world.Resources.RawMaster
	if chunkTypeAt(raws, runSeed, c, cols, spawnRowOf(world)) != chunkCrashSite {
		return nil
	}
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return fmt.Errorf("failed to load facility templates: %w", err)
	}
	fac := crashSiteAt(raws, runSeed, c)
	seed := consts.Mix2D(runSeed^crashSiteSalt, c.X, c.Y)
	_, cells, err := loader.FacilityChunk(fac, seed)
	if err != nil {
		return fmt.Errorf("failed to plan crash site %q: %w", fac, err)
	}
	ox := g.chunkW/2 - consts.Tile(len(cells[0]))/2
	oy := g.chunkH/2 - consts.Tile(len(cells))/2
	if _, _, err := spawnFacility(world, g, cells, ox, oy, fac, urbanDangerAt(world, c), seed); err != nil {
		return fmt.Errorf("failed to place crash site %q: %w", fac, err)
	}
	return nil
}
