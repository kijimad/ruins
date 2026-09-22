package hud

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChrome_Panel(t *testing.T) {
	t.Parallel()
	chrome := Chrome{}
	cv := &fakeCanvas{}
	chrome.Panel(cv, image.Rect(0, 0, 100, 50))

	assert.Equal(t, 1, cv.roundedFills, "角丸の背景塗りを1回敷く")
	assert.Equal(t, 1, cv.roundedStrokes, "角丸の枠を1回描く")
}
