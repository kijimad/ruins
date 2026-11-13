package maptemplate

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPalettes_rawの行を参照するidは全て実在する は、全てのパレットが書いた id を raw の行と部品の名前に突き合わせる。
// 重みの小さい家具の候補は seed によっては選ばれず、生成のテストでは書き間違いが見つからないので、候補を全て見る
func TestPalettes_rawの行を参照するidは全て実在する(t *testing.T) {
	t.Parallel()

	raws, err := raw.LoadFromFile("metadata/entities/raw/raw.toml")
	require.NoError(t, err)
	loader, err := FacilityTemplates()
	require.NoError(t, err)

	tileIDs, propIDs, memberIDs, tableIDs, groupIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, x := range raw.PtrSlice(raws.Tiles) {
		tileIDs[x.Id] = true
	}
	for _, x := range raw.PtrSlice(raws.Props) {
		propIDs[x.Id] = true
	}
	for _, x := range raw.PtrSlice(raws.Members) {
		memberIDs[x.Id] = true
	}
	for _, x := range raw.PtrSlice(raws.ItemTables) {
		tableIDs[x.Id] = true
	}
	// アイテムグループとアイテムコレクションは同じ id の空間で引かれる
	for _, x := range raw.PtrSlice(raws.ItemGroups) {
		groupIDs[x.Id] = true
	}
	for _, x := range raw.PtrSlice(raws.ItemCollections) {
		groupIDs[x.Id] = true
	}

	var bad []string
	check := func(palette, kind, char, id string, known map[string]bool) {
		if !known[id] {
			bad = append(bad, fmt.Sprintf("%s %s %q: %q", palette, kind, char, id))
		}
	}
	names := slices.Sorted(maps.Keys(loader.paletteCache))
	require.NotEmpty(t, names)
	for _, name := range names {
		p := loader.paletteCache[name]
		for char, tile := range p.Terrain {
			check(name, "terrain", char, tile, tileIDs)
		}
		for char, e := range p.Props {
			check(name, "props.tile", char, e.Tile, tileIDs)
			if e.ID != "" {
				check(name, "props.id", char, e.ID, propIDs)
			}
			for _, c := range e.Candidates {
				check(name, "props.candidates", char, c.ID, propIDs)
			}
			if e.LootTable != "" {
				check(name, "props.lootTable", char, e.LootTable, tableIDs)
			}
		}
		for char, e := range p.NPCs {
			if e.Tile != "" {
				check(name, "npcs.tile", char, e.Tile, tileIDs)
			}
			check(name, "npcs.id", char, e.ID, memberIDs)
		}
		for char, e := range p.Items {
			if e.Tile != "" {
				check(name, "items.tile", char, e.Tile, tileIDs)
			}
			check(name, "items.itemGroup", char, e.ItemGroup, groupIDs)
		}
		for char, e := range p.Beings {
			check(name, "beings.tile", char, e.Tile, tileIDs)
		}
		for char, e := range p.Nested {
			check(name, "nested.tile", char, e.Tile, tileIDs)
			for _, c := range e.Chunks {
				if c.Name != nullChunk && len(loader.chunkCache[c.Name]) == 0 {
					bad = append(bad, fmt.Sprintf("%s nested.chunks %q: %q", name, char, c.Name))
				}
			}
		}
	}
	assert.Empty(t, bad)
}
