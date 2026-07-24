package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetScreenDimensions_SetScreenDimensionsで設定した値を取得できる(t *testing.T) {
	t.Parallel()

	r := &Resources{}
	r.SetScreenDimensions(1920, 1080)

	w, h := r.GetScreenDimensions()
	assert.Equal(t, 1920, w)
	assert.Equal(t, 1080, h)
}

func TestInitGameResources_マップフィールドが空で初期化される(t *testing.T) {
	t.Parallel()

	r := InitGameResources()

	require.NotNil(t, r.SpriteSheets)
	require.NotNil(t, r.Fonts)
	require.NotNil(t, r.Faces)
	assert.Empty(t, r.SpriteSheets)
	assert.Empty(t, r.Fonts)
	assert.Empty(t, r.Faces)
	assert.Equal(t, ScreenDimensions{}, r.ScreenDimensions)
}

func TestInitializeResources_エラーなくフィールドを置き換える(t *testing.T) {
	t.Parallel()

	r := &Resources{}
	// 事前に値を入れておき、InitializeResources で上書きされることを確認する
	r.SetScreenDimensions(100, 100)

	err := r.InitializeResources()

	require.NoError(t, err)
	assert.NotNil(t, r.SpriteSheets)
	assert.Equal(t, ScreenDimensions{}, r.ScreenDimensions)
}
