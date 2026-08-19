package resources

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpriteImage_SpriteRenderがnilならエラー(t *testing.T) {
	t.Parallel()

	img, err := SpriteImage(map[string]components.SpriteSheet{}, nil)

	require.ErrorContains(t, err, "sprite render is nil")
	assert.Nil(t, img)
}

func TestSpriteImage_シートが見つからなければエラー(t *testing.T) {
	t.Parallel()

	sr := &components.SpriteRender{SpriteSheetName: "missing", SpriteKey: "a"}

	img, err := SpriteImage(map[string]components.SpriteSheet{}, sr)

	require.ErrorContains(t, err, `sprite sheet "missing" not found`)
	assert.Nil(t, img)
}

func TestSpriteImage_キーが見つからなければエラー(t *testing.T) {
	t.Parallel()

	sheets := map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{},
		},
	}
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "missing"}

	img, err := SpriteImage(sheets, sr)

	require.ErrorContains(t, err, `sprite key "missing" not found in sheet "sheet"`)
	assert.Nil(t, img)
}

func TestSpriteImage_テクスチャ画像が無ければエラー(t *testing.T) {
	t.Parallel()

	sheets := map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: nil},
			Sprites: map[string]components.Sprite{
				"key": {X: 0, Y: 0, Width: 4, Height: 4},
			},
		},
	}
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img, err := SpriteImage(sheets, sr)

	require.ErrorContains(t, err, "has no texture image")
	assert.Nil(t, img)
}

func TestSpriteImage_矩形どおりに切り出す(t *testing.T) {
	t.Parallel()

	sheets := map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(20, 20)},
			Sprites: map[string]components.Sprite{
				"key": {X: 2, Y: 3, Width: 5, Height: 6},
			},
		},
	}
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img, err := SpriteImage(sheets, sr)

	require.NoError(t, err)
	require.NotNil(t, img)
	assert.Equal(t, 5, img.Bounds().Dx())
	assert.Equal(t, 6, img.Bounds().Dy())
}

func TestSpriteImage_テクスチャ範囲をはみ出す矩形はクランプされる(t *testing.T) {
	t.Parallel()

	sheets := map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{
				"key": {X: -5, Y: -5, Width: 20, Height: 20},
			},
		},
	}
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img, err := SpriteImage(sheets, sr)

	require.NoError(t, err)
	require.NotNil(t, img)
	assert.Equal(t, 10, img.Bounds().Dx())
	assert.Equal(t, 10, img.Bounds().Dy())
}

func TestSpriteImage_右端だけはみ出す矩形は右端だけクランプされる(t *testing.T) {
	t.Parallel()

	sheets := map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{
				"key": {X: 5, Y: 0, Width: 20, Height: 4},
			},
		},
	}
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img, err := SpriteImage(sheets, sr)

	require.NoError(t, err)
	require.NotNil(t, img)
	assert.Equal(t, 5, img.Bounds().Dx())
	assert.Equal(t, 4, img.Bounds().Dy())
}
