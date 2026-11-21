package resources

import (
	"image"
	"image/color"
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

func TestSpriteStoreSized_同じキーは同じ画像を返す(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{
				Image:  ebiten.NewImage(40, 40),
				Source: image.NewRGBA(image.Rect(0, 0, 40, 40)),
			},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 40, Height: 40}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	first := s.Sized(sr, 20)
	require.NotNil(t, first)

	// 2度目はキャッシュから同じインスタンスが返る
	assert.Same(t, first, s.Sized(sr, 20))
}

func TestSpriteStoreSized_矩形が解決できなければnil(t *testing.T) {
	t.Parallel()
	s := newSpriteStore(t, map[string]components.SpriteSheet{})
	sr := &components.SpriteRender{SpriteSheetName: "missing", SpriteKey: "key"}

	assert.Nil(t, s.Sized(sr, 16))
}

func TestSpriteStoreSized_矩形の幅が0なら面積0でnil(t *testing.T) {
	t.Parallel()
	// Width=0のスプライトはspriteRectでMin.X==Max.Xの矩形になり、shrinkToFitが面積0として扱う
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{"key": {X: 3, Y: 0, Width: 0, Height: 5}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	assert.Nil(t, s.Sized(sr, 16))
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

func TestSpriteRect_SpriteRenderがnilならエラー(t *testing.T) {
	t.Parallel()

	_, _, err := spriteRect(map[string]components.SpriteSheet{}, nil)

	require.ErrorIs(t, err, errNilSpriteRender)
}

func TestSpriteStoreSized_縦長の矩形は縦を基準に縮める(t *testing.T) {
	t.Parallel()
	// 幅20・高さ40の縦長矩形をsize=20へ収める。高さ>=幅なので高さ基準でアスペクト比を保って縮める
	s := newSpriteStore(t, map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{
				Image:  ebiten.NewImage(20, 40),
				Source: image.NewRGBA(image.Rect(0, 0, 20, 40)),
			},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 20, Height: 40}},
		},
	})
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}

	img := s.Sized(sr, 20)

	require.NotNil(t, img)
	assert.Equal(t, 20, img.Bounds().Dy())
	assert.Equal(t, 10, img.Bounds().Dx())
}

func TestDrawGauge_満ちた段を色で残りを暗く描く(t *testing.T) {
	t.Parallel()
	dst := image.NewRGBA(image.Rect(0, 0, 19, 10))
	fill := color.RGBA{R: 1, G: 2, B: 3, A: 255}

	empty := color.RGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	drawGauge(dst, Gauge{Filled: 2, Total: 5, Fill: fill, Empty: empty})

	// 段は幅3で1ピクセル空けて並び、下端の3行に描く
	assert.Equal(t, fill, dst.RGBAAt(0, 9), "1段目は満ちた色")
	assert.Equal(t, fill, dst.RGBAAt(4, 7), "2段目は満ちた色")
	assert.Equal(t, empty, dst.RGBAAt(8, 9), "3段目は空きの色")
	assert.Equal(t, color.RGBA{}, dst.RGBAAt(3, 9), "段のあいだは空ける")
	assert.Equal(t, color.RGBA{}, dst.RGBAAt(0, 6), "ゲージの上は描かない")
}

func TestSizedGauge(t *testing.T) {
	t.Parallel()

	source := image.NewRGBA(image.Rect(0, 0, 8, 8))
	sheet := func(src image.Image) map[string]components.SpriteSheet {
		return map[string]components.SpriteSheet{"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(8, 8), Source: src},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 8, Height: 8}},
		}}
	}
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}
	gauge := Gauge{Filled: 3, Total: 5, Fill: color.RGBA{G: 255, A: 255}}

	t.Run("同じスプライトと段階なら同じ画像を使い回す", func(t *testing.T) {
		t.Parallel()
		s := newSpriteStore(t, sheet(source))
		img := s.SizedGauge(sr, 20, gauge)
		require.NotNil(t, img)
		assert.Same(t, img, s.SizedGauge(sr, 20, gauge))
		assert.NotSame(t, img, s.SizedGauge(sr, 20, Gauge{Filled: 2, Total: 5, Fill: gauge.Fill}))
	})
	t.Run("CPU 側の元画像が無ければゲージを焼かずに縮小画像を返す", func(t *testing.T) {
		t.Parallel()
		s := newSpriteStore(t, sheet(nil))
		assert.Same(t, s.Sized(sr, 20), s.SizedGauge(sr, 20, gauge))
	})
}
