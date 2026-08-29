package lintrule_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// moduleRoot は go.mod を上位へ辿ってリポジトリルートを返す。
// 検査はリポジトリ全体を歩くので、テストの実行位置に依存しない起点が要る。
func moduleRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod が見つからない")
		dir = parent
	}
}
