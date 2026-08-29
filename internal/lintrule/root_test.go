package lintrule_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kijimaD/ruins/internal/lintrule"
)

// moduleRoot は各検査がリポジトリを歩く起点を返す。
func moduleRoot(t *testing.T) string {
	t.Helper()

	root, err := lintrule.ModuleRoot()
	require.NoError(t, err)

	return root
}

func TestModuleRoot_go_modのあるディレクトリを返す(t *testing.T) {
	t.Parallel()

	root, err := lintrule.ModuleRoot()
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(root, "go.mod"))
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}
