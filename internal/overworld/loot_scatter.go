package overworld

import (
	"fmt"
	"math"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
)

// 屋外の散布 loot は開けた地形へ拾える物をまばらに撒く。建物内の役割別 loot と違い、区画に応じたテーブルから
// 引く。道沿いは人の営みの残りで廃墟テーブル、奥地は野外の森テーブルとし、廃墟感を地表にも出す。密度は
// 低くして原野を埋め尽くさない。土系の地面かつ非占有のタイルにだけ置き、草や樹木、建物と重ならない。

// outdoorLootDensity は屋外散布 loot の面積あたり係数。count = round(area * density) で個数を密度確率から
// 導く。反復乱数でなく密度で決めるので純関数のまま。低密度にして地表の loot を稀少に保つ。
const outdoorLootDensity = 0.008

// outdoorLootDepth は地上フロアの深度。廃墟・森テーブルは最も浅い entry が minDepth 1 なので、地表の loot は
// 1 で引く。深度 0 では全 entry が弾かれ何も出ない。収納 loot の populateStorageLoot と同じ扱い。
const outdoorLootDepth = 1

// outdoorLootTableFor は屋外ゾーンに応じた item テーブル id を返す。道沿いは人の営みの残りで廃墟テーブル、
// 奥地は野外の森テーブルから引く。exhaustive linter が outdoorZone の網羅を強制するので default を置かず、
// 末尾 panic で未知ゾーンのランタイム保護も兼ねる。
func outdoorLootTableFor(zone outdoorZone) string {
	switch zone {
	case zoneRoadside:
		return "ruins_area"
	case zoneWild:
		return "forest"
	}
	panic("unknown outdoorZone: " + string(zone))
}

// scatterOutdoorLoot は開けたチャンクの地表へ拾える物を疎に撒く。候補は土系の地面かつ非占有で、accept と
// occupied は草・樹木の散布と共有して重なりを避ける。乱数は散布とは別の salt から引き、地物や建物 loot の
// 消費順と干渉させない。同一 seed で同一結果、再訪で一致する。
func scatterOutdoorLoot(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], g chunkGeom, cat scatterCatalog, accept func(interior.Vec) bool, occupied map[gc.GridElement]bool) error {
	tableID := outdoorLootTableFor(cat.Zone)
	itemTable, err := raw.GetItemTable(world.Resources.RawMaster, tableID)
	if err != nil {
		return err
	}

	area := interior.Rect{X: 0, Y: 0, W: g.chunkW, H: g.chunkH}
	selSeed := ChunkSeed2D(runSeed^outdoorLootSalt, c.X, c.Y)
	rng := rand.New(rand.NewPCG(selSeed, outdoorLootSalt))
	count := int(math.Round(float64(g.chunkW) * float64(g.chunkH) * outdoorLootDensity))

	for _, rel := range interior.ScatterArea(area, accept, selSeed, count) {
		itemName, err := raw.SelectItemByWeight(world.Resources.RawMaster, itemTable, rng, outdoorLootDepth)
		if err != nil {
			return err
		}
		if itemName == "" {
			continue
		}
		bl := consts.Coord[consts.Tile]{X: g.offsetX + rel.X, Y: g.offsetY + rel.Y}
		if _, err := lifecycle.SpawnFieldItem(world, itemName, bl.X, bl.Y, 1); err != nil {
			return fmt.Errorf("failed to place outdoor loot (%s): %w", itemName, err)
		}
		occupied[gc.GridElement{Coord: bl}] = true
	}
	return nil
}
