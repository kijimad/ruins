package resources

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/components"
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
	assert.Empty(t, r.SpriteSheets)
	assert.Equal(t, ScreenDimensions{}, r.ScreenDimensions)
}

func TestSetSpriteSheets_シートと解決キャッシュの両方を更新する(t *testing.T) {
	t.Parallel()

	r := InitGameResources()
	sr := &components.SpriteRender{SpriteSheetName: "sheet", SpriteKey: "key"}
	sheetsA := map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 4, Height: 4}},
		},
	}
	sheetsB := map[string]components.SpriteSheet{
		"sheet": {
			Texture: components.Texture{Image: ebiten.NewImage(10, 10)},
			Sprites: map[string]components.Sprite{"key": {X: 0, Y: 0, Width: 4, Height: 4}},
		},
	}

	r.SetSpriteSheets(sheetsA)
	cachedUnderA := r.Sprites.Image(sr)
	require.NotNil(t, cachedUnderA)

	r.SetSpriteSheets(sheetsB)

	assert.Equal(t, sheetsB, r.SpriteSheets)
	// キャッシュが捨てられ、差し替え後のシートから改めて解決される。
	// ebiten.NewImageは呼び出しごとに新しいインスタンスを返すため、sheetsBのTextureはsheetsAと別ポインタになる
	resolvedUnderB := r.Sprites.Image(sr)
	require.NotNil(t, resolvedUnderB)
	assert.NotSame(t, cachedUnderA, resolvedUnderB)
}

func TestInitializeResources_エラーなくフィールドを置き換える(t *testing.T) {
	t.Parallel()

	r := &Resources{}
	// 事前に値を入れておき、InitializeResources で上書きされることを確認する
	r.SetScreenDimensions(100, 100)

	err := r.InitializeResources()

	require.NoError(t, err)
	assert.NotNil(t, r.SpriteSheets)
	// SetScreenDimensions で設定した値は *r 全体の置き換えにより消える
	w, h := r.GetScreenDimensions()
	assert.Zero(t, w)
	assert.Zero(t, h)
}
