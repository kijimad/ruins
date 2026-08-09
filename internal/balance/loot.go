package balance

import (
	"math/rand/v2"
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// FacilityLoot は1つの施設種別を trials 回生成し、床 loot と収納 loot を集計した分布。
type FacilityLoot struct {
	Facility string         `json:"facility"`
	Trials   int            `json:"trials"`
	Items    []LootItemStat `json:"items"`
}

// LootItemStat は1アイテムの出現統計。Prob は1棟あたり1個以上出る確率、ExpectedCount は1棟あたりの期待個数。
type LootItemStat struct {
	Name          string  `json:"name"`
	Prob          float64 `json:"prob"`
	ExpectedCount float64 `json:"expectedCount"`
	Value         int     `json:"value"`
}

const (
	// roomLootDepth は地上の建物の深度。床 loot も収納 loot も浅い戦利品を引く。
	roomLootDepth = 1
	// roomLootStream は施設 loot 抽選 RNG のストリーム。試行 seed と組で決定的にする。
	roomLootStream = 0x1007
)

// reportFacilities は集計対象の施設種別。interior の FacilityKind 定数と同じ文字列。
var reportFacilities = []interior.FacilityKind{"house", "store", "antique", "clinic", "lab", "office", "depot"}

// GenerateRoomLoot は各施設種別を trials 回生成し、床 loot と収納 loot を実際の抽選経路で materialize して
// アイテム別の出現確率と期待個数を集計する。解析でなくサンプリングで、PickN・Amount・pack・lootRaw・収納
// テーブルの相互作用をそのまま反映する。同一 seed で同一結果になる。
func GenerateRoomLoot(master oapi.Raws, trials int, seed uint64) []FacilityLoot {
	footprint := interior.Rect{X: 0, Y: 0, W: 28, H: 20}
	door := interior.Vec{X: 14, Y: 0}

	result := make([]FacilityLoot, 0, len(reportFacilities))
	for _, fac := range reportFacilities {
		total := map[string]int{}   // アイテム名 -> 全試行の合計個数
		present := map[string]int{} // アイテム名 -> 1個以上出た試行数
		for i := range trials {
			trialSeed := seed + uint64(i)
			rng := rand.New(rand.NewPCG(trialSeed, roomLootStream))
			_, placed := interior.FurnishBuilding(trialSeed, footprint, door, fac)

			counts := map[string]int{}
			for _, p := range placed {
				switch p.Kind {
				case interior.KindLoot:
					// 床 loot。抽象 Ref を item group へ写し、実行と同じ抽選で山を materialize する
					groupID, ok := interior.LootGroupName(p.Ref)
					if !ok {
						continue
					}
					draws, err := raw.SelectFromItemGroup(master, groupID, rng)
					if err != nil {
						continue
					}
					for _, d := range draws {
						if d.Name != "" {
							counts[d.Name] += d.Count
						}
					}
				case interior.KindFurniture:
					// 収納家具。raw の Storage.LootTableId から中身を引く
					name, ok := interior.PropRawName(p.Ref)
					if !ok {
						continue
					}
					for _, item := range rollStorageLoot(master, name, rng) {
						counts[item]++
					}
				default:
					// 装飾・敵・罠などは loot 集計の対象外
				}
			}
			for name, c := range counts {
				total[name] += c
				present[name]++
			}
		}

		items := make([]LootItemStat, 0, len(total))
		for name, c := range total {
			items = append(items, LootItemStat{
				Name:          name,
				Prob:          float64(present[name]) / float64(trials),
				ExpectedCount: float64(c) / float64(trials),
				Value:         itemValue(master, name),
			})
		}
		sort.Slice(items, func(a, b int) bool {
			if items[a].ExpectedCount != items[b].ExpectedCount {
				return items[a].ExpectedCount > items[b].ExpectedCount
			}
			return items[a].Name < items[b].Name
		})
		result = append(result, FacilityLoot{Facility: string(fac), Trials: trials, Items: items})
	}
	return result
}

// rollStorageLoot は収納家具1個の中身を抽選し、アイテム名を並べて返す。ゲームの populateStorageLoot と同型で、
// LootTableId のテーブルから LootCount 個を深度考慮で引く。収納でない家具は空を返す。
func rollStorageLoot(master oapi.Raws, propName string, rng *rand.Rand) []string {
	prop, err := raw.GetProp(master, propName)
	if err != nil || prop.Storage == nil || prop.Storage.LootTableId == nil || *prop.Storage.LootTableId == "" {
		return nil
	}
	table, err := raw.GetItemTable(master, *prop.Storage.LootTableId)
	if err != nil {
		return nil
	}
	n := 1
	if prop.Storage.LootCount != nil {
		if d, derr := consts.ParseDice(*prop.Storage.LootCount); derr == nil {
			n = d.Roll(rng)
		}
	}
	out := make([]string, 0, n)
	for range n {
		itemName, serr := raw.SelectItemByWeight(master, table, rng, roomLootDepth)
		if serr != nil || itemName == "" {
			continue
		}
		out = append(out, itemName)
	}
	return out
}

// itemValue は raw のアイテム価値を返す。未定義は0。
func itemValue(master oapi.Raws, name string) int {
	item, err := raw.FindItem(master, name)
	if err != nil {
		return 0
	}
	return int(item.Value)
}
