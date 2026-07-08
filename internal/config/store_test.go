package config

import (
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
	assert.Equal(t, "ja", dst.User.Language)    // デフォルトが残る
}
