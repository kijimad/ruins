package hud

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlotWidget_Draw(t *testing.T) {
	t.Parallel()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	chrome := Chrome{}
	rect := image.Rect(10, 20, 58, 68) // 48x48 のスロット矩形

	t.Run("空スロットは背景と番号を描き枠線は出さない", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		s := &slotWidget{rect: rect, chrome: chrome, face: res.Text.SmallFace, slot: HandSlotInfo{}, number: 1}
		s.Draw(cv)

		assert.Positive(t, cv.roundedFills, "背景パネルを敷く")
		txt := findText(t, cv.texts, "1")
		assert.Equal(t, rect.Min.X+slotNumberPad, txt.pos.X, "番号は左上パディング位置のX")
		assert.Equal(t, rect.Min.Y+slotNumberPad, txt.pos.Y, "番号は左上パディング位置のY")
		assert.Equal(t, 1, cv.roundedStrokes, "非選択は背景の枠だけで選択枠は描かない")
	})

	t.Run("残量を持つ物は下端に割合のぶんだけ塗ったゲージを描く", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		s := &slotWidget{rect: rect, chrome: chrome, face: res.Text.SmallFace, slot: HandSlotInfo{ChargeRatio: 0.5, HasCharge: true}, number: 1, gaugeFill: ebiten.NewImage(1, 1)}
		s.Draw(cv)

		// 欄の内側の下端6pxに枠を引き、枠の上下1px内側へ幅の半分を塗る
		assert.Equal(t, []image.Rectangle{image.Rect(14, 59, 34, 63)}, cv.tintedRects)
		assert.Equal(t, 2, cv.roundedStrokes, "背景の枠とゲージの枠")
	})

	t.Run("残量を持たない物はゲージを描かない", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		s := &slotWidget{rect: rect, chrome: chrome, face: res.Text.SmallFace, slot: HandSlotInfo{}, number: 1}
		s.Draw(cv)

		assert.Empty(t, cv.tintedRects)
		assert.Equal(t, 1, cv.roundedStrokes, "背景の枠だけでゲージの枠は描かない")
	})

	t.Run("選択中は矩形いっぱいに枠線を重ねる", func(t *testing.T) {
		t.Parallel()
		cv := &fakeCanvas{}
		s := &slotWidget{rect: rect, chrome: chrome, face: res.Text.SmallFace, slot: HandSlotInfo{}, number: 2, selected: true}
		s.Draw(cv)

		assert.Equal(t, 2, cv.roundedStrokes, "背景の枠に選択枠を重ねて2本")
	})
}

func TestHandSlots_Draw(t *testing.T) {
	t.Parallel()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	ws := NewHandSlots(res.Text.SmallFace, Chrome{}, nil)
	screen := ScreenDimensions{Width: 960, Height: 720}

	t.Run("スロット0件なら何も描かない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cv := &fakeCanvas{}
		ws.Draw(cv, HandSlotsData{Slots: nil, ScreenDimensions: screen}, world)
		assert.Empty(t, cv.texts)
		assert.Zero(t, cv.roundedFills)
	})

	t.Run("複数スロットは中央寄せで番号を左から順に並べる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cv := &fakeCanvas{}
		slots := []HandSlotInfo{{}, {}, {}}
		ws.Draw(cv, HandSlotsData{Slots: slots, ScreenDimensions: screen}, world)

		n1 := findText(t, cv.texts, "1")
		n2 := findText(t, cv.texts, "2")
		n3 := findText(t, cv.texts, "3")
		assert.Less(t, n1.pos.X, n2.pos.X, "番号は左から右へ並ぶ")
		assert.Less(t, n2.pos.X, n3.pos.X)
		assert.Equal(t, 3, cv.roundedFills, "スロットごとに背景を敷く")

		// 中央寄せ: 左端スロットの左に等しい余白が右端スロットの右にもある
		const slotSize, spacing = 48, 8
		totalWidth := 3*slotSize + 2*spacing
		wantStartX := (screen.Width - totalWidth) / 2
		assert.Equal(t, wantStartX+slotNumberPad, n1.pos.X, "先頭スロットは中央寄せの左端")
	})
}
