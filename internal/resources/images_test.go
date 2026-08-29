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
