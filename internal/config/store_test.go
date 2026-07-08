package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUserConfig_ファイルが無ければデフォルトを維持する(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.toml")
	cfg := &Config{User: DefaultUserConfig()}

	require.NoError(t, cfg.loadUserConfigFrom(path))
	assert.Equal(t, DefaultUserConfig(), cfg.User)
}

func TestSaveUserConfig_書き込んだ値を読み戻せる(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.toml")
	saved := &Config{User: UserConfig{WindowWidth: 1280, WindowHeight: 960}}
	require.NoError(t, saved.saveUserConfigTo(path))
	assert.FileExists(t, path)

	loaded := &Config{User: DefaultUserConfig()}
	require.NoError(t, loaded.loadUserConfigFrom(path))
	assert.Equal(t, saved.User, loaded.User)
}

func TestLoadUserConfig_ファイルに無いフィールドはデフォルトが残る(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.toml")
	// window_width だけを持つ設定ファイルを用意する
	require.NoError(t, os.WriteFile(path, []byte("window_width = 1024\n"), 0o644))

	cfg := &Config{User: DefaultUserConfig()}
	require.NoError(t, cfg.loadUserConfigFrom(path))

	assert.Equal(t, 1024, cfg.User.WindowWidth) // ファイルの値で上書き
	assert.Equal(t, 720, cfg.User.WindowHeight) // デフォルトが残る
}

func TestSaveUserConfig_ディレクトリが無ければ作成する(t *testing.T) {
	t.Parallel()

	// 未作成のサブディレクトリを含むパス
	path := filepath.Join(t.TempDir(), "ruins", "settings.toml")
	cfg := &Config{User: DefaultUserConfig()}

	require.NoError(t, cfg.saveUserConfigTo(path))
	assert.FileExists(t, path)
}

func TestEnsureUserConfigFile_無ければデフォルトで作成する(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "ruins", "settings.toml")
	require.NoError(t, ensureUserConfigFileAt(path))
	assert.FileExists(t, path)

	// 作成された内容はデフォルト値
	cfg := &Config{}
	require.NoError(t, cfg.loadUserConfigFrom(path))
	assert.Equal(t, DefaultUserConfig(), cfg.User)
}

func TestEnsureUserConfigFile_既存ファイルを上書きしない(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "settings.toml")
	// ユーザーが変更済みの設定を用意する
	existing := &Config{User: UserConfig{WindowWidth: 1920, WindowHeight: 1080}}
	require.NoError(t, existing.saveUserConfigTo(path))

	require.NoError(t, ensureUserConfigFileAt(path))

	// 既存の値が保持され、デフォルトで上書きされない
	cfg := &Config{}
	require.NoError(t, cfg.loadUserConfigFrom(path))
	assert.Equal(t, 1920, cfg.User.WindowWidth)
	assert.Equal(t, 1080, cfg.User.WindowHeight)
}
