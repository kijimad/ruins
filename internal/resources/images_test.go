package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageFromFile_存在する画像を読み込める(t *testing.T) {
	t.Parallel()

	img, err := newImageFromFile("assets/graphics/button-idle.png")

	require.NoError(t, err)
	require.NotNil(t, img)
	assert.Positive(t, img.Bounds().Dx())
	assert.Positive(t, img.Bounds().Dy())
}

func TestNewImageFromFile_存在しないパスはエラー(t *testing.T) {
	t.Parallel()

	_, err := newImageFromFile("assets/graphics/not-exist.png")

	require.Error(t, err)
	assert.ErrorContains(t, err, "assets/graphics/not-exist.png")
}

func TestNewImageFromFile_画像として復号できなければエラー(t *testing.T) {
	t.Parallel()

	// ディレクトリはOpenできるがimage.Decodeが読めるバイト列を返さない
	_, err := newImageFromFile("assets/graphics")

	require.Error(t, err)
}

func TestNewNineSliceTex_画像読み込みに失敗したらエラーを伝播する(t *testing.T) {
	t.Parallel()

	tex, err := newNineSliceTex("assets/graphics/not-exist.png", 10, 10)

	require.ErrorContains(t, err, "assets/graphics/not-exist.png")
	assert.Nil(t, tex)
}
