package balance

import (
	"fmt"
	"math/rand/v2"
	"sort"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
)

const (
	// facilityLootDepth は地上の建物の深度。収納 loot は浅い戦利品を引く。
	facilityLootDepth = 1
	// facilityLootStream は施設 loot 抽選 RNG のストリーム。試行 seed と組で決定的にする。
	facilityLootStream = 0x1007
	// facilityLootAll は建物全体の合算を表すキー。
	facilityLootAll = "all"
)

// GenerateFacilityLoot は各施設の絵を trials 回展開し、収納と床の loot を実際の抽選経路で materialize して、絵ごとに
// アイテム別の出現確率と期待個数を集計する。解析でなくサンプリングで抽選の相互作用をそのまま反映する。同一 seed で同一結果。
// 集計対象の施設は raw.toml の facilities 行から導出する。
func GenerateFacilityLoot(master rawschema.Raws, trials int, seed uint64) ([]FacilityLoot, error) {
	loader, err := maptemplate.FacilityTemplates()
	if err != nil {
		return nil, err
	}

	facilities := raw.PtrSlice(master.Facilities)
	result := make([]FacilityLoot, 0, len(facilities))
	for _, f := range facilities {
		fac := f.Id
		// 絵 -> item -> 合計個数 / 出た試行数
		total := map[string]map[string]int{}
		present := map[string]map[string]int{}
		picked := map[string]int{facilityLootAll: trials}

		for i := range trials {
			trialSeed := seed + uint64(i)
			rng := rand.New(rand.NewPCG(trialSeed, facilityLootStream))
			chunk, cells, err := loader.FacilityChunk(fac, trialSeed)
			if err != nil {
				return nil, fmt.Errorf("facility %q: %w", fac, err)
			}
			picked[chunk.Name]++

			// この試行の item -> 個数。試行内で集めてから present を1回だけ加算する
			perTrial := map[string]int{}
			for _, row := range cells {
				for _, c := range row {
					drawn, err := resolveCellLoot(master, c, rng)
					if err != nil {
						return nil, fmt.Errorf("facility %q %s: %w", fac, chunk.Name, err)
					}
					for _, it := range drawn {
						perTrial[it.name] += it.count
					}
				}
			}
			for _, key := range []string{chunk.Name, facilityLootAll} {
				if total[key] == nil {
					total[key] = map[string]int{}
					present[key] = map[string]int{}
				}
				for name, n := range perTrial {
					total[key][name] += n
					present[key][name]++
				}
			}
		}

		result = append(result, FacilityLoot{
			Facility:  fac,
			Trials:    trials,
			Templates: buildTemplateLoot(master, total, present, picked),
		})
	}
	return result, nil
}

// drawnLoot は materialize した1種のアイテムと個数。
type drawnLoot struct {
	name  string
	count int
}

// resolveCellLoot は施設の絵の1マスの収納の中身と床の戦利品を、実行と同じ表から引いて返す。表が引けなければ error を返す。
func resolveCellLoot(master rawschema.Raws, c maptemplate.MapCell, rng *rand.Rand) ([]drawnLoot, error) {
	var out []drawnLoot
	if c.Prop != "" && c.LootTable != "" {
		stored, err := rollStorageLoot(master, c.Prop, c.LootTable, rng)
		if err != nil {
			return nil, err
		}
		out = append(out, stored...)
	}
	for range c.ItemCount {
		draws, err := raw.SelectFromItemGroup(master, c.ItemGroup, rng)
		if err != nil {
			return nil, err
		}
		for _, d := range draws {
			if d.Name != "" {
				out = append(out, drawnLoot{name: d.Name, count: d.Count})
			}
		}
	}
	return out, nil
}

// buildTemplateLoot は集計 map を絵ごとの TemplateLoot へ整える。全体を先頭に、以降は期待個数の合計が多い絵の順に並べる。
// 確率と期待個数は、その絵が選ばれた試行数で割る。
func buildTemplateLoot(master rawschema.Raws, total, present map[string]map[string]int, picked map[string]int) []TemplateLoot {
	templates := make([]TemplateLoot, 0, len(total))
	for name, items := range total {
		n := picked[name]
		stats := make([]LootItemStat, 0, len(items))
		for item, c := range items {
			stats = append(stats, LootItemStat{
				Name:          item,
				Prob:          float64(present[name][item]) / float64(n),
				ExpectedCount: float64(c) / float64(n),
				Value:         itemValue(master, item),
			})
		}
		sort.Slice(stats, func(a, b int) bool {
			if stats[a].ExpectedCount != stats[b].ExpectedCount {
				return stats[a].ExpectedCount > stats[b].ExpectedCount
			}
			return stats[a].Name < stats[b].Name
		})
		templates = append(templates, TemplateLoot{Template: name, Trials: n, Items: stats})
	}
	sort.Slice(templates, func(a, b int) bool {
		if templates[a].Template == facilityLootAll {
			return true
		}
		if templates[b].Template == facilityLootAll {
			return false
		}
		if ea, eb := templateTotalExpected(templates[a]), templateTotalExpected(templates[b]); ea != eb {
			return ea > eb
		}
		return templates[a].Template < templates[b].Template
	})
	return templates
}

// templateTotalExpected は絵の期待個数の合計。絵の並び順に使う。
func templateTotalExpected(t TemplateLoot) float64 {
	var sum float64
	for _, it := range t.Items {
		sum += it.ExpectedCount
	}
	return sum
}

// rollStorageLoot は収納家具1個の中身を抽選する。ゲームの populateStorageLoot と同型で、パレットが書いた表 tableID から
// LootCount 回を深度考慮で引く。荒らされ率は掛けない。収納でない家具は空を返す。
func rollStorageLoot(master rawschema.Raws, propName, tableID string, rng *rand.Rand) ([]drawnLoot, error) {
	prop, err := raw.GetProp(master, propName)
	if err != nil {
		return nil, err
	}
	if prop.Storage == nil {
		return nil, nil
	}
	table, err := raw.GetItemTable(master, tableID)
	if err != nil {
		return nil, err
	}
	n := 1
	if prop.Storage.LootCount != nil {
		d, err := consts.ParseDice(*prop.Storage.LootCount)
		if err != nil {
			return nil, fmt.Errorf("lootCount of %q: %w", propName, err)
		}
		n = d.Roll(rng)
	}
	var out []drawnLoot
	for range n {
		draws, err := raw.DrawFromItemTable(master, table, rng, facilityLootDepth)
		if err != nil {
			return nil, err
		}
		for _, d := range draws {
			out = append(out, drawnLoot{name: d.Name, count: d.Count})
		}
	}
	return out, nil
}

// itemValue は raw のアイテム価値を返す。未定義は0。
func itemValue(master rawschema.Raws, name string) int {
	item, err := raw.FindItem(master, name)
	if err != nil {
		return 0
	}
	return item.Value
}
