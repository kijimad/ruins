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

// itemUseOrder は使い道を表に並べる順
var itemUseOrder = []itemUse{
	"着る", "近接", "射撃", "弾", "食べる・使う", "読む", "道具", "設置", "着火", "照らす", "脱出", "キューブの部品", "キューブの核",
	"燃料", "解体", "作成の材料",
}

// itemUsesOf は物 it の使い道を itemUseOrder の順に返す。燃料と解体は素材の表から、作成の材料はレシピから決まる
func itemUsesOf(it *rawschema.Item, materials map[rawschema.Material]rawschema.MaterialDef, recipeInputs map[rawschema.EntityID]bool) []itemUse {
	has := map[itemUse]bool{
		"着る":      it.Wearable != nil,
		"近接":      it.Melee != nil,
		"射撃":      it.Fire != nil,
		"弾":       it.Ammo != nil,
		"食べる・使う":  it.Consumable != nil || it.ProvidesNutrition != nil || it.ProvidesHealing != nil || it.Remedy != nil,
		"読む":      it.Book != nil,
		"道具":      it.Tool != nil,
		"設置":      it.Deployable != nil,
		"着火":      it.FireStarter != nil,
		"照らす":     it.LightSource != nil,
		"脱出":      it.EscapeDevice != nil,
		"キューブの部品": it.CubeModule != nil,
		"キューブの核":  it.CubeCore != nil,
		"作成の材料":   recipeInputs[it.Id],
	}
	if it.Material != nil {
		m := materials[*it.Material]
		has["燃料"] = m.HeatPerKg > 0
		has["解体"] = m.Disassembly != nil
	}
	uses := []itemUse{}
	for _, u := range itemUseOrder {
		if has[u] {
			uses = append(uses, u)
		}
	}
	return uses
}

// itemUsesReport は物の使い道の統計を Markdown の表にする
func itemUsesReport(raws rawschema.Raws) string {
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

	items := PtrSlice(raws.Items)
	byCount := map[int]int{}
	byUse := map[itemUse]int{}
	byCombo := map[string]int{}
	noUse := []string{}
	for i := range items {
		uses := itemUsesOf(&items[i], materials, recipeInputs)
		byCount[len(uses)]++
		for _, u := range uses {
			byUse[u]++
		}
		names := make([]string, len(uses))
		for j, u := range uses {
			names[j] = string(u)
		}
		byCombo[strings.Join(names, "・")]++
		if len(uses) == 0 {
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
	for _, u := range itemUseOrder {
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
	return b.String()
}

func TestGolden_ItemUses(t *testing.T) {
	t.Parallel()

	raws, err := LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)

	g := goldie.New(t, goldie.WithFixtureDir("."), goldie.WithNameSuffix(".golden.md"))
	g.Assert(t, "item_uses", []byte(itemUsesReport(raws)))
}
