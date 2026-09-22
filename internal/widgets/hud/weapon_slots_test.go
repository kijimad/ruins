package hud

import (
	"image"
	"testing"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlotWidget_Draw(t *testing.T) {
	t.Parallel()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	chrome := NewChrome(res)
	rect := image.Rect(10, 20, 58, 68) // 48x48 のスロット矩形

	t.Run("空スロットは背景と番号を描き枠線は出さない", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		s := &slotWidget{rect: rect, chrome: chrome, face: res.Text.SmallFace, slot: WeaponSlotInfo{}, number: 1}
		s.Draw(cv)

		assert.Positive(t, cv.nineSlices, "背景パネルを敷く")
		txt := findText(t, cv.texts, "1")
		assert.Equal(t, rect.Min.X+slotNumberPad, txt.pos.X, "番号は左上パディング位置のX")
		assert.Equal(t, rect.Min.Y+slotNumberPad, txt.pos.Y, "番号は左上パディング位置のY")
		assert.Empty(t, cv.strokeRects, "非選択なら選択枠は描かない")
	})

	t.Run("選択中は矩形いっぱいに枠線を重ねる", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		s := &slotWidget{rect: rect, chrome: chrome, face: res.Text.SmallFace, slot: WeaponSlotInfo{}, number: 2, selected: true}
		s.Draw(cv)

		assert.Contains(t, cv.strokeRects, rect, "選択枠をスロット矩形いっぱいに描く")
	})
}
