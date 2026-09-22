package hud

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChrome_Panel(t *testing.T) {
	t.Parallel()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)

	chrome := NewChrome(res)
	cv := &fakeCanvas{}
	chrome.Panel(cv, image.Rect(0, 0, 100, 50))

	assert.Equal(t, 1, cv.roundedFills, "角丸の背景塗りを1回敷く")
	assert.Equal(t, 1, cv.roundedStrokes, "角丸の枠を1回描く")
}
