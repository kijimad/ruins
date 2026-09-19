package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTempManager は一時ディレクトリを保存先にした SerializationManager を返す。
func newTempManager(t *testing.T) *save.SerializationManager {
	t.Helper()
	m, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	return m
}

// countAutoSaves はオートセーブスロット数を返す。
func countAutoSaves(t *testing.T, sm *save.SerializationManager) int {
	t.Helper()
	list, err := sm.ListAutoSaves()
	require.NoError(t, err)
	return len(list)
}

func TestAutoSave_ファイルを書く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.DisableAutoSave = false
	m := newTempManager(t)

	require.NoError(t, autoSave(world, m))

	assert.Equal(t, 1, countAutoSaves(t, m), "オートセーブファイルが1つできる")
}

func TestAutoSave_セーブ無効なら保存しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.SaveLoadEnabled = false
	world.Resources.Config.DisableAutoSave = false
	m := newTempManager(t)

	require.NoError(t, autoSave(world, m))

	assert.Equal(t, 0, countAutoSaves(t, m), "セーブ無効では保存しない")
}

func TestAutoSave_再生時は保存しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.SaveLoadEnabled = true
	world.Resources.Config.DisableAutoSave = true
	m := newTempManager(t)

	require.NoError(t, autoSave(world, m))

	assert.Equal(t, 0, countAutoSaves(t, m), "再生では保存しない")
}
