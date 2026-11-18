package components

import (
	"image"
	"io/fs"
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

		require.ErrorIs(t, err, fs.ErrNotExist)
		assert.Nil(t, tex.Image)
		assert.Nil(t, tex.Source)
	})
}

func TestDepthNum_Rank(t *testing.T) {
	t.Parallel()

	order := []DepthNum{DepthNumFloor, DepthNumSnow, DepthNumRemains, DepthNumDecal, DepthNumRug, DepthNumTaller, DepthNumItem, DepthNumPlayer}
	for i := 1; i < len(order); i++ {
		assert.Less(t, order[i-1].Rank(), order[i].Rank(), "%s は %s より先に描く", order[i-1], order[i])
	}
	assert.Equal(t, DepthNumFloor.Rank(), DepthNum("unknown").Rank(), "未知の段は床の順にする")
}
