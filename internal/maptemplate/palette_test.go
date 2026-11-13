package maptemplate

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaletteLoader_Load(t *testing.T) {
	t.Parallel()
	t.Run("正常なパレット定義を読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "test"
description = "テスト用パレット"

[palette.terrain]
"#" = "wall"
"." = "floor"

[palette.props]
"T" = { id = "table", tile = "floor" }
"C" = { id = "chair", tile = "floor" }
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Equal(t, "test", palette.ID)
		assert.Equal(t, "テスト用パレット", palette.Description)
		assert.Equal(t, "wall", palette.Terrain["#"])
		assert.Equal(t, "floor", palette.Terrain["."])
		assert.Equal(t, "table", palette.Props["T"].ID)
		assert.Equal(t, "chair", palette.Props["C"].ID)
	})

	t.Run("IDが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = ""
description = "無効なパレット"

[palette.terrain]
"#" = "wall"
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteIDEmpty)
	})

	t.Run("地形とPropsとNPCsが全て空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "empty"
description = "空のパレット"
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteNoDefinition)
	})

	t.Run("2文字以上のキーはエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "invalid"
description = "無効なキー"

[palette.terrain]
"##" = "wall"
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteKeyNotSingleChar)
	})

	t.Run("Propsにtileがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "no_tile"
description = "tile無し"

[palette.props]
"T" = { id = "table" }
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteTileRequired)
	})

	t.Run("NPCsにtileがない場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "no_tile"
description = "tile無し"

[palette.npcs]
"G" = { id = "guard" }
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteTileRequired)
	})

	t.Run("地形とPropsで文字が重複する場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "dup"
description = "重複"

[palette.terrain]
"T" = "floor"

[palette.props]
"T" = { id = "table", tile = "floor" }
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteCharDuplicated)
	})

	t.Run("地形とNPCsで文字が重複する場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "dup"
description = "重複"

[palette.terrain]
"G" = "floor"

[palette.npcs]
"G" = { id = "guard", tile = "floor" }
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteCharDuplicated)
	})

	t.Run("PropsとNPCsで文字が重複する場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "dup"
description = "重複"

[palette.props]
"X" = { id = "crate", tile = "floor" }

[palette.npcs]
"X" = { id = "guard", tile = "floor" }
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.ErrorIs(t, err, errPaletteCharDuplicated)
	})

	t.Run("NPCsを含むパレットを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "with_npcs"
description = "NPC付きパレット"

[palette.terrain]
"." = "floor"

[palette.npcs]
"G" = { id = "guard", tile = "floor" }
"M" = { id = "merchant", tile = "floor" }
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Equal(t, "with_npcs", palette.ID)
		assert.Equal(t, "guard", palette.NPCs["G"].ID)
		assert.Equal(t, "merchant", palette.NPCs["M"].ID)
	})

	t.Run("地形のみのパレットを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "terrain_only"
description = "地形のみ"

[palette.terrain]
"#" = "wall"
"." = "floor"
"~" = "water"
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Len(t, palette.Terrain, 3)
		assert.Equal(t, "wall", palette.Terrain["#"])
		assert.Equal(t, "floor", palette.Terrain["."])
		assert.Equal(t, "water", palette.Terrain["~"])
		assert.Empty(t, palette.Props)
		assert.Empty(t, palette.NPCs)
	})

	t.Run("Propsのみのパレットを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "props_only"
description = "Propsのみ"

[palette.props]
"T" = { id = "table", tile = "floor" }
"C" = { id = "chair", tile = "floor" }
"B" = { id = "bed", tile = "floor" }
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Len(t, palette.Props, 3)
		assert.Equal(t, "table", palette.Props["T"].ID)
		assert.Equal(t, "chair", palette.Props["C"].ID)
		assert.Equal(t, "bed", palette.Props["B"].ID)
		assert.Empty(t, palette.Terrain)
		assert.Empty(t, palette.NPCs)
	})

	t.Run("全種類を含むパレットを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "complete"
description = "完全なパレット"

[palette.terrain]
"#" = "wall"
"." = "floor"

[palette.props]
"T" = { id = "table", tile = "floor" }
"D" = { id = "door", tile = "floor" }

[palette.npcs]
"G" = { id = "guard", tile = "floor" }
"V" = { id = "villager", tile = "floor" }
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Len(t, palette.Terrain, 2)
		assert.Len(t, palette.Props, 2)
		assert.Len(t, palette.NPCs, 2)
		assert.Equal(t, "wall", palette.Terrain["#"])
		assert.Equal(t, "table", palette.Props["T"].ID)
		assert.Equal(t, "guard", palette.NPCs["G"].ID)
	})

	t.Run("マルチバイト文字のキーは1文字と判定される", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "japanese"
description = "日本語キー"

[palette.terrain]
"壁" = "wall"
"床" = "floor"
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Equal(t, "wall", palette.Terrain["壁"])
		assert.Equal(t, "floor", palette.Terrain["床"])
	})

	t.Run("tileフィールド付きのProp/NPCを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "with_tile"
description = "tile付き"

[palette.terrain]
"." = "floor"

[palette.props]
"+" = { id = "door", tile = "floor" }

[palette.npcs]
"$" = { id = "merchant", tile = "dirt" }
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Equal(t, "door", palette.Props["+"].ID)
		assert.Equal(t, "floor", palette.Props["+"].Tile)
		assert.Equal(t, "merchant", palette.NPCs["$"].ID)
		assert.Equal(t, "dirt", palette.NPCs["$"].Tile)
	})

	t.Run("tile付きエントリはGetTerrainで地形を返す", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "tile_terrain"
description = "tile経由の地形"

[palette.terrain]
"." = "floor"

[palette.npcs]
"$" = { id = "merchant", tile = "dirt" }
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))
		require.NoError(t, err)

		terrain, ok := palette.GetTerrain(".")
		assert.True(t, ok)
		assert.Equal(t, "floor", terrain)

		terrain, ok = palette.GetTerrain("$")
		assert.True(t, ok)
		assert.Equal(t, "dirt", terrain)

		_, ok = palette.GetTerrain("X")
		assert.False(t, ok)
	})
}

func TestPaletteLoader_LoadFile(t *testing.T) {
	t.Parallel()
	t.Run("実ファイルからパレット定義を読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewPaletteLoader()
		palette, err := loader.LoadFile("levels/palettes/standard.toml")

		require.NoError(t, err)
		assert.Equal(t, "standard", palette.ID)
		assert.NotEmpty(t, palette.Terrain)
	})

	t.Run("存在しないファイルはエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewPaletteLoader()
		_, err := loader.LoadFile("nonexistent.toml")

		require.Error(t, err)
	})
}

func TestMergePalettes(t *testing.T) {
	t.Parallel()
	t.Run("複数のパレットを正しくマージできる", func(t *testing.T) {
		t.Parallel()
		palette1 := &Palette{
			ID:          "base",
			Description: "基本パレット",
			Terrain: map[string]string{
				"#": "wall",
				".": "floor",
			},
			Props: map[string]PaletteEntry{
				"T": {ID: "table", Tile: "floor"},
			},
		}

		palette2 := &Palette{
			ID:          "extended",
			Description: "拡張パレット",
			Terrain: map[string]string{
				"#": "wall_metal",
				"~": "dirt",
			},
			Props: map[string]PaletteEntry{
				"M": {ID: "machine", Tile: "floor"},
			},
		}

		merged := MergePalettes(palette1, palette2)

		assert.Equal(t, "wall_metal", merged.Terrain["#"])
		assert.Equal(t, "floor", merged.Terrain["."])
		assert.Equal(t, "dirt", merged.Terrain["~"])
		assert.Equal(t, "table", merged.Props["T"].ID)
		assert.Equal(t, "machine", merged.Props["M"].ID)
	})

	t.Run("空のパレットリストでもエラーにならない", func(t *testing.T) {
		t.Parallel()
		merged := MergePalettes()

		assert.NotNil(t, merged)
		assert.Equal(t, "merged", merged.ID)
		assert.Empty(t, merged.Terrain)
		assert.Empty(t, merged.Props)
	})

	t.Run("tile付きエントリもマージされる", func(t *testing.T) {
		t.Parallel()
		p1 := &Palette{
			ID:      "p1",
			Terrain: map[string]string{".": "floor"},
			Props:   map[string]PaletteEntry{"+": {ID: "door", Tile: "floor"}},
			NPCs:    map[string]PaletteEntry{},
		}
		merged := MergePalettes(p1)

		assert.Equal(t, "door", merged.Props["+"].ID)
		assert.Equal(t, "floor", merged.Props["+"].Tile)
	})
}

func TestPalette_GetTerrainAndProp(t *testing.T) {
	t.Parallel()
	palette := &Palette{
		ID: "test",
		Terrain: map[string]string{
			"#": "wall",
			".": "floor",
		},
		Props: map[string]PaletteEntry{
			"T": {ID: "table", Tile: "floor"},
		},
	}

	t.Run("存在する地形を取得できる", func(t *testing.T) {
		t.Parallel()
		terrain, ok := palette.GetTerrain("#")
		assert.True(t, ok)
		assert.Equal(t, "wall", terrain)
	})

	t.Run("存在しない地形はfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := palette.GetTerrain("X")
		assert.False(t, ok)
	})

	t.Run("存在するPropsを取得できる", func(t *testing.T) {
		t.Parallel()
		prop, lootTable, ok := palette.GetProp("T", rand.New(rand.NewPCG(1, 1)))
		assert.True(t, ok)
		assert.Equal(t, "table", prop)
		assert.Empty(t, lootTable)
	})

	t.Run("存在しないPropsはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, _, ok := palette.GetProp("X", rand.New(rand.NewPCG(1, 1)))
		assert.False(t, ok)
	})
}

func TestPaletteLoader_Load_施設の文字(t *testing.T) {
	t.Parallel()

	const head = "[palette]\nid = \"facility\"\n"
	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{"候補と収納の中身と床の戦利品と敵を書ける", `
[palette.props.d]
tile = "floor"
lootTable = "house_closet"
candidates = [{ id = "dresser", weight = 2 }, { id = "bookshelf", weight = 1 }]
[palette.items.f]
tile = "floor"
itemGroup = "food"
[palette.beings.Z]
tile = "floor"
`, nil},
		{"prop は id と候補の両方を書けない", `
[palette.props.d]
tile = "floor"
id = "dresser"
candidates = [{ id = "bookshelf", weight = 1 }]
`, errPalettePropIDAmbiguous},
		{"prop は id か候補のどちらかが要る", `
[palette.props.d]
tile = "floor"
`, errPalettePropIDAmbiguous},
		{"候補の重みは正", `
[palette.props.d]
tile = "floor"
candidates = [{ id = "dresser", weight = 0 }]
`, errNonPositiveWeight},
		{"床の戦利品は itemGroup が要る", `
[palette.items.f]
tile = "floor"
`, errPaletteItemGroupRequired},
		{"敵は下に敷く地形が要る", `
[palette.beings.Z]
`, errPaletteTileRequired},
		{"床の戦利品は家具と同じ文字に重ねられる", `
[palette.props.t]
id = "table"
tile = "floor"
[palette.items.t]
itemGroup = "food"
chance = 20
repeat = [1, 2]
`, nil},
		{"確率は 100 以下", `
[palette.items.f]
tile = "floor"
itemGroup = "food"
chance = 101
`, errPaletteChanceOutOfRange},
		{"回数の範囲は最小が最大以下", `
[palette.items.f]
tile = "floor"
itemGroup = "food"
repeat = [3, 1]
`, errPaletteRepeatInvalid},
		{"敵の文字は prop と重ならない", `
[palette.props.Z]
id = "dresser"
tile = "floor"
[palette.beings.Z]
tile = "floor"
`, errPaletteCharDuplicated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewPaletteLoader().Load(strings.NewReader(head + tt.body))
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestResolveMapCells_施設の文字(t *testing.T) {
	t.Parallel()

	palette := &Palette{
		Terrain: map[string]string{".": "floor"},
		Props: map[string]PaletteEntry{
			"d": {Tile: "floor", LootTable: "house_closet", Candidates: []WeightedID{{ID: "dresser", Weight: 1}, {ID: "bookshelf", Weight: 1}}},
		},
		Items:  map[string]ItemEntry{"f": {Tile: "floor", ItemGroup: "food"}},
		Beings: map[string]BeingEntry{"Z": {Tile: "floor"}},
	}

	cells := ResolveMapCells("dfZ.", palette, rand.New(rand.NewPCG(1, 1)))
	require.Len(t, cells, 1)
	assert.Contains(t, []string{"dresser", "bookshelf"}, cells[0][0].Prop)
	cells[0][0].Prop = ""
	assert.Equal(t, []MapCell{
		{Terrain: "floor", LootTable: "house_closet"},
		{Terrain: "floor", ItemGroup: "food", ItemCount: 1},
		{Terrain: "floor", Being: true},
		{Terrain: "floor"},
	}, cells[0])

	// 候補は重みに従って両方とも選ばれる
	seen := map[string]bool{}
	for s := range uint64(64) {
		seen[ResolveMapCells("d", palette, rand.New(rand.NewPCG(s, s)))[0][0].Prop] = true
	}
	assert.Equal(t, map[string]bool{"dresser": true, "bookshelf": true}, seen)
}

func TestItemEntry_roll(t *testing.T) {
	t.Parallel()

	rng := rand.New(rand.NewPCG(1, 1))
	tests := []struct {
		name  string
		entry ItemEntry
		want  []int
	}{
		{"省くと1回", ItemEntry{}, []int{1}},
		{"確率100で回数の範囲の全てが出る", ItemEntry{Chance: 100, Repeat: []int{1, 3}}, []int{1, 2, 3}},
		{"確率50で0回から回数の上限まで出る", ItemEntry{Chance: 50, Repeat: []int{2, 2}}, []int{0, 1, 2}},
	}
	for _, tt := range tests {
		seen := map[int]bool{}
		for range 200 {
			seen[tt.entry.roll(rng)] = true
		}
		got := slices.Sorted(maps.Keys(seen))
		assert.Equal(t, tt.want, got, tt.name)
	}
}
