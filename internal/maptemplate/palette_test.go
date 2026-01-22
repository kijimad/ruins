package maptemplate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaletteLoader_LoadFromFile(t *testing.T) {
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

[palette.furniture]
"T" = "table"
"C" = "chair"
`
		require.NoError(t, os.WriteFile(paletteFile, []byte(content), 0644))

		// 読み込みテスト
		loader := NewPaletteLoader()
		palette, err := loader.LoadFromFile(paletteFile)

		require.NoError(t, err)
		assert.Equal(t, "test", palette.ID)
		assert.Equal(t, "テスト用パレット", palette.Description)
		assert.Equal(t, "wall", palette.Terrain["#"])
		assert.Equal(t, "floor", palette.Terrain["."])
		assert.Equal(t, "table", palette.Furniture["T"])
		assert.Equal(t, "chair", palette.Furniture["C"])
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
		_, err := loader.LoadFromFile(paletteFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "パレットIDが空です")
	})

	t.Run("地形と家具が両方空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		paletteFile := filepath.Join(tmpDir, "empty_palette.toml")

		content := `[palette]
id = "empty"
description = "空のパレット"
`
		require.NoError(t, os.WriteFile(paletteFile, []byte(content), 0644))

		loader := NewPaletteLoader()
		_, err := loader.LoadFromFile(paletteFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "地形または家具の定義が必要です")
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
		_, err := loader.LoadFromFile(paletteFile)

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
			Furniture: map[string]string{
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
			Furniture: map[string]string{
				"M": "machine", // 追加
			},
		}

		merged := MergePalettes(palette1, palette2)

		assert.Equal(t, "wall_metal", merged.Terrain["#"]) // palette2で上書きされている
		assert.Equal(t, "floor", merged.Terrain["."])
		assert.Equal(t, "dirt", merged.Terrain["~"])
		assert.Equal(t, "table", merged.Furniture["T"])
		assert.Equal(t, "machine", merged.Furniture["M"])
	})

	t.Run("空のパレットリストでもエラーにならない", func(t *testing.T) {
		t.Parallel()
		merged := MergePalettes()

		assert.NotNil(t, merged)
		assert.Equal(t, "merged", merged.ID)
		assert.Empty(t, merged.Terrain)
		assert.Empty(t, merged.Furniture)
	})
}

func TestPalette_GetTerrainAndFurniture(t *testing.T) {
	t.Parallel()
	palette := &Palette{
		ID: "test",
		Terrain: map[string]string{
			"#": "wall",
			".": "floor",
		},
		Furniture: map[string]string{
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

	t.Run("存在する家具を取得できる", func(t *testing.T) {
		t.Parallel()
		furniture, ok := palette.GetFurniture("T")
		assert.True(t, ok)
		assert.Equal(t, "table", furniture)
	})

	t.Run("存在しない家具はfalseを返す", func(t *testing.T) {
		t.Parallel()
		_, ok := palette.GetFurniture("X")
		assert.False(t, ok)
	})
}
