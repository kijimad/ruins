package maptemplate

import (
	"strings"
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
		assert.Equal(t, "small_room", template.Name)
		assert.Equal(t, 100, template.Weight)
		assert.Equal(t, [2]int{10, 10}, template.Size)
		assert.Equal(t, []string{"standard"}, template.Palettes)
		assert.NotEmpty(t, template.Map)
	})

	t.Run("チャンク定義を読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()
		err := loader.LoadChunk("../../assets/levels/chunks/rooms.toml")

		require.NoError(t, err)

		// 各チャンクが読み込まれているか確認
		bedroom, err := loader.GetChunk("bedroom")
		require.NoError(t, err)
		assert.Equal(t, "bedroom", bedroom.Name)
		assert.Equal(t, [2]int{3, 3}, bedroom.Size)

		meetingRoom, err := loader.GetChunk("meeting_room")
		require.NoError(t, err)
		assert.Equal(t, "meeting_room", meetingRoom.Name)
		assert.Equal(t, [2]int{5, 5}, meetingRoom.Size)

		storage, err := loader.GetChunk("storage")
		require.NoError(t, err)
		assert.Equal(t, "storage", storage.Name)
		assert.Equal(t, [2]int{4, 4}, storage.Size)

		office, err := loader.GetChunk("office")
		require.NoError(t, err)
		assert.Equal(t, "office", office.Name)
		assert.Equal(t, [2]int{5, 4}, office.Size)
	})

	t.Run("複合施設テンプレートを読み込んでチャンク展開できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// チャンクを読み込む
		err := loader.LoadChunk("../../assets/levels/chunks/rooms.toml")
		require.NoError(t, err)

		// 複合施設テンプレートを読み込む
		templates, err := loader.LoadFromFile("../../assets/levels/facilities/compound_building.toml")
		require.NoError(t, err)
		require.NotEmpty(t, templates)

		// 小規模オフィスをチャンク展開
		smallOffice := templates[0]
		assert.Equal(t, "small_office", smallOffice.Name)
		assert.Len(t, smallOffice.Chunks, 2)

		expandedMap, err := smallOffice.ExpandWithChunks(loader, 12345)
		require.NoError(t, err)
		assert.NotEmpty(t, expandedMap)

		// 展開後のマップが元のマップより情報量が増えていることを確認
		// チャンク領域('A', 'B')が実際の内容で置き換えられている
		assert.NotContains(t, expandedMap, "A")
		assert.NotContains(t, expandedMap, "B")

		// 展開後のマップサイズが維持されていることを確認
		lines := splitMapLines(expandedMap)
		assert.Len(t, lines, smallOffice.Size[1])
		for _, line := range lines {
			assert.Len(t, line, smallOffice.Size[0])
		}
	})
}

// splitMapLines はマップ文字列を行の配列に分割する
func splitMapLines(mapStr string) []string {
	return strings.Split(strings.TrimSpace(mapStr), "\n")
}
