//go:build !js || !wasm

package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteReadSettings_ラウンドトリップ(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.toml")
	require.NoError(t, writeSettingsTo(path, []byte("window_width = 1280\n")))
	assert.FileExists(t, path)

	data, ok, err := readSettingsFrom(path)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "window_width = 1280\n", string(data))
}

func TestReadSettings_ファイルが無ければ_ok_false(t *testing.T) {
	t.Parallel()

	_, ok, err := readSettingsFrom(filepath.Join(t.TempDir(), "settings.toml"))
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestWriteSettings_ディレクトリが無ければ作成する(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ruins", "settings.toml")
	require.NoError(t, writeSettingsTo(path, []byte("x")))
	assert.FileExists(t, path)
}

func TestSettingsExistAt(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.toml")

	ok, err := settingsExistAt(path)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, writeSettingsTo(path, []byte("x")))

	ok, err = settingsExistAt(path)
	require.NoError(t, err)
	assert.True(t, ok)
}
