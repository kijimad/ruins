package maptemplate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExpandWithPlacementsRecursive は再帰的なチャンク展開をテストする
func TestExpandWithPlacementsRecursive(t *testing.T) {
	t.Parallel()

	t.Run("2階層のチャンク展開", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// Level 0: 基本的な部屋（3x3）
		roomChunk := &ChunkTemplate{
			Name:   "room",
			Size:   Size{W: 3, H: 3},
			Weight: 100,
			Map: `###
#.#
###`,
		}
		loader.chunkCache["room"] = []*ChunkTemplate{roomChunk}

		// Level 1: 建物（2つの部屋を含む 7x3）
		buildingChunk := &ChunkTemplate{
			Name:   "building",
			Size:   Size{W: 7, H: 3},
			Weight: 100,
			Map: `@@@.@@B
@@@.@@@
@@A.@@@`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"room"}, ID: "A"},
				{Chunks: []string{"room"}, ID: "B"},
			},
		}
		loader.chunkCache["building"] = []*ChunkTemplate{buildingChunk}

		// Level 2: 街区（建物を含む 7x3）
		blockTemplate := &ChunkTemplate{
			Name:   "block",
			Size:   Size{W: 7, H: 3},
			Weight: 100,
			Map: `@@@@@@C
@@@@@@@
@@@@@@@`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"building"}, ID: "C"},
			},
		}

		// 展開実行
		expanded, err := blockTemplate.ExpandWithPlacements(loader, 0)
		require.NoError(t, err)

		// 期待される結果
		expected := `###.###
#.#.#.#
###.###`

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
				Name:   chunkType,
				Size:   Size{W: 2, H: 2},
				Weight: 100,
				Map: `@A
@@`,
				Placements: []ChunkPlacement{
					{Chunks: []string{nextType}, ID: "A"},
				},
			}
			loader.chunkCache[chunkType] = []*ChunkTemplate{chunk}
		}

		// 最終レベル
		loader.chunkCache["level11"] = []*ChunkTemplate{&ChunkTemplate{
			Name:   "level11",
			Size:   Size{W: 2, H: 2},
			Weight: 100,
			Map: `..
..`,
		}}

		rootTemplate := &ChunkTemplate{
			Name:   "root",
			Size:   Size{W: 2, H: 2},
			Weight: 100,
			Map: `@A
@@`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"level0"}, ID: "A"},
			},
		}

		// 深度制限を超えるのでエラーになるはず
		_, err := rootTemplate.ExpandWithPlacements(loader, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "深度が制限")
	})

	t.Run("循環参照を検出", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		// チャンクAがチャンクBを参照
		chunkA := &ChunkTemplate{
			Name:   "chunk_a",
			Size:   Size{W: 2, H: 2},
			Weight: 100,
			Map: `@A
@@`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"chunk_b"}, ID: "A"},
			},
		}
		loader.chunkCache["chunk_a"] = []*ChunkTemplate{chunkA}

		// チャンクBがチャンクAを参照（循環）
		chunkB := &ChunkTemplate{
			Name:   "chunk_b",
			Size:   Size{W: 2, H: 2},
			Weight: 100,
			Map: `@A
@@`,
			Placements: []ChunkPlacement{
				{Chunks: []string{"chunk_a"}, ID: "A"},
			},
		}
		loader.chunkCache["chunk_b"] = []*ChunkTemplate{chunkB}

		// 循環参照を検出するはず
		_, err := chunkA.ExpandWithPlacements(loader, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "循環参照")
	})

	t.Run("チャンクなしの場合はそのまま返す", func(t *testing.T) {
		t.Parallel()
		loader := NewTemplateLoader()

		template := &ChunkTemplate{
			Name:   "simple",
			Size:   Size{W: 3, H: 3},
			Weight: 100,
			Map: `###
#.#
###`,
		}

		expanded, err := template.ExpandWithPlacements(loader, 0)
		require.NoError(t, err)

		expected := strings.TrimSpace(template.Map)
		assert.Equal(t, expected, expanded)
	})
}
