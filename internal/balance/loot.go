package balance

import (
	"math/rand/v2"
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
)

// FacilityLoot は1つの施設種別を trials 回生成し、部屋役割ごとに loot を集計した分布。
type FacilityLoot struct {
	Facility string     `json:"facility"`
	Trials   int        `json:"trials"`
	Rooms    []RoomLoot `json:"rooms"`
}

// RoomLoot は施設内の1役割ぶんの loot 分布。Role が "全体" のときは建物全体の合算。
type RoomLoot struct {
	Role  string         `json:"role"`
	Items []LootItemStat `json:"items"`
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
	// roomLootAllRole は建物全体の合算を表す役割キー。表示名は UI 側で訳す。
	roomLootAllRole = "all"
	// roomLootOtherRole はどの部屋にも属さない配置の役割キー。表示名は UI 側で訳す。
	roomLootOtherRole = "other"
)

// reportFacilities は集計対象の施設種別。interior の FacilityKind 定数と同じ文字列。
var reportFacilities = []interior.FacilityKind{"house", "store", "antique", "clinic", "lab", "office", "depot"}

// GenerateRoomLoot は各施設種別を trials 回生成し、床 loot と収納 loot を実際の抽選経路で materialize して
// 部屋役割ごとにアイテム別の出現確率と期待個数を集計する。解析でなくサンプリングで、PickN・Amount・pack・
// lootRaw・収納テーブルの相互作用をそのまま反映する。同一 seed で同一結果になる。
func GenerateRoomLoot(master oapi.Raws, trials int, seed uint64) []FacilityLoot {
	footprint := interior.Rect{X: 0, Y: 0, W: 28, H: 20}
	door := interior.Vec{X: 14, Y: 0}

	result := make([]FacilityLoot, 0, len(reportFacilities))
	for _, fac := range reportFacilities {
		// role -> item -> 合計個数 / 出た試行数
		total := map[string]map[string]int{}
		present := map[string]map[string]int{}
		add := func(role, name string, count int) {
			if total[role] == nil {
				total[role] = map[string]int{}
				present[role] = map[string]int{}
			}
			total[role][name] += count
		}

		for i := range trials {
			trialSeed := seed + uint64(i)
			rng := rand.New(rand.NewPCG(trialSeed, roomLootStream))
			site, placed := interior.FurnishBuilding(trialSeed, footprint, door, fac)

			// この試行の role -> item -> 個数。試行内で集めてから present を1回だけ加算する
			perRole := map[string]map[string]int{}
			bump := func(role, name string, count int) {
				if perRole[role] == nil {
					perRole[role] = map[string]int{}
				}
				perRole[role][name] += count
			}
			for _, p := range placed {
				role := roomRoleAt(site, p.Pos)
				for _, it := range resolvePlacedLoot(master, p, rng) {
					bump(role, it.name, it.count)
					bump(roomLootAllRole, it.name, it.count) // 建物全体の合算
				}
			}
			for role, items := range perRole {
				for name, c := range items {
					add(role, name, c)
					present[role][name]++
				}
			}
		}

		result = append(result, FacilityLoot{
			Facility: string(fac),
			Trials:   trials,
			Rooms:    buildRoomLoot(master, total, present, trials),
		})
	}
	return result
}

// drawnLoot は materialize した1種のアイテムと個数。
type drawnLoot struct {
	name  string
	count int
}

// resolvePlacedLoot は1つの配置指示を実行と同じ経路で materialize してアイテムと個数を返す。床 loot は
// item group から、収納家具は Storage.LootTableId から引く。それ以外は何も返さない。
func resolvePlacedLoot(master oapi.Raws, p interior.Placed, rng *rand.Rand) []drawnLoot {
	switch p.Kind {
	case interior.KindLoot:
		groupID, ok := interior.LootGroupName(p.Ref)
		if !ok {
			return nil
		}
		draws, err := raw.SelectFromItemGroup(master, groupID, rng)
		if err != nil {
			return nil
		}
		out := make([]drawnLoot, 0, len(draws))
		for _, d := range draws {
			if d.Name != "" {
				out = append(out, drawnLoot{name: d.Name, count: d.Count})
			}
		}
		return out
	case interior.KindFurniture:
		name, ok := interior.PropRawName(p.Ref)
		if !ok {
			return nil
		}
		items := rollStorageLoot(master, name, rng)
		out := make([]drawnLoot, 0, len(items))
		for _, item := range items {
			out = append(out, drawnLoot{name: item, count: 1})
		}
		return out
	default:
		// 装飾・敵・罠などは loot でない
		return nil
	}
}

// roomRoleAt は座標 pos を含む部屋の役割を返す。どの部屋にも属さなければ "その他"。loot は部屋の内側に置かれる
// ので、外構や外皮の装飾以外は必ずいずれかの部屋に収まる。
func roomRoleAt(site interior.Site, pos interior.Vec) string {
	for _, r := range site.Rooms {
		rect := r.Room.Rect
		if pos.X >= rect.X && pos.X < rect.X+rect.W && pos.Y >= rect.Y && pos.Y < rect.Y+rect.H {
			role := string(r.Role)
			if role == "" {
				return roomLootOtherRole
			}
			return role
		}
	}
	return roomLootOtherRole
}

// buildRoomLoot は集計 map を役割ごとの RoomLoot へ整える。全体を先頭に、以降は期待個数の合計が多い役割順に並べる。
func buildRoomLoot(master oapi.Raws, total, present map[string]map[string]int, trials int) []RoomLoot {
	rooms := make([]RoomLoot, 0, len(total))
	for role, items := range total {
		stats := make([]LootItemStat, 0, len(items))
		for name, c := range items {
			stats = append(stats, LootItemStat{
				Name:          name,
				Prob:          float64(present[role][name]) / float64(trials),
				ExpectedCount: float64(c) / float64(trials),
				Value:         itemValue(master, name),
			})
		}
		sort.Slice(stats, func(a, b int) bool {
			if stats[a].ExpectedCount != stats[b].ExpectedCount {
				return stats[a].ExpectedCount > stats[b].ExpectedCount
			}
			return stats[a].Name < stats[b].Name
		})
		rooms = append(rooms, RoomLoot{Role: role, Items: stats})
	}
	sort.Slice(rooms, func(a, b int) bool {
		// 全体を必ず先頭にする
		if rooms[a].Role == roomLootAllRole {
			return true
		}
		if rooms[b].Role == roomLootAllRole {
			return false
		}
		return roomTotalExpected(rooms[a]) > roomTotalExpected(rooms[b])
	})
	return rooms
}

// roomTotalExpected は役割の期待個数合計。役割の並び順に使う。
func roomTotalExpected(r RoomLoot) float64 {
	var sum float64
	for _, it := range r.Items {
		sum += it.ExpectedCount
	}
	return sum
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
