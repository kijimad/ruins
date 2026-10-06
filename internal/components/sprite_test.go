package components

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubImage_部分矩形を切り出す(t *testing.T) {
	t.Parallel()

	img := ebiten.NewImage(10, 10)
	r := image.Rect(2, 3, 6, 9)

	sub := SubImage(img, r)

	require.NotNil(t, sub)
	assert.Equal(t, r, sub.Bounds())
}

func TestTexture_UnmarshalText(t *testing.T) {
	t.Parallel()

	t.Run("存在する画像を読み込める", func(t *testing.T) {
		t.Parallel()
		var tex Texture
		err := tex.UnmarshalText([]byte("file/textures/dist/bg.png"))

		require.NoError(t, err)
		assert.NotNil(t, tex.Image)
		assert.NotNil(t, tex.Source)
		assert.Equal(t, tex.Source.Bounds(), tex.Image.Bounds())
	})

	t.Run("存在しないパスはエラー", func(t *testing.T) {
		t.Parallel()
		var tex Texture
		err := tex.UnmarshalText([]byte("file/textures/dist/nonexistent.png"))

		require.Error(t, err)
		assert.Nil(t, tex.Image)
		assert.Nil(t, tex.Source)
	})
}
