package raw

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// itemUse は物の使い道の1つ。統計の表にそのまま出す
type itemUse string

// itemUseCheck は使い道と、物がそれを持つかの判定の組。並びの順に表へ出す
type itemUseCheck struct {
	use itemUse
	has bool
}

// itemUseChecks は物 it の使い道の判定を表に並べる順で返す。燃料と解体は素材の表から、作成の材料はレシピから、
// 性質の使い道は性質の行から決まる。使い道を足すときはここへ1行足すだけで、判定と並び順が揃う
func itemUseChecks(it *rawschema.Item, material rawschema.MaterialDef, recipeInputs, propertyRemedies map[rawschema.EntityID]bool) []itemUseCheck {
	eat := it.ProvidesNutrition != nil
	heal := it.ProvidesHealing != nil || it.Remedy != nil || slices.ContainsFunc(PtrSlice(it.Properties), func(id rawschema.EntityID) bool {
		return propertyRemedies[id]
	})
	throw := it.InflictsDamage != nil
	escape := it.EscapeDevice != nil
	core := it.CubeCore != nil
	return []itemUseCheck{
		{"着る", it.Wearable != nil},
		{"近接", it.Melee != nil},
		{"射撃", it.Fire != nil},
		{"弾", it.Ammo != nil},
		{"食べる", eat},
		{"治す", heal},
		{"投げる", throw},
		{"使う", it.Consumable != nil && !eat && !heal && !throw && !escape && !core},
		{"読む", it.Book != nil},
		{"道具", it.Tool != nil},
		{"設置", it.Deployable != nil},
		{"着火", it.FireStarter != nil},
		{"照らす", it.LightSource != nil},
		{"脱出", escape},
		{"キューブの部品", it.CubeModule != nil},
		{"キューブの核", core},
		{"燃料", material.HeatPerKg > 0},
		{"解体", material.Disassembly != nil},
		{"作成の材料", recipeInputs[it.Id]},
	}
}

// itemUsesReport は物の使い道の統計を Markdown の表にする
func itemUsesReport(raws rawschema.Raws) (string, error) {
	materials := map[rawschema.Material]rawschema.MaterialDef{}
	for _, m := range PtrSlice(raws.Materials) {
		materials[m.Id] = m
	}
	recipeInputs := map[rawschema.EntityID]bool{}
	for _, r := range PtrSlice(raws.Recipes) {
		for _, in := range r.Inputs {
			recipeInputs[in.Id] = true
		}
	}

	propertyRemedies := map[rawschema.EntityID]bool{}
	for _, p := range PtrSlice(raws.ItemProperties) {
		propertyRemedies[p.Id] = p.Remedy != nil
	}

	items := PtrSlice(raws.Items)
	byCount := map[int]int{}
	byUse := map[itemUse]int{}
	byCombo := map[string]int{}
	noUse := []string{}
	columns := itemUseChecks(&rawschema.Item{}, rawschema.MaterialDef{}, nil, nil)
	order := make([]itemUse, 0, len(columns))
	for _, c := range columns {
		order = append(order, c.use)
	}
	for i := range items {
		var material rawschema.MaterialDef
		if items[i].Material != nil {
			m, ok := materials[*items[i].Material]
			if !ok {
				return "", fmt.Errorf("item %s: unknown material %s", items[i].Id, *items[i].Material)
			}
			material = m
		}
		checks := itemUseChecks(&items[i], material, recipeInputs, propertyRemedies)
		names := []string{}
		for _, c := range checks {
			if c.has {
				byUse[c.use]++
				names = append(names, string(c.use))
			}
		}
		byCount[len(names)]++
		byCombo[strings.Join(names, "・")]++
		if len(names) == 0 {
			noUse = append(noUse, items[i].Id)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# 物の使い道\n\n物は %d 種類。燃料と解体は素材の表で、作成の材料はレシピで決まる。\n\n", len(items))

	b.WriteString("## 使い道の数\n\n| 使い道の数 | 物の数 |\n|---|---|\n")
	for _, n := range slices.Sorted(maps.Keys(byCount)) {
		fmt.Fprintf(&b, "| %d | %d |\n", n, byCount[n])
	}

	b.WriteString("\n## 使い道ごとの物の数\n\n| 使い道 | 物の数 |\n|---|---|\n")
	for _, u := range order {
		fmt.Fprintf(&b, "| %s | %d |\n", u, byUse[u])
	}

	b.WriteString("\n## 使い道の組み合わせ\n\n| 物の数 | 組み合わせ |\n|---|---|\n")
	combos := slices.SortedFunc(maps.Keys(byCombo), func(x, y string) int {
		return cmp.Or(cmp.Compare(byCombo[y], byCombo[x]), cmp.Compare(x, y))
	})
	for _, c := range combos {
		label := c
		if label == "" {
			label = "なし"
		}
		fmt.Fprintf(&b, "| %d | %s |\n", byCombo[c], label)
	}

	b.WriteString("\n## 使い道の無い物\n\n")
	slices.Sort(noUse)
	for _, id := range noUse {
		fmt.Fprintf(&b, "- %s\n", id)
	}
	return b.String(), nil
}

func TestGolden_ItemUses(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	report, err := itemUsesReport(raws)
	require.NoError(t, err)

	g := goldie.New(t, goldie.WithFixtureDir("."), goldie.WithNameSuffix(".golden.md"))
	g.Assert(t, "item_uses", []byte(report))
}
