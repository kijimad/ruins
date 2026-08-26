package resources

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUIResources_正常系でフォントと素材が構築される(t *testing.T) {
	t.Parallel()

	src := newTestFaceSource(t)

	ui, err := NewUIResources([]*text.GoTextFaceSource{src})

	require.NoError(t, err)
	assert.NotNil(t, ui.Fonts)
	assert.NotNil(t, ui.GradientLine)
	assert.NotNil(t, ui.GaugeFill)
	assert.NotNil(t, ui.Text)
	assert.NotNil(t, ui.Text.SmallFace)
	assert.NotNil(t, ui.Text.BodyFace)
	assert.NotNil(t, ui.Text.KeycapFace)
	assert.NotNil(t, ui.Text.TitleFontFace)
	assert.NotNil(t, ui.Text.SplashFontFace)
}

func TestNewUIResources_フォントソースが空だとエラー(t *testing.T) {
	t.Parallel()

	ui, err := NewUIResources(nil)

	require.Error(t, err)
	require.ErrorIs(t, err, errNoFontSource)
	require.ErrorContains(t, err, "failed to load small font")
	assert.Equal(t, UIResources{}, ui)
}
