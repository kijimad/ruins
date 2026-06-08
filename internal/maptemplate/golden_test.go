package maptemplate

import (
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

// TestChunkExpansionGolden は各チャンクの展開結果をゴールデンファイルと比較する。
func TestChunkExpansionGolden(t *testing.T) {
	t.Parallel()

	// 全アセットを一度にロード
	loader := NewTemplateLoader()
	err := loader.RegisterAllPalettes([]string{"levels/palettes"})
	require.NoError(t, err)
	err = loader.RegisterAllChunks([]string{
		"levels/chunks",
		"levels/facilities",
		"levels/layouts",
	})
	require.NoError(t, err)

	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata/golden"),
		goldie.WithNameSuffix(".txt"),
		goldie.WithDiffEngine(goldie.ColoredDiff),
	)

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
			_, _, resolvedMap, err := loader.LoadTemplateByName(name, 12345)
			require.NoError(t, err)

			// セル配列を可読文字列に変換
			actual := strings.TrimSpace(FormatResolvedMap(resolvedMap))

			g.Assert(t, name, []byte(actual))
		})
	}
}
