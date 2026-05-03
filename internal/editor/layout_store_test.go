package editor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testLayoutTOML = `[[chunk]]
name = "3x3_test"
palettes = ["test_pal"]
weight = 100
map = """
###
#.#
###
"""
`

func TestLayoutStoreList(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layoutDir := filepath.Join(tmpDir, "layouts")
	require.NoError(t, os.MkdirAll(layoutDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layoutDir, "test.toml"), []byte(testLayoutTOML), 0o644))

	store, err := NewLayoutStore([]string{layoutDir})
	require.NoError(t, err)

	entries, err := store.List()
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.Equal(t, "layouts", entries[0].Dir)
	assert.Equal(t, "test.toml", entries[0].FileName)
	require.Len(t, entries[0].Chunks, 1)
	assert.Equal(t, "3x3_test", entries[0].Chunks[0].Name)
	assert.Equal(t, 100, entries[0].Chunks[0].Weight)
	assert.Equal(t, []string{"test_pal"}, entries[0].Chunks[0].Palettes)
	assert.Contains(t, entries[0].Chunks[0].Map, "###")
}

func TestLayoutStoreSaveChunk(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layoutDir := filepath.Join(tmpDir, "layouts")
	require.NoError(t, os.MkdirAll(layoutDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layoutDir, "test.toml"), []byte(testLayoutTOML), 0o644))

	store, err := NewLayoutStore([]string{layoutDir})
	require.NoError(t, err)

	newMap := "...\n...\n..."
	err = store.SaveChunk("layouts", "test.toml", "3x3_test", newMap)
	require.NoError(t, err)

	// 保存した内容を読み戻して検証する
	chunk, err := store.GetChunk("layouts", "test.toml", "3x3_test")
	require.NoError(t, err)
	assert.Equal(t, newMap, chunk.Map)
	assert.Equal(t, "3x3_test", chunk.Name)
}

func TestLayoutStoreGetChunk_NotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	layoutDir := filepath.Join(tmpDir, "layouts")
	require.NoError(t, os.MkdirAll(layoutDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(layoutDir, "test.toml"), []byte(testLayoutTOML), 0o644))

	store, err := NewLayoutStore([]string{layoutDir})
	require.NoError(t, err)

	_, err = store.GetChunk("layouts", "test.toml", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestLayoutStoreNewLayoutStore_InvalidDir(t *testing.T) {
	t.Parallel()

	_, err := NewLayoutStore([]string{"/nonexistent/path/that/does/not/exist"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ディレクトリが存在しません")
}

func TestFileKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "layouts/test", FileKey("layouts", "test.toml"))
	assert.Equal(t, "chunks/room", FileKey("chunks", "room.toml"))
	assert.Equal(t, "dir/file", FileKey("dir", "file"))
}
