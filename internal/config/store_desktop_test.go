//go:build !js || !wasm

package config

import (
	"os"
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

func TestUserConfigPath_XDG_CONFIG_HOME配下にruinsディレクトリを作る(t *testing.T) {
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

	data, ok, err := readSettings()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "window_width = 1280\n", string(data))
}

func TestLoadUserConfig_保存済み設定がなければ何もしない(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	c := &Config{User: DefaultUserConfig()}
	err := c.loadUserConfig()
	require.NoError(t, err)
	assert.Equal(t, DefaultUserConfig(), c.User)
}

func TestLoadUserConfig_保存済み設定を読み込んで上書きする(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, writeSettings([]byte("window_width = 1280\n")))

	c := &Config{User: DefaultUserConfig()}
	err := c.loadUserConfig()
	require.NoError(t, err)
	assert.Equal(t, 1280, c.User.WindowWidth)
	assert.Equal(t, 720, c.User.WindowHeight) // 保存に無いフィールドはデフォルトが残る
}

func TestUserConfigPath_HOMEもXDG_CONFIG_HOMEも無ければエラーになる(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	_, err := userConfigPath()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get config directory")
}

func TestReadSettings_userConfigPathの解決に失敗するとエラーを返す(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	_, _, err := readSettings()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get config directory")
}

func TestWriteSettings_userConfigPathの解決に失敗するとエラーを返す(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	err := writeSettings([]byte("x"))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get config directory")
}

func TestReadSettingsFrom_ファイル以外が原因の読み込み失敗はエラーを返す(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// パスをファイルではなくディレクトリにして、ENOENT以外の理由で読み込みを失敗させる
	adir := filepath.Join(dir, "adir")
	require.NoError(t, os.Mkdir(adir, 0o755))

	_, _, err := readSettingsFrom(adir)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read config file")
}

func TestSettingsExistAt_ファイル以外が原因の確認失敗はエラーを返す(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// パス途中の要素をディレクトリではなくファイルにして、ENOENT以外の理由で確認を失敗させる
	notADir := filepath.Join(dir, "notadir")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0o644))
	path := filepath.Join(notADir, "settings.toml")

	_, err := settingsExistAt(path)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to check config file existence")
}

func TestWriteSettingsTo_ディレクトリ作成に失敗するとエラーを返す(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// パス途中の要素をディレクトリではなくファイルにして、MkdirAllを失敗させる
	notADir := filepath.Join(dir, "notadir")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0o644))
	path := filepath.Join(notADir, "sub", "settings.toml")

	err := writeSettingsTo(path, []byte("x"))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to create config directory")
}

func TestWriteSettingsTo_一時ファイルの書き込みに失敗するとエラーを返す(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	// 一時ファイルのパスを先にディレクトリとして作っておき、WriteFileを失敗させる
	require.NoError(t, os.Mkdir(path+".tmp", 0o755))

	err := writeSettingsTo(path, []byte("x"))

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to write config file")
}

func TestWriteSettingsTo_リネームに失敗するとエラーを返す(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	// 置き換え先を既存のディレクトリにして、Renameを失敗させる
	require.NoError(t, os.Mkdir(path, 0o755))

	err := writeSettingsTo(path, []byte("x"))

	require.ErrorContains(t, err, "failed to replace config file")
	assert.NoFileExists(t, path+".tmp") // 失敗時は一時ファイルを削除する
}

func TestLoadUserConfig_不正なTOMLはエラーを返す(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, writeSettings([]byte("window_width = [invalid")))

	c := &Config{User: DefaultUserConfig()}
	err := c.loadUserConfig()
	assert.ErrorContains(t, err, "failed to parse config")
}

func TestLoad_設定ファイルが無ければデフォルトのenになる(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	require.NoError(t, err)
	// EnsureUserConfigFile を通さず Load 単体では、ファイルが無いのでデフォルトの en が残る
	assert.Equal(t, "en", cfg.User.Language)
}

func TestLoad_languageの明示指定は尊重される(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, writeSettings([]byte("language = \"ja\"\n")))

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "ja", cfg.User.Language)
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
