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

// 屋外の散布 loot は開けた地形へ拾える物をごくまばらに撒く。建物内の役割別 loot と違い、屋外は屑物だけを
// 置く。屋外に武器・防具・回復薬のような価値ある品を撒くと拾い放題になりバランスを壊し、原野に装備が
// 落ちている絵面も不自然なので、価値の低い item group からだけ引く。道沿いは人の営みの残りの紙屑、奥地は
// 自然に転がる廃材・鉱片。密度も低くして原野を埋め尽くさない。土系の地面かつ非占有のタイルにだけ置く。

// outdoorLootDensity は屋外散布 loot の面積あたり係数。count = round(area * density) で置くタイル数を密度から
// 導く。反復乱数でなく密度で決めるので純関数のまま。屋外の loot は稀であるべきなので低く抑える。1チャンク
// 約600タイルで期待 1 タイル強に相当する。屑物 group 側も低確率なので、実際に物が出るのはさらに疎になる。
const outdoorLootDensity = 0.002

// outdoorLootItemChannel はアイテム抽選を座標選択と無相関にするハッシュチャネル。ScatterArea のタイル選択
// seed と別チャネルにし、どのタイルに何が出るかの相関を切る。scatter.go の grass/weed チャネルと同じ考え方。
const outdoorLootItemChannel uint64 = 0x6c6f6f745f69746d // "loot_itm"

// outdoorLootGroupFor は屋外ゾーンに応じた低価値な item group を返す。屋外には屑物だけを置き、武器・防具・
// 回復薬のような価値ある品は建物内の loot に限る。道沿いは人の営みの残りの紙屑、奥地は自然に転がる廃材・鉱片。
// exhaustive linter が outdoorZone の網羅を強制するので default を置かず、末尾 panic で未知ゾーンのランタイム
// 保護も兼ねる。
func outdoorLootGroupFor(zone outdoorZone) string {
	switch zone {
	case zoneRoadside:
		return "scrap_of_paper"
	case zoneWild:
		return "materials"
	}
	panic("unknown outdoorZone: " + string(zone))
}

// scatterOutdoorLoot は開けたチャンクの地表へ屑物をごく疎に撒く。候補は土系の地面かつ非占有で、accept と
// occupied は草・樹木の散布と共有して重なりを避ける。抽選は建物 loot と同じ raw.SelectFromItemGroup で低価値
// group から引き、深度は扱わない。乱数は散布とは別の salt から引き、地物や建物 loot の消費順と干渉させない。
// 同一 seed で同一結果、再訪で一致する。
func scatterOutdoorLoot(world w.World, runSeed uint64, c consts.Coord[consts.Chunk], g chunkGeom, cat scatterCatalog, accept func(interior.Vec) bool, occupied map[gc.GridElement]bool) error {
	groupID := outdoorLootGroupFor(cat.Zone)

	area := interior.Rect{X: 0, Y: 0, W: g.chunkW, H: g.chunkH}
	// タイル選択の seed とアイテム抽選の rng を別チャネルにする。同一 seed だと「どこに置くか」と「何を置くか」が
	// 相関しうるので、アイテム側は専用チャネルを XOR して無相関にする
	selSeed := ChunkSeed2D(runSeed^outdoorLootSalt, c.X, c.Y)
	rng := rand.New(rand.NewPCG(selSeed^outdoorLootItemChannel, outdoorLootSalt))
	count := int(math.Round(float64(g.chunkW) * float64(g.chunkH) * outdoorLootDensity))

	for _, rel := range interior.ScatterArea(area, accept, selSeed, count) {
		draws, err := raw.SelectFromItemGroup(world.Resources.RawMaster, groupID, rng)
		if err != nil {
			return err
		}
		if len(draws) == 0 {
			continue // 屑物 group は確率が低く、何も出ないタイルが多い。そのタイルは空けておく
		}
		bl := consts.Coord[consts.Tile]{X: g.offsetX + rel.X, Y: g.offsetY + rel.Y}
		for _, d := range draws {
			if d.Name == "" {
				continue
			}
			if _, err := lifecycle.SpawnFieldItem(world, d.Name, bl.X, bl.Y, d.Count); err != nil {
				return fmt.Errorf("failed to place outdoor loot (%s): %w", d.Name, err)
			}
		}
		occupied[gc.GridElement{Coord: bl}] = true
	}
	return nil
}
