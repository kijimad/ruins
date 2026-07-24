package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateComponents_Golden は登録表からの生成結果を、ゴールデンである
// コミット済み components_gen.go と突き合わせる。テンプレートや登録表の変更が
// 生成物に反映されていること（再生成漏れの検出）をローカルの make test で確認する。
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

// TestRunGenComponents_出力ファイルへ書き込む はgencomponentsコマンドが--outで
// 指定したパスへ整形済みコードを書き込むことを確認する
//
//nolint:paralleltest // CmdGenComponentsはパッケージ変数で共有されており、Runのたびに--outフラグ値が書き換わるため並列実行しない
func TestRunGenComponents_出力ファイルへ書き込む(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "components_gen.go")

	err := CmdGenComponents.Run(context.Background(), []string{"gencomponents", "--out", outPath})
	require.NoError(t, err)

	got, err := os.ReadFile(outPath)
	require.NoError(t, err)

	want, err := generateComponents()
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

// TestRunGenComponents_書き込み先が存在しない場合はエラーになる は--outの親ディレクトリが
// 存在しないときにos.WriteFileのエラーがラップされて返ることを確認する
//
//nolint:paralleltest // CmdGenComponentsはパッケージ変数で共有されており、Runのたびに--outフラグ値が書き換わるため並列実行しない
func TestRunGenComponents_書き込み先が存在しない場合はエラーになる(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "no-such-dir", "components_gen.go")

	err := CmdGenComponents.Run(context.Background(), []string{"gencomponents", "--out", outPath})

	require.Error(t, err)
	assert.ErrorContains(t, err, "生成コードの書き込みに失敗した")
}
