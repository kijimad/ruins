package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMainApp_サブコマンドを登録する はアプリの基本情報と登録済みサブコマンドを検証する
func TestNewMainApp_サブコマンドを登録する(t *testing.T) {
	t.Parallel()

	app := NewMainApp()

	assert.Equal(t, "ruins", app.Name)

	names := make([]string, 0, len(app.Commands))
	for _, c := range app.Commands {
		names = append(names, c.Name)
	}
	assert.ElementsMatch(t, []string{"play", "simulate-balance", "genreadme", "gencomponents", "designdoc"}, names)
}

// TestRunMainApp_ヘルプ表示は成功する はグローバルな--helpフラグがエラーにならないことを確認する
//
//nolint:paralleltest // NewMainAppはCmdPlay等パッケージ変数のサブコマンドを共有し、Runのたびにsetup処理が書き換わるため並列実行しない
func TestRunMainApp_ヘルプ表示は成功する(t *testing.T) {
	app := NewMainApp()

	err := RunMainApp(app, "ruins", "--help")

	require.NoError(t, err)
}

// TestRunMainApp_サブコマンドのエラーをラップする はサブコマンドが返したエラーが
// 「コマンド実行が失敗した」というプレフィックス付きでラップされることを確認する
//
//nolint:paralleltest // NewMainAppはCmdPlay等パッケージ変数のサブコマンドを共有し、Runのたびにsetup処理が書き換わるため並列実行しない
func TestRunMainApp_サブコマンドのエラーをラップする(t *testing.T) {
	app := NewMainApp()

	// docs/design はカレントディレクトリ内には存在しないため、
	// designdoc validate は決定的にエラーになる
	err := RunMainApp(app, "ruins", "designdoc", "validate")

	require.Error(t, err)
	require.ErrorContains(t, err, "コマンド実行が失敗した")
	require.ErrorContains(t, err, "docs/design")
}
