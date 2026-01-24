package maptemplate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExpandWithChunksRecursive は再帰的なチャンク展開をテストする
func TestExpandWithChunksRecursive(t *testing.T) {
	t.Parallel()

	t.Run("2階層のチャンク展開", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// Level 0: 基本的な部屋（3x3）
		roomChunk := &ChunkTemplate{
			Name:   "room",
			Size:   [2]int{3, 3},
			Weight: 100,
			Map: `###
#.#
###`,
		}
		loader.chunkCache["room"] = roomChunk

		// Level 1: 建物（2つの部屋を含む 6x3）
		buildingChunk := &ChunkTemplate{
			Name:         "building",
			Size:         [2]int{6, 3},
			Weight:       100,
			Chunks:       []string{"room"},
			ChunkMapping: map[string][]string{"A": {"room"}, "B": {"room"}},
			Map: `AAABBB
AAABBB
AAABBB`,
		}
		loader.chunkCache["building"] = buildingChunk

		// Level 2: 街区（建物を含む 6x3）
		blockTemplate := &ChunkTemplate{
			Name:         "block",
			Size:         [2]int{6, 3},
			Weight:       100,
			Chunks:       []string{"building"},
			ChunkMapping: map[string][]string{"X": {"building"}},
			Map: `XXXXXX
XXXXXX
XXXXXX`,
		}

		// 展開実行
		expanded, err := blockTemplate.ExpandWithChunks(loader, 0)
		require.NoError(t, err)

		// 期待される結果
		expected := `######
#.##.#
######`

		assert.Equal(t, expected, expanded)
	})

	t.Run("深度制限でエラー", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// 深すぎる階層構造を作成 (maxDepth=10を超える11階層)
		for i := 0; i < 11; i++ {
			chunkType := "level" + string(rune('0'+i))
			nextType := "level" + string(rune('1'+i))

			chunk := &ChunkTemplate{
				Name:         chunkType,
				Size:         [2]int{2, 2},
				Weight:       100,
				Chunks:       []string{nextType},
				ChunkMapping: map[string][]string{"X": {nextType}},
				Map: `XX
XX`,
			}
			loader.chunkCache[chunkType] = chunk
		}

		// 最終レベル
		loader.chunkCache["level11"] = &ChunkTemplate{
			Name:   "level11",
			Size:   [2]int{2, 2},
			Weight: 100,
			Map: `..
..`,
		}

		rootTemplate := &ChunkTemplate{
			Name:         "root",
			Size:         [2]int{2, 2},
			Weight:       100,
			Chunks:       []string{"level0"},
			ChunkMapping: map[string][]string{"X": {"level0"}},
			Map: `XX
XX`,
		}

		// 深度制限を超えるのでエラーになるはず
		_, err := rootTemplate.ExpandWithChunks(loader, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "深度が制限")
	})

	t.Run("循環参照を検出", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// チャンクAがチャンクBを参照
		chunkA := &ChunkTemplate{
			Name:         "chunk_a",
			Size:         [2]int{2, 2},
			Weight:       100,
			Chunks:       []string{"chunk_b"},
			ChunkMapping: map[string][]string{"B": {"chunk_b"}},
			Map: `BB
BB`,
		}
		loader.chunkCache["chunk_a"] = chunkA

		// チャンクBがチャンクAを参照（循環）
		chunkB := &ChunkTemplate{
			Name:         "chunk_b",
			Size:         [2]int{2, 2},
			Weight:       100,
			Chunks:       []string{"chunk_a"},
			ChunkMapping: map[string][]string{"A": {"chunk_a"}},
			Map: `AA
AA`,
		}
		loader.chunkCache["chunk_b"] = chunkB

		// 循環参照を検出するはず
		_, err := chunkA.ExpandWithChunks(loader, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "循環参照")
	})

	t.Run("チャンクなしの場合はそのまま返す", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		template := &ChunkTemplate{
			Name:   "simple",
			Size:   [2]int{3, 3},
			Weight: 100,
			Map: `###
#.#
###`,
		}

		expanded, err := template.ExpandWithChunks(loader, 0)
		require.NoError(t, err)

		expected := strings.TrimSpace(template.Map)
		assert.Equal(t, expected, expanded)
	})
}
