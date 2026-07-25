package resources

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUIResources_正常系で全リソースが構築される(t *testing.T) {
	t.Parallel()

	src := newTestFaceSource(t)

	ui, err := NewUIResources([]*text.GoTextFaceSource{src})

	require.NoError(t, err)
	assert.NotNil(t, ui.Fonts)
	assert.NotNil(t, ui.Background)
	assert.NotNil(t, ui.SeparatorColor)
	assert.NotNil(t, ui.GradientLine)
	assert.NotNil(t, ui.GaugeFill)
	assert.NotNil(t, ui.Text)
	assert.NotNil(t, ui.Text.SmallFace)
	assert.NotNil(t, ui.Text.BodyFace)
	assert.NotNil(t, ui.Text.TitleFontFace)
	assert.NotNil(t, ui.Text.SplashFontFace)
	assert.NotNil(t, ui.Button)
	assert.NotNil(t, ui.Button.Image)
	assert.NotNil(t, ui.Label)
	assert.NotNil(t, ui.Checkbox)
	assert.NotNil(t, ui.Checkbox.Image)
	assert.NotNil(t, ui.Checkbox.Graphic)
	assert.NotNil(t, ui.ComboButton)
	assert.NotNil(t, ui.ComboButton.Graphic)
	assert.NotNil(t, ui.List)
	assert.NotNil(t, ui.List.Image)
	assert.NotNil(t, ui.List.ImageTrans)
	assert.NotNil(t, ui.List.Track)
	assert.NotNil(t, ui.List.Handle)
	assert.NotNil(t, ui.Slider)
	assert.NotNil(t, ui.Slider.TrackImage)
	assert.NotNil(t, ui.Slider.Handle)
	assert.NotNil(t, ui.ProgressBar)
	assert.NotNil(t, ui.ProgressBar.TrackImage)
	assert.NotNil(t, ui.ProgressBar.FillImage)
	assert.NotNil(t, ui.Panel)
	assert.NotNil(t, ui.Panel.Image)
	assert.NotNil(t, ui.Panel.TitleBar)
	assert.NotNil(t, ui.Panel.SelectionBar)
	assert.NotNil(t, ui.TabBook)
	assert.NotNil(t, ui.TabBook.ButtonFace)
	assert.NotNil(t, ui.Header)
	assert.NotNil(t, ui.Header.Background)
	assert.NotNil(t, ui.TextInput)
	assert.NotNil(t, ui.TextInput.Image)
	assert.NotNil(t, ui.TextArea)
	assert.NotNil(t, ui.TextArea.Image)
	assert.NotNil(t, ui.ToolTip)
	assert.NotNil(t, ui.ToolTip.Background)
}

func TestNewUIResources_フォントソースが空だとエラー(t *testing.T) {
	t.Parallel()

	ui, err := NewUIResources(nil)

	require.Error(t, err)
	require.ErrorIs(t, err, errNoFontSource)
	require.ErrorContains(t, err, "failed to load small font")
	assert.Equal(t, UIResources{}, ui)
}

func TestHexToColor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		hex  string
		want color.NRGBA
	}{
		{"白", "ffffff", color.NRGBA{R: 255, G: 255, B: 255, A: 255}},
		{"黒", "000000", color.NRGBA{R: 0, G: 0, B: 0, A: 255}},
		{"赤", "ff0000", color.NRGBA{R: 255, G: 0, B: 0, A: 255}},
		{"緑", "00ff00", color.NRGBA{R: 0, G: 255, B: 0, A: 255}},
		{"青", "0000ff", color.NRGBA{R: 0, G: 0, B: 255, A: 255}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := hexToColor(tc.hex)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHexToColor_不正な16進数文字列はpanicする(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		hexToColor("zzzzzz")
	})
}
