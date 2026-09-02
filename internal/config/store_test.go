package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserConfigEncodeDecode_ラウンドトリップ(t *testing.T) {
	t.Parallel()

	src := &Config{User: UserConfig{WindowWidth: 1280, WindowHeight: 960, Language: "en"}}
	data, err := src.encodeUserConfig()
	require.NoError(t, err)

	dst := &Config{User: DefaultUserConfig()}
	require.NoError(t, toml.Unmarshal(data, &dst.User))
	assert.Equal(t, src.User, dst.User)
}

func TestUserConfig_保存に無いフィールドはデフォルトが残る(t *testing.T) {
	t.Parallel()

	// window_width だけを持つ設定を土台のデフォルトに復元する
	dst := &Config{User: DefaultUserConfig()}
	require.NoError(t, toml.Unmarshal([]byte("window_width = 1024\n"), &dst.User))

	assert.Equal(t, 1024, dst.User.WindowWidth) // 保存値で上書き
	assert.Equal(t, 720, dst.User.WindowHeight) // デフォルトが残る
	assert.Equal(t, "en", dst.User.Language)    // デフォルトが残る
}

// t.Setenv で XDG_CONFIG_HOME を書き換えるため t.Parallel は呼ばない。
func TestLoadUserConfig_読み込みエラーをそのまま返す(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// 設定ファイルの位置をディレクトリにして、ENOENT以外の理由で読み込みを失敗させる
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "ruins", "settings.toml"), 0o755))

	c := &Config{User: DefaultUserConfig()}
	err := c.loadUserConfig()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to read config file")
}

// t.Setenv で HOME/XDG_CONFIG_HOME を書き換えるため t.Parallel は呼ばない。
func TestEnsureUserConfigFile_存在確認に失敗するとエラーを返す(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	err := EnsureUserConfigFile()

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to get config directory")
}
