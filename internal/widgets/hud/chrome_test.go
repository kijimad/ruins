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

	t.Run("テクスチャがあれば9スライスで矩形へ敷く", func(t *testing.T) {
		t.Parallel()
		res, err := loader.LoadUIResources()
		require.NoError(t, err)

		chrome := NewChrome(res)
		cv := &fakeCanvas{}
		chrome.Panel(cv, image.Rect(0, 0, 100, 50))

		assert.Equal(t, 1, cv.nineSlices)
	})

	t.Run("テクスチャが無ければ何も描かない", func(t *testing.T) {
		t.Parallel()
		chrome := Chrome{}
		cv := &fakeCanvas{}
		chrome.Panel(cv, image.Rect(0, 0, 10, 10))

		assert.Equal(t, 0, cv.nineSlices)
	})
}
