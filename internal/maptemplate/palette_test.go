package maptemplate

import (
	"os"
	"path/filepath"
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
"T" = "table"
"C" = "chair"
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Equal(t, "test", palette.ID)
		assert.Equal(t, "テスト用パレット", palette.Description)
		assert.Equal(t, "wall", palette.Terrain["#"])
		assert.Equal(t, "floor", palette.Terrain["."])
		assert.Equal(t, "table", palette.Props["T"])
		assert.Equal(t, "chair", palette.Props["C"])
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

	t.Run("NPCsを含むパレットを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[palette]
id = "with_npcs"
description = "NPC付きパレット"

[palette.terrain]
"." = "floor"

[palette.npcs]
"G" = "guard"
"M" = "merchant"
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Equal(t, "with_npcs", palette.ID)
		assert.Equal(t, "guard", palette.NPCs["G"])
		assert.Equal(t, "merchant", palette.NPCs["M"])
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
"T" = "table"
"C" = "chair"
"B" = "bed"
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Len(t, palette.Props, 3)
		assert.Equal(t, "table", palette.Props["T"])
		assert.Equal(t, "chair", palette.Props["C"])
		assert.Equal(t, "bed", palette.Props["B"])
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
"T" = "table"
"D" = "door"

[palette.npcs]
"G" = "guard"
"V" = "villager"
`
		loader := NewPaletteLoader()
		palette, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		assert.Len(t, palette.Terrain, 2)
		assert.Len(t, palette.Props, 2)
		assert.Len(t, palette.NPCs, 2)
		assert.Equal(t, "wall", palette.Terrain["#"])
		assert.Equal(t, "table", palette.Props["T"])
		assert.Equal(t, "guard", palette.NPCs["G"])
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
}

func TestPaletteLoader_LoadFile(t *testing.T) {
	t.Parallel()
	t.Run("正常なパレット定義を読み込める", func(t *testing.T) {
		t.Parallel()
		// テスト用のTOMLファイルを作成
		tmpDir := t.TempDir()
		paletteFile := filepath.Join(tmpDir, "test_palette.toml")

		content := `[palette]
id = "test"
description = "テスト用パレット"

[palette.terrain]
"#" = "wall"
"." = "floor"

[palette.props]
"T" = "table"
"C" = "chair"
`
		require.NoError(t, os.WriteFile(paletteFile, []byte(content), 0644))

		// 読み込みテスト
		loader := NewPaletteLoader()
		palette, err := loader.LoadFile(paletteFile)

		require.NoError(t, err)
		assert.Equal(t, "test", palette.ID)
		assert.Equal(t, "テスト用パレット", palette.Description)
		assert.Equal(t, "wall", palette.Terrain["#"])
		assert.Equal(t, "floor", palette.Terrain["."])
		assert.Equal(t, "table", palette.Props["T"])
		assert.Equal(t, "chair", palette.Props["C"])
	})

	t.Run("IDが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paletteFile := filepath.Join(tmpDir, "invalid_palette.toml")

		content := `[palette]
id = ""
description = "無効なパレット"

[palette.terrain]
"#" = "wall"
`
		require.NoError(t, os.WriteFile(paletteFile, []byte(content), 0644))

		loader := NewPaletteLoader()
		_, err := loader.LoadFile(paletteFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "パレットIDが空です")
	})

	t.Run("地形とPropsとNPCsが全て空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paletteFile := filepath.Join(tmpDir, "empty_palette.toml")

		content := `[palette]
id = "empty"
description = "空のパレット"
`
		require.NoError(t, os.WriteFile(paletteFile, []byte(content), 0644))

		loader := NewPaletteLoader()
		_, err := loader.LoadFile(paletteFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "地形、Props、またはNPCsの定義が必要です")
	})

	t.Run("2文字以上のキーはエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paletteFile := filepath.Join(tmpDir, "invalid_key.toml")

		content := `[palette]
id = "invalid"
description = "無効なキー"

[palette.terrain]
"##" = "wall"
`
		require.NoError(t, os.WriteFile(paletteFile, []byte(content), 0644))

		loader := NewPaletteLoader()
		_, err := loader.LoadFile(paletteFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "1文字である必要があります")
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
			Props: map[string]string{
				"T": "table",
			},
		}

		palette2 := &Palette{
			ID:          "extended",
			Description: "拡張パレット",
			Terrain: map[string]string{
				"#": "wall_metal", // 上書き
				"~": "dirt",       // 追加
			},
			Props: map[string]string{
				"M": "machine", // 追加
			},
		}

		merged := MergePalettes(palette1, palette2)

		assert.Equal(t, "wall_metal", merged.Terrain["#"]) // palette2で上書きされている
		assert.Equal(t, "floor", merged.Terrain["."])
		assert.Equal(t, "dirt", merged.Terrain["~"])
		assert.Equal(t, "table", merged.Props["T"])
		assert.Equal(t, "machine", merged.Props["M"])
	})

	t.Run("空のパレットリストでもエラーにならない", func(t *testing.T) {
		t.Parallel()
		merged := MergePalettes()

		assert.NotNil(t, merged)
		assert.Equal(t, "merged", merged.ID)
		assert.Empty(t, merged.Terrain)
		assert.Empty(t, merged.Props)
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
		Props: map[string]string{
			"T": "table",
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
