//go:build !js || !wasm

package save

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveDataImpl_書き込みに失敗しても既存のセーブを残す(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sm, err := NewSerializationManager(WithSaveDir(dir))
	require.NoError(t, err)
	require.NoError(t, sm.saveDataImpl("slot1", []byte("old")))

	// 一時ファイルの位置をディレクトリで塞ぎ、書き込みを失敗させる
	require.NoError(t, os.Mkdir(filepath.Join(dir, "slot1"+saveFileExt+".tmp"), 0o755))
	require.Error(t, sm.saveDataImpl("slot1", []byte("new")))

	got, err := os.ReadFile(filepath.Join(dir, "slot1"+saveFileExt))
	require.NoError(t, err)
	assert.Equal(t, "old", string(got))
}

func TestSaveDataImpl_上書きすると一時ファイルを残さない(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sm, err := NewSerializationManager(WithSaveDir(dir))
	require.NoError(t, err)
	require.NoError(t, sm.saveDataImpl("slot1", []byte("old")))
	require.NoError(t, sm.saveDataImpl("slot1", []byte("new")))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{"slot1" + saveFileExt}, names)
	got, err := os.ReadFile(filepath.Join(dir, "slot1"+saveFileExt))
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))
}
