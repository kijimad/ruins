package maptemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRealFiles(t *testing.T) {
	t.Parallel()

	t.Run("標準パレットを読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewPaletteLoader()
		palette, err := loader.LoadFromFile("../../assets/levels/palettes/standard.toml")

		require.NoError(t, err)
		assert.Equal(t, "standard", palette.ID)

		// 地形の確認
		assert.Equal(t, "Wall", palette.Terrain["#"])
		assert.Equal(t, "Floor", palette.Terrain["."])

		// 家具の確認
		assert.Equal(t, "table", palette.Furniture["T"])
		assert.Equal(t, "chair", palette.Furniture["C"])
	})

	t.Run("小部屋テンプレートを読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()
		templates, err := loader.LoadFromFile("../../assets/levels/facilities/small_room.toml")

		require.NoError(t, err)
		require.Len(t, templates, 1)

		template := templates[0]
		assert.Equal(t, "small_room", template.Type)
		assert.Equal(t, "小部屋", template.Name)
		assert.Equal(t, 100, template.Weight)
		assert.Equal(t, [2]int{10, 10}, template.Size)
		assert.Equal(t, []string{"standard"}, template.Palettes)
		assert.NotEmpty(t, template.Map)
	})
}
