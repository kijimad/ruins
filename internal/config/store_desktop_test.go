//go:build !js || !wasm

package config

import (
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
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

// 以下は XDG_CONFIG_HOME を書き換えてパス解決を検証するため、
// t.Setenv の制約上 t.Parallel は呼ばない。

func TestUserConfigPath_XDG_CONFIG_HOME配下にrunisディレクトリを作る(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := userConfigPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "ruins", "settings.toml"), path)
}

func TestWriteSettings_ReadSettings_SettingsExist_ラウンドトリップ(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	ok, err := settingsExist()
	require.NoError(t, err)
	assert.False(t, ok)

	_, ok, err = readSettings()
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, writeSettings([]byte("window_width = 1280\n")))

	ok, err = settingsExist()
	require.NoError(t, err)
	assert.True(t, ok)

	data, ok, err := readSettings()
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "window_width = 1280\n", string(data))
}

func TestSaveUserConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	c := &Config{User: UserConfig{WindowWidth: 1600, WindowHeight: 900, Language: "en"}}
	require.NoError(t, c.SaveUserConfig())

	data, ok, err := readSettings()
	require.NoError(t, err)
	require.True(t, ok)

	var got UserConfig
	require.NoError(t, toml.Unmarshal(data, &got))
	assert.Equal(t, c.User, got)
}

func TestEnsureUserConfigFile_ファイルが無ければデフォルト値で作成する(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	require.NoError(t, EnsureUserConfigFile())

	ok, err := settingsExist()
	require.NoError(t, err)
	assert.True(t, ok)

	data, _, err := readSettings()
	require.NoError(t, err)
	var got UserConfig
	require.NoError(t, toml.Unmarshal(data, &got))
	assert.Equal(t, DefaultUserConfig(), got)
}

func TestEnsureUserConfigFile_既にあれば上書きしない(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	require.NoError(t, writeSettings([]byte("window_width = 1280\n")))
	require.NoError(t, EnsureUserConfigFile())

	data, _, err := readSettings()
	require.NoError(t, err)
	assert.Equal(t, "window_width = 1280\n", string(data))
}

func TestLoadUserConfig_保存済み設定がなければ何もしない(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	c := &Config{User: DefaultUserConfig()}
	require.NoError(t, c.loadUserConfig())
	assert.Equal(t, DefaultUserConfig(), c.User)
}

func TestLoadUserConfig_保存済み設定を読み込んで上書きする(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, writeSettings([]byte("window_width = 1280\n")))

	c := &Config{User: DefaultUserConfig()}
	require.NoError(t, c.loadUserConfig())
	assert.Equal(t, 1280, c.User.WindowWidth)
	assert.Equal(t, 720, c.User.WindowHeight) // 保存に無いフィールドはデフォルトが残る
}

func TestLoadUserConfig_不正なTOMLはエラーを返す(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, writeSettings([]byte("window_width = [invalid")))

	c := &Config{User: DefaultUserConfig()}
	err := c.loadUserConfig()
	assert.ErrorContains(t, err, "設定の解析に失敗しました")
}

func TestLoad_RUINS_SEEDを指定すると再現可能なSeedになる(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("RUINS_SEED", "12345")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, uint64(12345), cfg.Seed)
}

func TestLoad_ユーザー設定ファイルが不正でもデフォルト値で継続する(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, writeSettings([]byte("window_width = [invalid")))

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, 960, cfg.User.WindowWidth) // 不正な設定は無視されデフォルトが残る
}
