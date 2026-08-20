package resources

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFaceSource(t *testing.T) *text.GoTextFaceSource {
	t.Helper()

	f, err := NewFont("file/fonts/dougenzaka/KH-Dot-Dougenzaka-16.ttf")
	require.NoError(t, err)

	return f.FaceSource
}

func TestLoadFont_ソースが空ならエラー(t *testing.T) {
	t.Parallel()

	_, err := loadFont(nil, 16)

	require.Error(t, err)
	assert.ErrorIs(t, err, errNoFontSource)
}

func TestLoadFont_要素が全てnilならエラー(t *testing.T) {
	t.Parallel()

	_, err := loadFont([]*text.GoTextFaceSource{nil, nil}, 16)

	require.Error(t, err)
	assert.ErrorIs(t, err, errNoFontSource)
}

func TestLoadFont_単一ソースはそのままFaceを返す(t *testing.T) {
	t.Parallel()

	src := newTestFaceSource(t)

	face, err := loadFont([]*text.GoTextFaceSource{src}, 16)

	require.NoError(t, err)
	assert.NotNil(t, face)
}

func TestLoadFont_複数ソースはMultiFaceにフォールバックする(t *testing.T) {
	t.Parallel()

	src1 := newTestFaceSource(t)
	src2 := newTestFaceSource(t)

	face, err := loadFont([]*text.GoTextFaceSource{src1, src2}, 16)

	require.NoError(t, err)
	assert.NotNil(t, face)
	// 複数ソース指定時は text.NewMultiFace が返す *text.MultiFace になる
	_, ok := face.(*text.MultiFace)
	assert.True(t, ok)
}

func TestLoadFonts_成功時は4サイズ分のFaceを持つ(t *testing.T) {
	t.Parallel()

	src := newTestFaceSource(t)

	fs, err := loadFonts([]*text.GoTextFaceSource{src})

	require.NoError(t, err)
	// fonts は非公開型で外部から個別サイズを取り出す手段がないため、同一パッケージのテストとしてフィールドへ直接アクセスする
	assert.NotNil(t, fs.smallFace)
	assert.NotNil(t, fs.bodyFace)
	assert.NotNil(t, fs.titleFontFace)
	assert.NotNil(t, fs.splashFontFace)
}

func TestLoadFonts_ソースが空ならエラー(t *testing.T) {
	t.Parallel()

	_, err := loadFonts(nil)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to load small font")
	assert.ErrorIs(t, err, errNoFontSource)
}
