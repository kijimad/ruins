package maptemplate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateLoader_LoadFromFile(t *testing.T) {
	t.Parallel()
	t.Run("正常なテンプレート定義を読み込める", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "test_template.toml")

		content := `[[facility]]
type = "test_facility"
name = "テスト施設"
weight = 100
size = [5, 3]
entrance = [2, 0]
palettes = ["standard"]
map = """
#####
#...#
#####
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		templates, err := loader.LoadFromFile(templateFile)

		require.NoError(t, err)
		require.Len(t, templates, 1)

		template := templates[0]
		assert.Equal(t, "test_facility", template.Type)
		assert.Equal(t, "テスト施設", template.Name)
		assert.Equal(t, 100, template.Weight)
		assert.Equal(t, [2]int{5, 3}, template.Size)
		assert.Equal(t, [2]int{2, 0}, template.Entrance)
		assert.Equal(t, []string{"standard"}, template.Palettes)
	})

	t.Run("複数のテンプレートを読み込める", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "multi_template.toml")

		content := `[[facility]]
type = "small"
name = "小型施設"
weight = 50
size = [3, 3]
entrance = [1, 0]
palettes = ["standard"]
map = """
###
#.#
###
"""

[[facility]]
type = "large"
name = "大型施設"
weight = 30
size = [5, 5]
entrance = [2, 0]
palettes = ["standard"]
map = """
#####
#...#
#...#
#...#
#####
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		templates, err := loader.LoadFromFile(templateFile)

		require.NoError(t, err)
		require.Len(t, templates, 2)
		assert.Equal(t, "small", templates[0].Type)
		assert.Equal(t, "large", templates[1].Type)
	})

	t.Run("typeが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "invalid_type.toml")

		content := `[[facility]]
type = ""
name = "無効な施設"
weight = 100
size = [3, 3]
entrance = [1, 0]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFromFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "施設タイプが空です")
	})

	t.Run("weightが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "invalid_weight.toml")

		content := `[[facility]]
type = "test"
name = "無効な重み"
weight = 0
size = [3, 3]
entrance = [1, 0]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFromFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "重みは正の整数である必要があります")
	})

	t.Run("マップサイズが定義と一致しない場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "size_mismatch.toml")

		content := `[[facility]]
type = "test"
name = "サイズ不一致"
weight = 100
size = [5, 3]
entrance = [1, 0]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFromFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "実サイズ")
		assert.Contains(t, err.Error(), "定義サイズ")
	})

	t.Run("マップの行長が不一致の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "irregular_map.toml")

		content := `[[facility]]
type = "test"
name = "不規則マップ"
weight = 100
size = [5, 3]
entrance = [1, 0]
palettes = ["standard"]
map = """
#####
#..#
#####
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFromFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "行")
		assert.Contains(t, err.Error(), "長さが不一致")
	})

}

func TestFacilityTemplate_GetMapLines(t *testing.T) {
	t.Parallel()
	template := FacilityTemplate{
		Map: `###
#.#
###`,
	}

	lines := template.GetMapLines()
	require.Len(t, lines, 3)
	assert.Equal(t, "###", lines[0])
	assert.Equal(t, "#.#", lines[1])
	assert.Equal(t, "###", lines[2])
}

func TestFacilityTemplate_GetCharAt(t *testing.T) {
	t.Parallel()
	template := FacilityTemplate{
		Map: `###
#.#
###`,
	}

	t.Run("正常な座標の文字を取得できる", func(t *testing.T) {
		t.Parallel()
		char, err := template.GetCharAt(0, 0)
		require.NoError(t, err)
		assert.Equal(t, "#", char)

		char, err = template.GetCharAt(1, 1)
		require.NoError(t, err)
		assert.Equal(t, ".", char)
	})

	t.Run("範囲外のY座標はエラー", func(t *testing.T) {
		t.Parallel()
		_, err := template.GetCharAt(0, 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "y座標が範囲外です")
	})

	t.Run("範囲外のX座標はエラー", func(t *testing.T) {
		t.Parallel()
		_, err := template.GetCharAt(5, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "x座標が範囲外です")
	})
}
