package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// go testはカレントディレクトリをパッケージのソースディレクトリに設定するため、
// internal/cmd 配下には docs/design が存在しない。designdoc.DefaultDir は
// 相対パス "docs/design" 固定なので、以下は決定的にエラーになる

// TestRunDesignDocValidate_ディレクトリが存在しない場合はエラーになる はLoadDir失敗時に
// エラーがそのまま返ることを確認する
func TestRunDesignDocValidate_ディレクトリが存在しない場合はエラーになる(t *testing.T) {
	t.Parallel()

	err := runDesignDocValidate(context.Background(), &cli.Command{})

	require.Error(t, err)
	assert.ErrorContains(t, err, "docs/design")
}

// TestRunDesignDocGen_ディレクトリが存在しない場合はエラーになる はBackfillDir失敗時に
// エラーがそのまま返ることを確認する
func TestRunDesignDocGen_ディレクトリが存在しない場合はエラーになる(t *testing.T) {
	t.Parallel()

	err := runDesignDocGen(context.Background(), &cli.Command{})

	require.Error(t, err)
	assert.ErrorContains(t, err, "docs/design")
}

// TestRunDesignDocList_ディレクトリが存在しない場合はエラーになる はLoadDir失敗時に
// フラグを読む前にエラーがそのまま返ることを確認する
func TestRunDesignDocList_ディレクトリが存在しない場合はエラーになる(t *testing.T) {
	t.Parallel()

	err := runDesignDocList(context.Background(), &cli.Command{})

	require.Error(t, err)
	assert.ErrorContains(t, err, "docs/design")
}
