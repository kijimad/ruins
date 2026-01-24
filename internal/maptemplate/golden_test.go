package maptemplate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestChunkExpansionGolden は各チャンクの展開結果をゴールデンファイルと比較する
func TestChunkExpansionGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		chunkName    string
		seed         uint64
		needsChunks  bool // 子チャンクが必要かどうか
		needsPalette bool // パレット登録が必要かどうか
	}{
		{
			name:         "13x8_small_office",
			chunkName:    "13x8_small_office",
			seed:         12345,
			needsChunks:  true,
			needsPalette: true,
		},
		{
			name:         "15x12_office_building",
			chunkName:    "15x12_office_building",
			seed:         12345,
			needsChunks:  true,
			needsPalette: true,
		},
		{
			name:         "10x10_small_room",
			chunkName:    "10x10_small_room",
			seed:         12345,
			needsChunks:  false,
			needsPalette: true,
		},
		{
			name:         "50x50_small_town",
			chunkName:    "50x50_small_town",
			seed:         12345,
			needsChunks:  true,
			needsPalette: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loader := NewTemplateLoader()

			// チャンクが必要な場合は読み込む
			if tt.needsChunks {
				err := loader.LoadChunk("../../assets/levels/chunks/rooms.toml")
				require.NoError(t, err)
			}

			// パレットが必要な場合は登録
			if tt.needsPalette {
				err := loader.RegisterAllPalettes([]string{"../../assets/levels/palettes"})
				require.NoError(t, err)
			}

			// 各チャンクタイプごとにファイルを読み込む
			switch tt.chunkName {
			case "13x8_small_office", "15x12_office_building":
				err := loader.LoadChunk("../../assets/levels/facilities/compound_building.toml")
				require.NoError(t, err)
			case "10x10_small_room":
				err := loader.LoadChunk("../../assets/levels/facilities/small_room.toml")
				require.NoError(t, err)
			case "50x50_small_town":
				// small_townは複合施設を含むので、先に読み込む
				err := loader.LoadChunk("../../assets/levels/facilities/compound_building.toml")
				require.NoError(t, err)
				err = loader.LoadChunk("../../assets/levels/layouts/small_town.toml")
				require.NoError(t, err)
			}

			// テンプレートを取得して展開
			template, _, err := loader.LoadTemplateByName(tt.chunkName, tt.seed)
			require.NoError(t, err)

			// ゴールデンファイルのパス
			goldenPath := filepath.Join("testdata", "golden", tt.name+".txt")

			// UPDATE_GOLDEN=1 が設定されている場合はゴールデンファイルを更新
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				err := os.MkdirAll(filepath.Dir(goldenPath), 0755)
				require.NoError(t, err)

				err = os.WriteFile(goldenPath, []byte(template.Map), 0644)
				require.NoError(t, err)
				t.Logf("Updated golden file: %s", goldenPath)
				return
			}

			// ゴールデンファイルを読み込んで比較
			golden, err := os.ReadFile(goldenPath)
			require.NoError(t, err, "ゴールデンファイルが見つかりません。UPDATE_GOLDEN=1 で生成してください")

			expected := strings.TrimSpace(string(golden))
			actual := strings.TrimSpace(template.Map)

			if expected != actual {
				t.Errorf("展開結果がゴールデンファイルと一致しません\n--- Expected ---\n%s\n--- Actual ---\n%s\n", expected, actual)
			}
		})
	}
}
