package maptemplate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindPlaceholderRegionByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mapStr         string
		id             string
		expectedX      int
		expectedY      int
		expectedWidth  int
		expectedHeight int
		shouldError    bool
		errorMsg       string
	}{
		{
			name: "正常: 右下に識別子A",
			mapStr: `..........
.@@@A.....
.@@@@@....
.@@@@@....
..........`,
			id:             "A",
			expectedX:      1,
			expectedY:      1,
			expectedWidth:  4,
			expectedHeight: 3,
			shouldError:    false,
		},
		{
			name: "正常: 右下に識別子B (大きな領域)",
			mapStr: `..........
.@@@@@@@B.
.@@@@@@@@.
.@@@@@@@@.
.@@@@@@@@.
..........`,
			id:             "B",
			expectedX:      1,
			expectedY:      1,
			expectedWidth:  8,
			expectedHeight: 4,
			shouldError:    false,
		},
		{
			name: "正常: 1x1の領域",
			mapStr: `..........
.C........
..........`,
			id:             "C",
			expectedX:      1,
			expectedY:      1,
			expectedWidth:  1,
			expectedHeight: 1,
			shouldError:    false,
		},
		{
			name: "エラー: 識別子が見つからない",
			mapStr: `..........
.@@@@@....
.@@@@@....
..........`,
			id:          "Z",
			shouldError: true,
			errorMsg:    "識別子 'Z' が見つかりません",
		},
		{
			name: "エラー: 矩形が不完全 (途中で切れている)",
			mapStr: `..........
.@@@A.....
.@@.......
.@@@......
..........`,
			id:          "A",
			shouldError: true,
			errorMsg:    "不正な文字",
		},
		{
			name: "正常: マップの端に配置",
			mapStr: `@@@@@@@A
@@@@@@@@
@@@@@@@@`,
			id:             "A",
			expectedX:      0,
			expectedY:      0,
			expectedWidth:  8,
			expectedHeight: 3,
			shouldError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			template := &ChunkTemplate{Map: tt.mapStr}
			lines := template.GetMapLines()
			regions, err := findAllPlaceholderRegionsByID(lines, tt.id)

			if tt.shouldError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, regions)
				r := regions[0]
				require.Equal(t, tt.expectedX, r.x, "X座標が不一致")
				require.Equal(t, tt.expectedY, r.y, "Y座標が不一致")
				require.Equal(t, tt.expectedWidth, r.width, "幅が不一致")
				require.Equal(t, tt.expectedHeight, r.height, "高さが不一致")
			}
		})
	}
}

func TestFindAllPlaceholderRegionsByID(t *testing.T) {
	t.Parallel()

	t.Run("同じIDが複数箇所にある場合、全て検出する", func(t *testing.T) {
		t.Parallel()

		mapStr := `@@@@@.....@@@@@
@@@@@.....@@@@@
@@@@@.....@@@@@
@@@@@.....@@@@@
@@@@A.....@@@@A`

		template := &ChunkTemplate{Map: mapStr}
		lines := template.GetMapLines()
		regions, err := findAllPlaceholderRegionsByID(lines, "A")
		require.NoError(t, err)
		require.Len(t, regions, 2)

		// 1つ目の領域 (左側)
		require.Equal(t, 0, regions[0].x)
		require.Equal(t, 0, regions[0].y)
		require.Equal(t, 5, regions[0].width)
		require.Equal(t, 5, regions[0].height)

		// 2つ目の領域 (右側)
		require.Equal(t, 10, regions[1].x)
		require.Equal(t, 0, regions[1].y)
		require.Equal(t, 5, regions[1].width)
		require.Equal(t, 5, regions[1].height)
	})

	t.Run("IDが1箇所だけの場合、1つの領域を返す", func(t *testing.T) {
		t.Parallel()

		mapStr := `.........
.@@@A....
.@@@@....
.........`

		template := &ChunkTemplate{Map: mapStr}
		lines := template.GetMapLines()
		regions, err := findAllPlaceholderRegionsByID(lines, "A")
		require.NoError(t, err)
		require.Len(t, regions, 1)
		require.Equal(t, 1, regions[0].x)
		require.Equal(t, 1, regions[0].y)
		require.Equal(t, 4, regions[0].width)
		require.Equal(t, 2, regions[0].height)
	})
}

func TestValidatePlaceholders_WithID(t *testing.T) {
	t.Parallel()

	loader := NewTemplateLoader()

	// 3x2のチャンクを登録
	child := &ChunkTemplate{
		Name:   "child",
		Size:   Size{W: 3, H: 2},
		Weight: 100,
		Map:    "...\n...",
	}
	loader.chunkCache["child"] = []*ChunkTemplate{child}

	tests := []struct {
		name        string
		parentMap   string
		parentSize  Size
		placement   ChunkPlacement
		shouldError bool
		errorMsg    string
	}{
		{
			name: "正常: 識別子Aで3x2領域を指定",
			parentMap: `.........
.@@@.....
.@@A.....
.........`,
			parentSize: Size{W: 9, H: 4},
			placement: ChunkPlacement{
				Chunks: []string{"child"},
				ID:     "A",
			},
			shouldError: false,
		},
		{
			name: "エラー: サイズが不一致 (幅が大きい)",
			parentMap: `.........
.@@@@....
.@@@A....
.........`,
			parentSize: Size{W: 9, H: 4},
			placement: ChunkPlacement{
				Chunks: []string{"child"},
				ID:     "A",
			},
			shouldError: true,
			errorMsg:    "サイズが不一致",
		},
		{
			name: "エラー: サイズが不一致 (高さが大きい)",
			parentMap: `.........
.@@@.....
.@@@.....
.@@A.....
.........`,
			parentSize: Size{W: 9, H: 5},
			placement: ChunkPlacement{
				Chunks: []string{"child"},
				ID:     "A",
			},
			shouldError: true,
			errorMsg:    "サイズが不一致",
		},
		{
			name: "エラー: 識別子が見つからない",
			parentMap: `.........
.@@@.....
.@@@.....
.........`,
			parentSize: Size{W: 9, H: 4},
			placement: ChunkPlacement{
				Chunks: []string{"child"},
				ID:     "Z",
			},
			shouldError: true,
			errorMsg:    "識別子 'Z' が見つかりません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parentTemplate := &ChunkTemplate{
				Name:       "parent",
				Size:       tt.parentSize,
				Weight:     100,
				Map:        tt.parentMap,
				Placements: []ChunkPlacement{tt.placement},
			}

			err := parentTemplate.validatePlaceholders(loader)

			if tt.shouldError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExpandWithPlacements_WithID(t *testing.T) {
	t.Parallel()

	loader := NewTemplateLoader()

	// 3x2のチャンクを登録
	child := &ChunkTemplate{
		Name:   "child",
		Size:   Size{W: 3, H: 2},
		Weight: 100,
		Map:    "ABC\nDEF",
	}
	loader.chunkCache["child"] = []*ChunkTemplate{child}

	parentTemplate := &ChunkTemplate{
		Name:   "parent",
		Size:   Size{W: 7, H: 4},
		Weight: 100,
		Map: `.......
.@@@...
.@@A...
.......`,
		Placements: []ChunkPlacement{
			{
				Chunks: []string{"child"},
				ID:     "A",
			},
		},
	}

	loader.chunkCache["parent"] = []*ChunkTemplate{parentTemplate}

	expanded, err := parentTemplate.ExpandWithPlacements(loader, 12345)
	require.NoError(t, err)

	expected := `.......
.ABC...
.DEF...
.......`

	require.Equal(t, expected, cellsToString(expanded))
}
