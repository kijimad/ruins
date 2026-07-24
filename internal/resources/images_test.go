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

func TestLoadGraphicImages_disabled未指定ならDisabledはnil(t *testing.T) {
	t.Parallel()

	gi, err := loadGraphicImages("assets/graphics/button-idle.png", "")

	require.NoError(t, err)
	assert.NotNil(t, gi.Idle)
	assert.Nil(t, gi.Disabled)
}

func TestLoadGraphicImages_disabled指定時は両方読み込む(t *testing.T) {
	t.Parallel()

	gi, err := loadGraphicImages("assets/graphics/button-idle.png", "assets/graphics/button-disabled.png")

	require.NoError(t, err)
	assert.NotNil(t, gi.Idle)
	assert.NotNil(t, gi.Disabled)
}

func TestLoadGraphicImages_idleが存在しなければエラー(t *testing.T) {
	t.Parallel()

	_, err := loadGraphicImages("assets/graphics/not-exist.png", "")

	require.Error(t, err)
	assert.ErrorContains(t, err, "assets/graphics/not-exist.png")
}

func TestLoadGraphicImages_disabledが存在しなければエラー(t *testing.T) {
	t.Parallel()

	_, err := loadGraphicImages("assets/graphics/button-idle.png", "assets/graphics/not-exist.png")

	require.Error(t, err)
	assert.ErrorContains(t, err, "assets/graphics/not-exist.png")
}

func TestLoadImageNineSlice_中央サイズを引いた最小サイズになる(t *testing.T) {
	t.Parallel()

	path := "assets/graphics/panel-idle.png"
	img, err := newImageFromFile(path)
	require.NoError(t, err)

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	centerWidth, centerHeight := w/2, h/2

	ns, err := loadImageNineSlice(path, centerWidth, centerHeight)

	require.NoError(t, err)
	minW, minH := ns.MinSize()
	// widths[0] は (w-center)/2 の整数除算結果になるが、widths[2] はその同じ値を
	// w-center から引いて求めるため、両端の合計は端数に関わらず常に w-centerWidth になる
	assert.Equal(t, w-centerWidth, minW)
	assert.Equal(t, h-centerHeight, minH)
}

func TestLoadImageNineSlice_存在しないパスはエラー(t *testing.T) {
	t.Parallel()

	_, err := loadImageNineSlice("assets/graphics/not-exist.png", 1, 1)

	require.Error(t, err)
	assert.ErrorContains(t, err, "assets/graphics/not-exist.png")
}
