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

func TestDepthNum_LiesFlat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		depth DepthNum
		want  bool
	}{
		{"床の段の草などは立て板で描く", DepthNumFloor, false},
		{"雪の層は段で描かない", DepthNumSnow, false},
		{"跡は寝かせる", DepthNumRemains, true},
		{"床の模様は寝かせる", DepthNumDecal, true},
		{"床に置くものは立てる", DepthNumRug, false},
		{"高さのあるものは立てる", DepthNumTaller, false},
		{"アイテムは立てる", DepthNumItem, false},
		{"プレイヤーは立てる", DepthNumPlayer, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.depth.LiesFlat())
		})
	}
}
