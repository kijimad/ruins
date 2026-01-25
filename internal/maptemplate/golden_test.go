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

	// 全アセットを一度にロード
	loader := NewTemplateLoader()
	err := loader.RegisterAllPalettes([]string{"../../assets/levels/palettes"})
	require.NoError(t, err)
	err = loader.RegisterAllChunks([]string{
		"../../assets/levels/chunks",
		"../../assets/levels/facilities",
		"../../assets/levels/layouts",
	})
	require.NoError(t, err)

	tests := []string{
		"11x6_small_office",
		"15x10_office_building",
		"10x10_small_room",
		"50x50_small_town",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// テンプレートを取得して展開
			template, _, err := loader.LoadTemplateByName(name, 12345)
			require.NoError(t, err)

			// ゴールデンファイルのパス
			goldenPath := filepath.Join("testdata", "golden", name+".txt")

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
