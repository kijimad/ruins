package cmd

import (
	"os"
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
