package maptemplate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateLoader_Load(t *testing.T) {
	t.Parallel()
	t.Run("正常なテンプレート定義を読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "test_facility"
weight = 100
size = [5, 3]
palettes = ["standard"]
map = """
#####
#...#
#####
"""
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 1)

		template := templates[0]
		assert.Equal(t, "test_facility", template.Name)
		assert.Equal(t, 100, template.Weight)
		assert.Equal(t, [2]int{5, 3}, template.Size)
		assert.Equal(t, []string{"standard"}, template.Palettes)
	})

	t.Run("複数のテンプレートを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "small"
weight = 50
size = [3, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""

[[chunk]]
name = "large"
weight = 30
size = [5, 5]
palettes = ["standard"]
map = """
#####
#...#
#...#
#...#
#####
"""
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 2)
		assert.Equal(t, "small", templates[0].Name)
		assert.Equal(t, "large", templates[1].Name)
	})

	t.Run("nameが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = ""
weight = 100
size = [3, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		loader := NewTemplateLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "チャンク名（キー）が空です")
	})

	t.Run("weightが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "無効な重み"
weight = 0
size = [3, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		loader := NewTemplateLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "重みは正の整数である必要があります")
	})

	t.Run("マップサイズが定義と一致しない場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "サイズ不一致"
weight = 100
size = [5, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		loader := NewTemplateLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "実サイズ")
		assert.Contains(t, err.Error(), "定義サイズ")
	})

	t.Run("マップの行長が不一致の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "不規則マップ"
weight = 100
size = [5, 3]
palettes = ["standard"]
map = """
#####
#..#
#####
"""
`
		loader := NewTemplateLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "行")
		assert.Contains(t, err.Error(), "長さが不一致")
	})

	t.Run("複数パレットを指定できる", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "multi_palette"
weight = 100
size = [3, 3]
palettes = ["standard", "town", "dungeon"]
map = """
###
#.#
###
"""
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, []string{"standard", "town", "dungeon"}, templates[0].Palettes)
	})

	t.Run("パレット指定なしでも読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "no_palette"
weight = 100
size = [3, 3]
map = """
###
#.#
###
"""
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Empty(t, templates[0].Palettes)
	})

	t.Run("PlaceNestedを持つテンプレートを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "with_nested"
weight = 100
size = [5, 5]
palettes = ["standard"]
map = """
#####
#@@@#
#@@A#
#@@@#
#####
"""

[[chunk.place_nested]]
chunks = ["room"]
id = "A"
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Len(t, templates[0].PlaceNested, 1)
		assert.Equal(t, []string{"room"}, templates[0].PlaceNested[0].Chunks)
		assert.Equal(t, "A", templates[0].PlaceNested[0].ID)
	})

	t.Run("サイズ0はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "zero_size"
weight = 100
size = [0, 3]
palettes = ["standard"]
map = """
"""
`
		loader := NewTemplateLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "サイズは正の整数である必要があります")
	})

	t.Run("負の重みはエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "negative_weight"
weight = -10
size = [3, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		loader := NewTemplateLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "重みは正の整数である必要があります")
	})

	t.Run("マップが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "empty_map"
weight = 100
size = [3, 3]
palettes = ["standard"]
map = ""
`
		loader := NewTemplateLoader()
		_, err := loader.Load(strings.NewReader(content))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "マップが空です")
	})

	t.Run("大きなマップを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "large_map"
weight = 100
size = [10, 10]
palettes = ["standard"]
map = """
##########
#........#
#........#
#........#
#........#
#........#
#........#
#........#
#........#
##########
"""
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, [2]int{10, 10}, templates[0].Size)
	})

	t.Run("マルチバイト文字を含むマップを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "japanese_map"
weight = 100
size = [5, 3]
palettes = ["standard"]
map = """
壁壁壁壁壁
壁床床床壁
壁壁壁壁壁
"""
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Contains(t, templates[0].Map, "壁")
		assert.Contains(t, templates[0].Map, "床")
	})
}

func TestTemplateLoader_LoadFile(t *testing.T) {
	t.Parallel()
	t.Run("正常なテンプレート定義を読み込める", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "test_template.toml")

		content := `[[chunk]]
name = "test_facility"
weight = 100
size = [5, 3]
palettes = ["standard"]
map = """
#####
#...#
#####
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		templates, err := loader.LoadFile(templateFile)

		require.NoError(t, err)
		require.Len(t, templates, 1)

		template := templates[0]
		assert.Equal(t, "test_facility", template.Name)
		assert.Equal(t, 100, template.Weight)
		assert.Equal(t, [2]int{5, 3}, template.Size)
		assert.Equal(t, []string{"standard"}, template.Palettes)
	})

	t.Run("複数のテンプレートを読み込める", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "multi_template.toml")

		content := `[[chunk]]
name = "small"
weight = 50
size = [3, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""

[[chunk]]
name = "large"
weight = 30
size = [5, 5]
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
		templates, err := loader.LoadFile(templateFile)

		require.NoError(t, err)
		require.Len(t, templates, 2)
		assert.Equal(t, "small", templates[0].Name)
		assert.Equal(t, "large", templates[1].Name)
	})

	t.Run("nameが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "invalid_name.toml")

		content := `[[chunk]]
name = ""
weight = 100
size = [3, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "チャンク名（キー）が空です")
	})

	t.Run("weightが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "invalid_weight.toml")

		content := `[[chunk]]
type = "test"
name = "無効な重み"
weight = 0
size = [3, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "重みは正の整数である必要があります")
	})

	t.Run("マップサイズが定義と一致しない場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "size_mismatch.toml")

		content := `[[chunk]]
type = "test"
name = "サイズ不一致"
weight = 100
size = [5, 3]
palettes = ["standard"]
map = """
###
#.#
###
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "実サイズ")
		assert.Contains(t, err.Error(), "定義サイズ")
	})

	t.Run("マップの行長が不一致の場合はエラー", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		templateFile := filepath.Join(tmpDir, "irregular_map.toml")

		content := `[[chunk]]
type = "test"
name = "不規則マップ"
weight = 100
size = [5, 3]
palettes = ["standard"]
map = """
#####
#..#
#####
"""
`
		require.NoError(t, os.WriteFile(templateFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		_, err := loader.LoadFile(templateFile)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "行")
		assert.Contains(t, err.Error(), "長さが不一致")
	})

}

func TestChunkTemplate_GetMapLines(t *testing.T) {
	t.Parallel()
	template := ChunkTemplate{
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

func TestChunkTemplate_GetCharAt(t *testing.T) {
	t.Parallel()
	template := ChunkTemplate{
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

func TestTemplateLoader_ChunkOperations(t *testing.T) {
	t.Parallel()

	t.Run("チャンク定義を読み込んでキャッシュできる", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		chunkFile := filepath.Join(tmpDir, "chunk.toml")

		content := `[[chunk]]
name = "test_chunk"
weight = 100
size = [3, 3]
palettes = ["standard"]
map = """
...
.T.
...
"""
`
		require.NoError(t, os.WriteFile(chunkFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		err := loader.LoadChunk(chunkFile)
		require.NoError(t, err)

		// キャッシュから取得できるか確認
		chunks, err := loader.GetChunks("test_chunk")
		require.NoError(t, err)
		require.Len(t, chunks, 1)
		assert.Equal(t, "test_chunk", chunks[0].Name)
		assert.Equal(t, [2]int{3, 3}, chunks[0].Size)
	})

	t.Run("存在しないチャンクはエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()
		_, err := loader.GetChunks("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "見つかりません")
	})

	t.Run("同じ名前で複数のバリエーションを登録できる", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		chunkFile := filepath.Join(tmpDir, "variants.toml")

		// 同じ名前で重みが異なる3つのバリエーション
		content := `[[chunk]]
name = "room"
weight = 100
size = [3, 3]
palettes = ["standard"]
map = """
...
.1.
...
"""

[[chunk]]
name = "room"
weight = 50
size = [3, 3]
palettes = ["standard"]
map = """
...
.2.
...
"""

[[chunk]]
name = "room"
weight = 10
size = [3, 3]
palettes = ["standard"]
map = """
...
.3.
...
"""
`
		require.NoError(t, os.WriteFile(chunkFile, []byte(content), 0644))

		loader := NewTemplateLoader()
		err := loader.LoadChunk(chunkFile)
		require.NoError(t, err)

		// GetChunksで3つのバリエーションを取得
		chunks, err := loader.GetChunks("room")
		require.NoError(t, err)
		assert.Len(t, chunks, 3)

		// 各バリエーションの確認
		assert.Equal(t, 100, chunks[0].Weight)
		assert.Equal(t, 50, chunks[1].Weight)
		assert.Equal(t, 10, chunks[2].Weight)

		// 最初のバリエーションを確認
		assert.Equal(t, "room", chunks[0].Name)
		assert.Contains(t, chunks[0].Map, "1")
	})
}

func TestChunkTemplate_ExpandWithPlaceNested(t *testing.T) {
	t.Parallel()

	t.Run("place_nestedなしではそのまま返す", func(t *testing.T) {
		t.Parallel()
		template := ChunkTemplate{
			Map: `###
#.#
###`,
			Weight: 100,
			Size:   [2]int{3, 3},
		}

		loader := NewTemplateLoader()
		result, err := template.ExpandWithPlaceNested(loader, 0)
		require.NoError(t, err)
		assert.Equal(t, template.Map, result)
	})

	t.Run("単一チャンクを展開できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// チャンクを登録（内部のみ、外壁なし）
		chunk := &ChunkTemplate{
			Name:   "room",
			Weight: 100,
			Size:   [2]int{3, 3},
			Map: `...
.T.
...`,
		}
		loader.chunkCache["room"] = []*ChunkTemplate{chunk}

		// メインテンプレート（外壁を提供）
		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   [2]int{5, 5},
			Map: `#####
#@@@#
#@@A#
#@@@#
#####`,
			PlaceNested: []ChunkPlacement{
				{Chunks: []string{"room"}, ID: "A"},
			},
		}

		result, err := template.ExpandWithPlaceNested(loader, 0)
		require.NoError(t, err)

		expected := `#####
#...#
#.T.#
#...#
#####`
		assert.Equal(t, expected, result)
	})

	t.Run("複数チャンクを展開できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 部屋チャンク（内部のみ、外壁なし）
		room := &ChunkTemplate{
			Name:   "room",
			Weight: 100,
			Size:   [2]int{3, 3},
			Map: `...
.T.
...`,
		}
		loader.chunkCache["room"] = []*ChunkTemplate{room}

		// 倉庫チャンク（内部のみ、外壁なし）
		storage := &ChunkTemplate{
			Name:   "storage",
			Weight: 100,
			Size:   [2]int{3, 3},
			Map: `...
.X.
...`,
		}
		loader.chunkCache["storage"] = []*ChunkTemplate{storage}

		// メインテンプレート（外壁と廊下を提供）
		template := ChunkTemplate{
			Name:   "compound",
			Weight: 100,
			Size:   [2]int{9, 5},
			Map: `###+####+#
#@@@#.#@@@
#@@@#.#@@@
#@@A#.#@@B
#####.####`,
			PlaceNested: []ChunkPlacement{
				{Chunks: []string{"room"}, ID: "A"},
				{Chunks: []string{"storage"}, ID: "B"},
			},
		}

		result, err := template.ExpandWithPlaceNested(loader, 0)
		require.NoError(t, err)

		expected := `###+####+#
#...#.#...
#.T.#.#.X.
#...#.#...
#####.####`
		assert.Equal(t, expected, result)
	})

	t.Run("チャンクサイズが不一致の場合はエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 3x3のチャンク（内部のみ）
		chunk := &ChunkTemplate{
			Name:   "room",
			Weight: 100,
			Size:   [2]int{3, 3},
			Map: `...
.T.
...`,
		}
		loader.chunkCache["room"] = []*ChunkTemplate{chunk}

		// 4x4のマップに2x2のプレースホルダーしかない（サイズ不一致）
		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   [2]int{4, 4},
			Map: `....
.@A.
.@@.
....`,
			PlaceNested: []ChunkPlacement{
				{Chunks: []string{"room"}, ID: "A"}, // 2x2のプレースホルダーに3x3のチャンクは配置できない
			},
		}

		_, err := template.ExpandWithPlaceNested(loader, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "サイズが不一致")
	})

	t.Run("未登録のチャンクはエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   [2]int{5, 5},
			Map: `#####
#@@@#
#@@A#
#@@@#
#####`,
			PlaceNested: []ChunkPlacement{
				{Chunks: []string{"nonexistent"}, ID: "A"},
			},
		}

		_, err := template.ExpandWithPlaceNested(loader, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "見つかりません")
	})

	t.Run("複数候補から重みづけランダム選択", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 重み100の部屋（内部のみ）
		room1 := &ChunkTemplate{
			Name:   "room1",
			Weight: 100,
			Size:   [2]int{3, 3},
			Map: `...
.1.
...`,
		}
		loader.chunkCache["room1"] = []*ChunkTemplate{room1}

		// 重み50の部屋（内部のみ）
		room2 := &ChunkTemplate{
			Name:   "room2",
			Weight: 50,
			Size:   [2]int{3, 3},
			Map: `...
.2.
...`,
		}
		loader.chunkCache["room2"] = []*ChunkTemplate{room2}

		// 重み10の部屋（内部のみ）
		room3 := &ChunkTemplate{
			Name:   "room3",
			Weight: 10,
			Size:   [2]int{3, 3},
			Map: `...
.3.
...`,
		}
		loader.chunkCache["room3"] = []*ChunkTemplate{room3}

		// メインテンプレート（外壁を提供）
		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   [2]int{5, 5},
			Map: `#####
#@@@#
#@@A#
#@@@#
#####`,
			PlaceNested: []ChunkPlacement{
				{Chunks: []string{"room1", "room2", "room3"}, ID: "A"},
			},
		}

		// 同じシードで複数回実行すると同じ結果になることを確認
		result1, err := template.ExpandWithPlaceNested(loader, 12345)
		require.NoError(t, err)

		result2, err := template.ExpandWithPlaceNested(loader, 12345)
		require.NoError(t, err)

		assert.Equal(t, result1, result2, "同じシードで同じ結果が得られるべき")

		// 異なるシードで実行すると異なる可能性がある（確率的）
		result3, err := template.ExpandWithPlaceNested(loader, 99999)
		require.NoError(t, err)

		// いずれかのチャンクが選択されていることを確認
		assert.True(t,
			result3 == `#####
#...#
#.1.#
#...#
#####` ||
				result3 == `#####
#...#
#.2.#
#...#
#####` ||
				result3 == `#####
#...#
#.3.#
#...#
#####`,
			"いずれかのチャンクが選択されているべき")
	})
}

func TestTemplateLoader_RegisterAllChunks(t *testing.T) {
	t.Parallel()

	t.Run("ディレクトリからすべてのチャンクを登録できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 実際のディレクトリから読み込む
		err := loader.RegisterAllChunks([]string{
			"../../assets/levels/chunks",
		})
		require.NoError(t, err)

		// 登録されたチャンクを確認
		chunks, err := loader.GetChunks("3x3_bedroom")
		require.NoError(t, err)
		require.NotEmpty(t, chunks)
		assert.Equal(t, "3x3_bedroom", chunks[0].Name)
	})

	t.Run("存在しないディレクトリはエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		err := loader.RegisterAllChunks([]string{"nonexistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ディレクトリ読み込みエラー")
	})
}

func TestTemplateLoader_LoadTemplateByName(t *testing.T) {
	t.Parallel()

	t.Run("テンプレート名で展開済みテンプレートを取得できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// すべてのチャンクを登録
		err := loader.RegisterAllChunks([]string{
			"../../assets/levels/chunks",
			"../../assets/levels/facilities",
		})
		require.NoError(t, err)

		// すべてのパレットを登録
		err = loader.RegisterAllPalettes([]string{
			"../../assets/levels/palettes",
		})
		require.NoError(t, err)

		// 15x12_office_buildingを読み込み
		template, palette, err := loader.LoadTemplateByName("15x12_office_building", 12345)
		require.NoError(t, err)
		require.NotNil(t, template)
		require.NotNil(t, palette)

		// 展開されたマップにはチャンク文字（A, B, C, D）が含まれないはず
		// ただし、bedroomチャンクの"B"や、他のチャンクに含まれる大文字は残る可能性がある
		// ここでは展開が実行されたことを確認するため、マップが変更されたことをチェック
		assert.NotContains(t, template.Map, "AAAAA", "officeチャンクが展開されているべき")

		// パレットがマージされていることを確認
		_, ok := palette.GetTerrain(".")
		assert.True(t, ok, "標準パレットの地形が含まれているべき")
	})

	t.Run("存在しないテンプレート名はエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		_, _, err := loader.LoadTemplateByName("nonexistent", 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "見つかりません")
	})

	t.Run("チャンクなしのテンプレートも読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		err := loader.RegisterAllChunks([]string{
			"../../assets/levels/facilities",
		})
		require.NoError(t, err)

		err = loader.RegisterAllPalettes([]string{
			"../../assets/levels/palettes",
		})
		require.NoError(t, err)

		template, palette, err := loader.LoadTemplateByName("10x10_small_room", 0)
		require.NoError(t, err)
		require.NotNil(t, template)
		// 10x10_small_roomはパレット指定がないのでnilの可能性がある

		assert.Equal(t, "10x10_small_room", template.Name)

		// パレット指定がある場合のみチェック
		if len(template.Palettes) > 0 {
			require.NotNil(t, palette)
		}
	})
}
