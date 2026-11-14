package raw

import (
	"maps"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// craftableFrom は have の素材から、レシピを繰り返し作って得られる物の id を作った順に返す
func craftableFrom(raws rawschema.Raws, materials map[string]bool) []string {
	have := maps.Clone(materials)
	var made []string
	for changed := true; changed; {
		changed = false
		for _, r := range PtrSlice(raws.Recipes) {
			missing := slices.ContainsFunc(r.Inputs, func(in rawschema.RecipeInput) bool { return !have[in.Id] })
			if have[r.Id] || missing {
				continue
			}
			have[r.Id] = true
			made = append(made, r.Id)
			changed = true
		}
	}
	return made
}

func TestRecipes_道具の分類が無い分解の素材から木刀まで作れる(t *testing.T) {
	t.Parallel()
	raws := loadTestRaws(t)

	have := map[string]bool{}
	for _, p := range PtrSlice(raws.Props) {
		if p.Disassembly == nil || p.Disassembly.ToolCategory != nil {
			continue
		}
		for _, y := range p.Disassembly.Yields {
			have[y.Id] = true
		}
	}

	assert.Equal(t, []string{"wooden_stick", "wooden_sword"}, craftableFrom(raws, have))
}

// tableItemIDs は表 table から出うるアイテムの id を返す
func tableItemIDs(raws rawschema.Raws, table string) []string {
	members := map[string][]string{}
	for _, g := range PtrSlice(raws.ItemGroups) {
		for _, e := range g.Entries {
			members[g.Id] = append(members[g.Id], e.Id)
		}
	}
	for _, c := range PtrSlice(raws.ItemCollections) {
		for _, e := range c.Entries {
			members[c.Id] = append(members[c.Id], e.Id)
		}
	}
	var ids []string
	for _, tb := range PtrSlice(raws.ItemTables) {
		if tb.Id != table {
			continue
		}
		for _, e := range tb.Entries {
			ids = append(ids, members[e.Id]...)
		}
	}
	return ids
}

// toolCategoriesOf は表 table から出うるアイテムの道具の分類を返す
func toolCategoriesOf(raws rawschema.Raws, table string) []rawschema.ToolCategory {
	all := PtrSlice(raws.Items)
	items := map[string]*rawschema.Item{}
	for i := range all {
		items[all[i].Id] = &all[i]
	}
	var cats []rawschema.ToolCategory
	for _, id := range tableItemIDs(raws, table) {
		if it, ok := items[id]; ok && it.Tool != nil {
			cats = append(cats, it.Tool.Categories...)
		}
	}
	slices.Sort(cats)
	return slices.Compact(cats)
}

func TestItemTables_施設の表が序盤の道具を出しうる(t *testing.T) {
	t.Parallel()
	raws := loadTestRaws(t)

	tests := []struct {
		table string
		want  []rawschema.ToolCategory
	}{
		{"house_kitchen", []rawschema.ToolCategory{rawschema.Chopping}},
		{"house_closet", []rawschema.ToolCategory{rawschema.Chopping, rawschema.Mining, rawschema.Prying}},
		{"depot_storage", []rawschema.ToolCategory{rawschema.Chopping, rawschema.Mining, rawschema.Prying}},
	}
	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, toolCategoriesOf(raws, tt.table))
		})
	}
}

func TestItemTables_研究所の機密ケースの表から脱出の装置を直せる(t *testing.T) {
	t.Parallel()
	raws := loadTestRaws(t)

	parts := tableItemIDs(raws, "classified_case")
	assert.ElementsMatch(t, []string{"broken_escape_core", "signal_amplifier", "power_cell"}, parts)

	have := map[string]bool{"electric_wire": true}
	for _, id := range parts {
		have[id] = true
	}
	assert.Equal(t, []string{"escape_core"}, craftableFrom(raws, have))
}

func TestRecipes_布と革の製品を分解した素材から包帯と革のブーツを作れる(t *testing.T) {
	t.Parallel()
	raws := loadTestRaws(t)
	defs, err := NewMaterialDefs(raws)
	require.NoError(t, err)

	type yield struct {
		id    string
		count int
	}
	got := map[string]yield{}
	for _, item := range []string{"cloth_shirt", "leather_gloves"} {
		spec, err := NewItemSpec(raws, item)
		require.NoError(t, err)
		id, count, ok := defs.DisassemblyOf(item, spec.Material.Kind, spec.Weight.Milligram)
		require.True(t, ok)
		got[item] = yield{id, count}
	}
	assert.Equal(t, map[string]yield{"cloth_shirt": {"rag", 3}, "leather_gloves": {"leather_scrap", 2}}, got)

	assert.Equal(t, []string{"leather_boots", "bandage"}, craftableFrom(raws, map[string]bool{"rag": true, "leather_scrap": true}))
}
