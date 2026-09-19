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

func TestAutoSaver_saveがファイルを書く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	m := newTempManager(t)
	a := &autoSaver{manager: m}

	a.save(world)

	assert.Equal(t, 1, countAutoSaves(t, m), "save でオートセーブファイルが1つできる")
}

func TestAutoSaver_マネージャが無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	a := &autoSaver{} // manager は nil

	assert.NotPanics(t, func() { a.save(world) }, "no-op で panic しない")
}

func TestNewAutoSaver_セーブ無効なら無効化する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.SaveLoadEnabled = false
	world.Resources.Config.DisableAutoSave = false // SaveLoadEnabled だけを要因に絞る

	a := newAutoSaver(world)

	assert.Nil(t, a.manager, "セーブ無効ではマネージャを持たない")
}

func TestNewAutoSaver_再生時は無効化する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	world.Resources.Config.SaveLoadEnabled = true
	world.Resources.Config.DisableAutoSave = true // DisableAutoSave だけを要因に絞る

	a := newAutoSaver(world)

	assert.Nil(t, a.manager, "再生ではマネージャを持たず副作用を出さない")
}
