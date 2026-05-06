package maptemplate

import (
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
name = "5x3_test_facility"
weight = 100
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
		assert.Equal(t, "5x3_test_facility", template.Name)
		assert.Equal(t, 100, template.Weight)
		assert.Equal(t, Size{W: 5, H: 3}, template.Size)
		assert.Equal(t, []string{"standard"}, template.Palettes)
	})

	t.Run("複数のテンプレートを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "3x3_small"
weight = 50
palettes = ["standard"]
map = """
###
#.#
###
"""

[[chunk]]
name = "5x5_large"
weight = 30
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
		assert.Equal(t, "3x3_small", templates[0].Name)
		assert.Equal(t, "5x5_large", templates[1].Name)
	})

	t.Run("nameが空の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = ""
weight = 100
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
		assert.Contains(t, err.Error(), "名前パースエラー")
	})

	t.Run("weightが0以下の場合はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "3x3_無効な重み"
weight = 0
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
name = "5x3_サイズ不一致"
weight = 100
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
name = "5x3_不規則マップ"
weight = 100
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
name = "3x3_multi_palette"
weight = 100
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
name = "3x3_no_palette"
weight = 100
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

	t.Run("Placementsを持つテンプレートを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "5x5_with_nested"
weight = 100
palettes = ["standard"]
map = """
#####
#@@@#
#@@A#
#@@@#
#####
"""

[[chunk.placements]]
chunks = ["room"]
id = "A"
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))

		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Len(t, templates[0].Placements, 1)
		assert.Equal(t, []string{"room"}, templates[0].Placements[0].Chunks)
		assert.Equal(t, "A", templates[0].Placements[0].ID)
	})

	t.Run("サイズ0はエラー", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "0x0_zero_size"
weight = 100
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
name = "3x3_negative_weight"
weight = -10
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
name = "1x1_empty_map"
weight = 100
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
name = "10x10_large_map"
weight = 100
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
		assert.Equal(t, Size{W: 10, H: 10}, templates[0].Size)
	})

	t.Run("マルチバイト文字を含むマップを読み込める", func(t *testing.T) {
		t.Parallel()
		content := `[[chunk]]
name = "5x3_japanese_map"
weight = 100
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
	t.Run("実ファイルからテンプレート定義を読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()
		templates, err := loader.LoadFile("levels/facilities/small_room.toml")

		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "10x10_small_room", templates[0].Name)
		assert.Equal(t, 100, templates[0].Weight)
		assert.NotEmpty(t, templates[0].Map)
	})

	t.Run("複数テンプレートを含むファイルを読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()
		templates, err := loader.LoadFile("levels/facilities/compound_building.toml")

		require.NoError(t, err)
		assert.True(t, len(templates) >= 2, "複合施設ファイルは複数テンプレートを含む")
	})

	t.Run("存在しないファイルはエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()
		_, err := loader.LoadFile("nonexistent.toml")

		require.Error(t, err)
	})

	t.Run("バリデーションエラーのテスト用TOMLを読み込む", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			content string
			errMsg  string
		}{
			{
				name: "nameが空の場合はエラー",
				content: `[[chunk]]
name = ""
weight = 100
palettes = ["standard"]
map = """
###
#.#
###
"""
`,
				errMsg: "名前パースエラー",
			},
			{
				name: "weightが0以下の場合はエラー",
				content: `[[chunk]]
name = "3x3_無効な重み"
weight = 0
palettes = ["standard"]
map = """
###
#.#
###
"""
`,
				errMsg: "重みは正の整数である必要があります",
			},
			{
				name: "マップサイズが定義と一致しない場合はエラー",
				content: `[[chunk]]
name = "5x3_サイズ不一致"
weight = 100
palettes = ["standard"]
map = """
###
#.#
###
"""
`,
				errMsg: "実サイズ",
			},
			{
				name: "マップの行長が不一致の場合はエラー",
				content: `[[chunk]]
name = "5x3_不規則マップ"
weight = 100
palettes = ["standard"]
map = """
#####
#..#
#####
"""
`,
				errMsg: "長さが不一致",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				loader := NewTemplateLoader()
				_, err := loader.Load(strings.NewReader(tt.content))
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
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

	t.Run("実ファイルからチャンク定義を読み込んでキャッシュできる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()
		err := loader.LoadChunk("levels/chunks/rooms.toml")
		require.NoError(t, err)

		// キャッシュから取得できるか確認
		chunks, err := loader.GetChunks("3x3_bedroom")
		require.NoError(t, err)
		require.NotEmpty(t, chunks)
		assert.Equal(t, "3x3_bedroom", chunks[0].Name)
		assert.Equal(t, Size{W: 3, H: 3}, chunks[0].Size)
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
		content := `[[chunk]]
name = "3x3_room"
weight = 100
palettes = ["standard"]
map = """
...
.1.
...
"""

[[chunk]]
name = "3x3_room"
weight = 50
palettes = ["standard"]
map = """
...
.2.
...
"""

[[chunk]]
name = "3x3_room"
weight = 10
palettes = ["standard"]
map = """
...
.3.
...
"""
`
		loader := NewTemplateLoader()
		templates, err := loader.Load(strings.NewReader(content))
		require.NoError(t, err)

		// 各テンプレートをキャッシュに登録
		for i := range templates {
			loader.chunkCache[templates[i].Name] = append(loader.chunkCache[templates[i].Name], &templates[i])
		}

		// GetChunksで3つのバリエーションを取得
		chunks, err := loader.GetChunks("3x3_room")
		require.NoError(t, err)
		assert.Len(t, chunks, 3)

		// 各バリエーションの確認
		assert.Equal(t, 100, chunks[0].Weight)
		assert.Equal(t, 50, chunks[1].Weight)
		assert.Equal(t, 10, chunks[2].Weight)

		// 最初のバリエーションを確認
		assert.Equal(t, "3x3_room", chunks[0].Name)
		assert.Contains(t, chunks[0].Map, "1")
	})
}

// cellsToString はセル配列をコンパクトな文字列に変換する。
// パレットなしのテストで、1文字Terrain名を文字列化する
func cellsToString(cells [][]MapCell) string {
	var sb strings.Builder
	for y, row := range cells {
		if y > 0 {
			sb.WriteByte('\n')
		}
		for _, cell := range row {
			if len(cell.Terrain) == 1 {
				sb.WriteString(cell.Terrain)
			} else {
				sb.WriteByte('?')
			}
		}
	}
	return sb.String()
}

func TestChunkTemplate_ExpandWithPlacements(t *testing.T) {
	t.Parallel()

	t.Run("placementsなしではそのまま返す", func(t *testing.T) {
		t.Parallel()
		template := ChunkTemplate{
			Map: `###
#.#
###`,
			Weight: 100,
			Size:   Size{W: 3, H: 3},
		}

		loader := NewTemplateLoader()
		result, err := template.ExpandWithPlacements(loader, 0)
		require.NoError(t, err)
		assert.Equal(t, strings.TrimSpace(template.Map), cellsToString(result))
	})

	t.Run("単一チャンクを展開できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// チャンクを登録（内部のみ、外壁なし）
		chunk := &ChunkTemplate{
			Name:   "room",
			Weight: 100,
			Size:   Size{W: 3, H: 3},
			Map: `...
.T.
...`,
		}
		loader.chunkCache["room"] = []*ChunkTemplate{chunk}

		// メインテンプレート（外壁を提供）
		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   Size{W: 5, H: 5},
			Map: `#####
#@@@#
#@@A#
#@@@#
#####`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"room"}, ID: "A"},
			},
		}

		result, err := template.ExpandWithPlacements(loader, 0)
		require.NoError(t, err)

		expected := `#####
#...#
#.T.#
#...#
#####`
		assert.Equal(t, expected, cellsToString(result))
	})

	t.Run("複数チャンクを展開できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 部屋チャンク（内部のみ、外壁なし）
		room := &ChunkTemplate{
			Name:   "room",
			Weight: 100,
			Size:   Size{W: 3, H: 3},
			Map: `...
.T.
...`,
		}
		loader.chunkCache["room"] = []*ChunkTemplate{room}

		// 倉庫チャンク（内部のみ、外壁なし）
		storage := &ChunkTemplate{
			Name:   "storage",
			Weight: 100,
			Size:   Size{W: 3, H: 3},
			Map: `...
.X.
...`,
		}
		loader.chunkCache["storage"] = []*ChunkTemplate{storage}

		// メインテンプレート（外壁と廊下を提供）
		template := ChunkTemplate{
			Name:   "compound",
			Weight: 100,
			Size:   Size{W: 9, H: 5},
			Map: `###+####+#
#@@@#.#@@@
#@@@#.#@@@
#@@A#.#@@B
#####.####`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"room"}, ID: "A"},
				{Chunks: []string{"storage"}, ID: "B"},
			},
		}

		result, err := template.ExpandWithPlacements(loader, 0)
		require.NoError(t, err)

		expected := `###+####+#
#...#.#...
#.T.#.#.X.
#...#.#...
#####.####`
		assert.Equal(t, expected, cellsToString(result))
	})

	t.Run("チャンクサイズが不一致の場合はエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 3x3のチャンク（内部のみ）
		chunk := &ChunkTemplate{
			Name:   "room",
			Weight: 100,
			Size:   Size{W: 3, H: 3},
			Map: `...
.T.
...`,
		}
		loader.chunkCache["room"] = []*ChunkTemplate{chunk}

		// 4x4のマップに2x2のプレースホルダーしかない（サイズ不一致）
		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   Size{W: 4, H: 4},
			Map: `....
.@A.
.@@.
....`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"room"}, ID: "A"}, // 2x2のプレースホルダーに3x3のチャンクは配置できない
			},
		}

		_, err := template.ExpandWithPlacements(loader, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "が不一致")
	})

	t.Run("未登録のチャンクはエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   Size{W: 5, H: 5},
			Map: `#####
#@@@#
#@@A#
#@@@#
#####`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"nonexistent"}, ID: "A"},
			},
		}

		_, err := template.ExpandWithPlacements(loader, 0)
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
			Size:   Size{W: 3, H: 3},
			Map: `...
.1.
...`,
		}
		loader.chunkCache["room1"] = []*ChunkTemplate{room1}

		// 重み50の部屋（内部のみ）
		room2 := &ChunkTemplate{
			Name:   "room2",
			Weight: 50,
			Size:   Size{W: 3, H: 3},
			Map: `...
.2.
...`,
		}
		loader.chunkCache["room2"] = []*ChunkTemplate{room2}

		// 重み10の部屋（内部のみ）
		room3 := &ChunkTemplate{
			Name:   "room3",
			Weight: 10,
			Size:   Size{W: 3, H: 3},
			Map: `...
.3.
...`,
		}
		loader.chunkCache["room3"] = []*ChunkTemplate{room3}

		// メインテンプレート（外壁を提供）
		template := ChunkTemplate{
			Name:   "building",
			Weight: 100,
			Size:   Size{W: 5, H: 5},
			Map: `#####
#@@@#
#@@A#
#@@@#
#####`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"room1", "room2", "room3"}, ID: "A"},
			},
		}

		// 同じシードで複数回実行すると同じ結果になることを確認
		result1, err := template.ExpandWithPlacements(loader, 12345)
		require.NoError(t, err)

		result2, err := template.ExpandWithPlacements(loader, 12345)
		require.NoError(t, err)

		assert.Equal(t, cellsToString(result1), cellsToString(result2), "同じシードで同じ結果が得られるべき")

		// 異なるシードで実行すると異なる可能性がある（確率的）
		result3, err := template.ExpandWithPlacements(loader, 99999)
		require.NoError(t, err)

		// いずれかのチャンクが選択されていることを確認（中央セルで判定）
		centerTerrain := result3[2][2].Terrain
		assert.True(t,
			centerTerrain == "1" || centerTerrain == "2" || centerTerrain == "3",
			"いずれかのチャンクが選択されているべき: got %s", centerTerrain)
	})
}

func TestTemplateLoader_RegisterAllChunks(t *testing.T) {
	t.Parallel()

	t.Run("ディレクトリからすべてのチャンクを登録できる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 実際のディレクトリから読み込む
		err := loader.RegisterAllChunks([]string{
			"levels/chunks",
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

		err := loader.RegisterAllChunks([]string{
			"levels/chunks",
			"levels/facilities",
		})
		require.NoError(t, err)

		err = loader.RegisterAllPalettes([]string{
			"levels/palettes",
		})
		require.NoError(t, err)

		template, palette, resolvedMap, err := loader.LoadTemplateByName("15x10_office_building", 12345)
		require.NoError(t, err)
		require.NotNil(t, template)
		require.NotNil(t, palette)
		require.NotNil(t, resolvedMap)

		// セル配列のサイズが正しいことを確認
		assert.Len(t, resolvedMap, template.Size.H)
		if len(resolvedMap) > 0 {
			assert.Len(t, resolvedMap[0], template.Size.W)
		}

		// パレットがマージされていることを確認
		_, ok := palette.GetTerrain(".")
		assert.True(t, ok, "標準パレットの地形が含まれているべき")
	})

	t.Run("子チャンクが独立にパレット解決される", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 子チャンク: '.' を floor, '#' を wall として定義
		childPalette := &Palette{
			ID:      "child_palette",
			Terrain: map[string]string{".": "floor", "#": "wall"},
		}
		loader.RegisterPalette(childPalette)

		childChunk := &ChunkTemplate{
			Name:     "3x3_child",
			Weight:   100,
			Size:     Size{W: 3, H: 3},
			Map:      "###\n#.#\n###\n",
			Palettes: []string{"child_palette"},
		}
		loader.RegisterChunk(childChunk)

		// 親チャンク: 'r' を floor として定義（子の '.' と同じ地形に異なる文字）
		parentPalette := &Palette{
			ID:      "parent_palette",
			Terrain: map[string]string{"r": "floor", "#": "wall"},
		}
		loader.RegisterPalette(parentPalette)

		parentChunk := &ChunkTemplate{
			Name:     "6x3_parent",
			Weight:   100,
			Size:     Size{W: 6, H: 3},
			Map:      "###@@@\n###@@@\n###@@A\n",
			Palettes: []string{"parent_palette"},
			Placements: []ChunkPlacement{
				{ID: "A", Chunks: []string{"3x3_child"}},
			},
		}
		loader.RegisterChunk(parentChunk)

		_, _, resolvedMap, err := loader.LoadTemplateByName("6x3_parent", 12345)
		require.NoError(t, err)

		// 子の '.' は子のパレットで "floor" に解決される
		// リマップ不要 — 各チャンクが独立にパレット解決する
		assert.Equal(t, "floor", resolvedMap[1][4].Terrain, "子の '.' は floor に解決されるべき")
		assert.Equal(t, "wall", resolvedMap[0][0].Terrain, "親の '#' は wall に解決されるべき")
	})

	t.Run("親と子で同じパレットなら同じ地形名になる", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		childPalette := &Palette{
			ID:      "child_pal",
			Terrain: map[string]string{".": "floor", "#": "wall"},
		}
		loader.RegisterPalette(childPalette)

		childChunk := &ChunkTemplate{
			Name:     "2x2_child",
			Weight:   100,
			Size:     Size{W: 2, H: 2},
			Map:      ".#\n#.\n",
			Palettes: []string{"child_pal"},
		}
		loader.RegisterChunk(childChunk)

		parentPalette := &Palette{
			ID:      "parent_pal",
			Terrain: map[string]string{".": "floor", "#": "wall"},
		}
		loader.RegisterPalette(parentPalette)

		parentChunk := &ChunkTemplate{
			Name:     "4x2_parent",
			Weight:   100,
			Size:     Size{W: 4, H: 2},
			Map:      "##@@\n##@A\n",
			Palettes: []string{"parent_pal"},
			Placements: []ChunkPlacement{
				{ID: "A", Chunks: []string{"2x2_child"}},
			},
		}
		loader.RegisterChunk(parentChunk)

		_, _, resolvedMap, err := loader.LoadTemplateByName("4x2_parent", 0)
		require.NoError(t, err)

		// 子のセルは floor/wall に解決される
		assert.Equal(t, "wall", resolvedMap[0][0].Terrain)
		assert.Equal(t, "floor", resolvedMap[0][2].Terrain) // 子の '.'
		assert.Equal(t, "wall", resolvedMap[0][3].Terrain)  // 子の '#'
		assert.Equal(t, "wall", resolvedMap[1][2].Terrain)  // 子の '#'
		assert.Equal(t, "floor", resolvedMap[1][3].Terrain) // 子の '.'
	})

	t.Run("存在しないテンプレート名はエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		_, _, _, err := loader.LoadTemplateByName("nonexistent", 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "見つかりません")
	})

	t.Run("チャンクなしのテンプレートも読み込める", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		err := loader.RegisterAllChunks([]string{
			"levels/facilities",
		})
		require.NoError(t, err)

		err = loader.RegisterAllPalettes([]string{
			"levels/palettes",
		})
		require.NoError(t, err)

		template, palette, resolvedMap, err := loader.LoadTemplateByName("10x10_small_room", 0)
		require.NoError(t, err)
		require.NotNil(t, template)
		require.NotNil(t, resolvedMap)

		assert.Equal(t, "10x10_small_room", template.Name)

		// パレット指定がある場合のみチェック
		if len(template.Palettes) > 0 {
			require.NotNil(t, palette)
		}
	})
}
