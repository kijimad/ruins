package cmd

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestFormatRaw(t *testing.T) {
	t.Parallel()

	src := `[[dropTables]]
name = "コウモリ"
id = "bat"
nothing = 0.9
[[dropTables.entries]]
weight = 1.0
id = "scrap_iron"
`
	got, err := formatRaw([]byte(src))
	require.NoError(t, err)
	assert.Equal(t, `[[dropTables]]
  id = "bat"
  name = "コウモリ"
  nothing = 0.9

  [[dropTables.entries]]
    id = "scrap_iron"
    weight = 1.0
`, string(got))
}

func TestFormatRaw_整形済みはそのまま(t *testing.T) {
	t.Parallel()

	src := `[[items]]
name = "a"
id = "a"
[items.melee]
damage = 1
`
	once, err := formatRaw([]byte(src))
	require.NoError(t, err)
	twice, err := formatRaw(once)
	require.NoError(t, err)
	assert.Equal(t, string(once), string(twice))
}

func TestFormatRaw_不正なTOMLはエラー(t *testing.T) {
	t.Parallel()

	_, err := formatRaw([]byte("id = "))
	require.Error(t, err)
}

// fmtRawApp は runFmtRaw を Action に持つ単体の cli.Command を返す。
// パッケージ変数 CmdFmtRaw を直接 Run すると実行ごとの状態が共有されるため、テストごとに新規に作る。
func fmtRawApp() *cli.Command {
	return &cli.Command{
		Name:      "fmtraw",
		ArgsUsage: "[path]",
		Action:    runFmtRaw,
	}
}

func TestRunFmtRaw_明示パスを整形して書き込む(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "raw.toml")
	require.NoError(t, os.WriteFile(path, []byte("[[items]]\nname = \"b\"\nid = \"a\"\n"), 0o644))

	err := fmtRawApp().Run(context.Background(), []string{"fmtraw", path})
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "[[items]]\n  id = \"a\"\n  name = \"b\"\n", string(got))
}

func TestRunFmtRaw_整形済みなら書き込まない(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "raw.toml")
	canonical := "[[items]]\n  id = \"a\"\n  name = \"b\"\n"
	// 書き込めないファイルにして、書き込もうとすればエラーになるようにする
	require.NoError(t, os.WriteFile(path, []byte(canonical), 0o444))

	err := fmtRawApp().Run(context.Background(), []string{"fmtraw", path})
	require.NoError(t, err)
}

func TestRunFmtRaw_存在しないファイルはエラー(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "no-such-file.toml")

	err := fmtRawApp().Run(context.Background(), []string{"fmtraw", path})
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestRunFmtRaw_不正なTOMLはエラー(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "raw.toml")
	require.NoError(t, os.WriteFile(path, []byte("id = "), 0o644))

	err := fmtRawApp().Run(context.Background(), []string{"fmtraw", path})
	var pe toml.ParseError
	require.ErrorAs(t, err, &pe)
}
