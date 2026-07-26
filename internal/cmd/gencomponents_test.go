package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// TestGenerateComponents_Golden は登録表からの生成結果を、ゴールデンである
// コミット済み components_gen.go と突き合わせる。テンプレートや登録表の変更が
// 生成物に反映されていることをローカルの make test で確認し、再生成漏れを検出する。
// generateComponents は format.Source を通すため、Goとして妥当でなければ require.NoError で落ちる。
func TestGenerateComponents_Golden(t *testing.T) {
	t.Parallel()

	got, err := generateComponents()
	require.NoError(t, err)

	want, err := os.ReadFile("../components/components_gen.go")
	require.NoError(t, err)

	assert.Equal(t, string(want), string(got),
		"components_gen.go が登録表と一致しない。`make generate` を実行すること")
}

// newGenComponentsApp はテストごとに独立したFlagインスタンスを持つコマンドを組み立てる。
// CmdGenComponentsをそのまま共有すると、並列実行時にFlagの内部パース状態が競合する。
func newGenComponentsApp() *cli.Command {
	return &cli.Command{
		Name: "ruins",
		Commands: []*cli.Command{
			{
				Name: "gencomponents",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "out", Value: "internal/components/components_gen.go"},
				},
				Action: runGenComponents,
			},
		},
	}
}

func TestRunGenComponents_出力ファイルに生成コードを書き込む(t *testing.T) {
	t.Parallel()

	outPath := filepath.Join(t.TempDir(), "components_gen.go")

	err := newGenComponentsApp().Run(context.Background(), []string{"ruins", "gencomponents", "--out", outPath})
	require.NoError(t, err)

	got, err := os.ReadFile(outPath)
	require.NoError(t, err)

	want, err := generateComponents()
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

func TestRunGenComponents_書き込み失敗時はエラーを返す(t *testing.T) {
	t.Parallel()

	// 存在しないディレクトリへの書き込みを指定してos.WriteFileを失敗させる
	outPath := filepath.Join(t.TempDir(), "no-such-dir", "components_gen.go")

	err := newGenComponentsApp().Run(context.Background(), []string{"ruins", "gencomponents", "--out", outPath})
	require.Error(t, err)
	assert.ErrorContains(t, err, "生成コードの書き込みに失敗した")
}
