package raw

import (
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/stretchr/testify/assert"
)

// craftableFrom は have の素材から、レシピを繰り返し作って得られる物の id を作った順に返す
func craftableFrom(raws rawschema.Raws, have map[string]bool) []string {
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

// toolCategoriesOf は表 table から出うるアイテムの道具の分類を返す
func toolCategoriesOf(raws rawschema.Raws, table string) []rawschema.ToolCategory {
	all := PtrSlice(raws.Items)
	items := map[string]*rawschema.Item{}
	for i := range all {
		items[all[i].Id] = &all[i]
	}
	var ids []string
	for _, tb := range PtrSlice(raws.ItemTables) {
		if tb.Id != table {
			continue
		}
		for _, e := range tb.Entries {
			for _, g := range PtrSlice(raws.ItemGroups) {
				if g.Id == e.Id {
					for _, ge := range g.Entries {
						ids = append(ids, ge.Id)
					}
				}
			}
			for _, c := range PtrSlice(raws.ItemCollections) {
				if c.Id == e.Id {
					for _, ce := range c.Entries {
						ids = append(ids, ce.Id)
					}
				}
			}
		}
	}
	var cats []rawschema.ToolCategory
	for _, id := range ids {
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
