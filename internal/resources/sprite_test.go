package resources

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSpriteStore(t *testing.T, sheets map[string]components.SpriteSheet) *SpriteStore {
	t.Helper()
	s := NewSpriteStore()
	s.SetSheets(sheets)
	return s
}

func TestSpriteStoreImage_解決できなければnil(t *testing.T) {
	t.Parallel()

	t.Run("SpriteRenderがnil", func(t *testing.T) {
		t.Parallel()
		s := newSpriteStore(t, map[string]components.SpriteSheet{})
		assert.Nil(t, s.Image(nil))
	})

	t.Run("シートが見つからない", func(t *testing.T) {
		t.Parallel()
		s := newSpriteStore(t, map[string]components.SpriteSheet{})
		sr := &components.SpriteRender{SpriteSheetName: "missing", SpriteKey: "a"}
		assert.Nil(t, s.Image(sr))
	})

	t.Run("キーが見つからない", func(t *testing.T) {
		t.Parallel()
		s := newSpriteStore(t, map[string]components.SpriteSheet{
			"sheet": {
				Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
				Sprites: map[string]components.Sprite{},
			},
		})
		sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "missing"}
		assert.Nil(t, s.Image(sr))
	})

	t.Run("テクスチャ画像が無い", func(t *testing.T) {
		t.Parallel()
		s := newSpriteStore(t, map[string]components.SpriteSheet{
			"sheet": {
				Texture: components.Texture{Image: nil},
				Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 4, Height: 4}},
			},
		})
		sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}
		assert.Nil(t, s.Image(sr))
	})
}

func TestSpriteStoreImage_矩形どおりに切り出す(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(20, 20)},
			Sprites: map[string]components.Sprite{"key": {X: 2, Y: 3, Width: 5, Height: 6}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img := s.Image(sr)

	require.NotNil(t, img)
	assert.Equal(t, 5, img.Bounds().Dx())
	assert.Equal(t, 6, img.Bounds().Dy())
}

func TestSpriteStoreImage_テクスチャ範囲をはみ出す矩形はクランプされる(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{"key": {X: -5, Y: -5, Width: 20, Height: 20}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img := s.Image(sr)

	require.NotNil(t, img)
	assert.Equal(t, 10, img.Bounds().Dx())
	assert.Equal(t, 10, img.Bounds().Dy())
}

func TestSpriteStoreImage_右端だけはみ出す矩形は右端だけクランプされる(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{"key": {X: 5, Y: 0, Width: 20, Height: 4}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img := s.Image(sr)

	require.NotNil(t, img)
	assert.Equal(t, 5, img.Bounds().Dx())
	assert.Equal(t, 4, img.Bounds().Dy())
}

func TestSpriteStoreImage_同じキーは同じ画像を返す(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(20, 20)},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 8, Height: 8}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	// 2度目はキャッシュから同じインスタンスが返る
	first := s.Image(sr)
	assert.Same(t, first, s.Image(sr))
}

func TestSpriteStoreSized_収まっていれば等倍の部分画像を返す(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(20, 20)},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 8, Height: 8}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	// 8x8 は size=16 に収まるので、縮小せず元の寸法で返る
	img := s.Sized(sr, 16)

	require.NotNil(t, img)
	assert.Equal(t, 8, img.Bounds().Dx())
	assert.Equal(t, 8, img.Bounds().Dy())
}

func TestSpriteStoreSized_サイズが0以下ならnil(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}
	assert.Nil(t, s.Sized(sr, 0))
}

func TestSpriteStoreImage_等倍と縮小は別のキーで共存する(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{
				Image: ebiten.NewImage(40, 40),
				// Source は CPU 側の画像。縮小は At で画素を読むので image.RGBA を渡す
				Source: image.NewRGBA(image.Rect(0, 0, 40, 40)),
			},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 40, Height: 40}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	full := s.Image(sr)
	small := s.Sized(sr, 20)

	require.NotNil(t, full)
	require.NotNil(t, small)
	assert.Equal(t, 40, full.Bounds().Dx())
	assert.Equal(t, 20, small.Bounds().Dx())
}
