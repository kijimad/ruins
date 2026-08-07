package maptemplate

import (
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "パレットIDが空です")
	})

	t.Run("地形とPropsとNPCsが全て空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "empty"
description = "空のパレット"
`
		loader := NewPaletteLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "地形、Props、またはNPCsの定義が必要です")
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "1文字である必要があります")
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "propsのtileは必須です")
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "npcsのtileは必須です")
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "重複しています")
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "重複しています")
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "重複しています")
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
		prop, ok := palette.GetProp("T")
		assert.True(t, ok)
		assert.Equal(t, "table", prop)
	})

	t.Run("存在しないPropsはfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := palette.GetProp("X")
		assert.False(t, ok)
	})
}
