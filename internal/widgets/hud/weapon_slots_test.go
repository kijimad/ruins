package hud

import (
	"testing"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestWeaponSlots は face 無し、実 UIResources から作った Chrome を持つ WeaponSlots を返す。
// face は fakeCanvas が無視するため nil でよい。
func newTestWeaponSlots(t *testing.T) *WeaponSlots {
	t.Helper()
	res, err := loader.LoadUIResources()
	require.NoError(t, err)
	return NewWeaponSlots(nil, NewChrome(res))
}

// TestWeaponSlots_Draw_空スロットは何も描かない はスロットが無いとき早期 return することを検証する。
func TestWeaponSlots_Draw_空スロットは何も描かない(t *testing.T) {
	t.Parallel()

	ws := newTestWeaponSlots(t)
	world := testutil.InitTestWorld(t)
	cv := &fakeCanvas{}

	ws.Draw(cv, WeaponSlotsData{Slots: nil}, world)

	assert.Equal(t, 0, cv.nineSlices, "背景を描かない")
	assert.Empty(t, cv.texts, "番号を描かない")
	assert.Empty(t, cv.strokeRects, "枠線を描かない")
}

// TestWeaponSlots_Draw_各スロットを描画する は各スロットに背景と番号を描き、
// 選択中スロットに枠線を重ね、装備の有無でスプライト解決経路が分かれることを検証する。
func TestWeaponSlots_Draw_各スロットを描画する(t *testing.T) {
	t.Parallel()

	ws := newTestWeaponSlots(t)
	world := testutil.InitTestWorld(t)
	cv := &fakeCanvas{}

	// 武器スロットは仕様上最大5枠で番号は1桁に収まるため、番号は '0'+n で単純に描ける。
	// テスト用シートは画像を持たないので、装備ありでもスプライトは解決に失敗する。
	// ここでは WeaponName の有無で drawWeaponSprite の入口分岐が分かれることを通す
	data := WeaponSlotsData{
		Slots: []WeaponSlotInfo{
			// 装備あり。スプライト解決を試みるが画像が無く描画はしない
			{WeaponName: "剣", SpriteSheet: "field", SpriteName: "player"},
			// 装備あり、未知のスプライト名。解決に失敗し描画しない
			{WeaponName: "槍", SpriteSheet: "field", SpriteName: "unknown_sprite"},
			// 装備なし。スプライトは描かない
			{WeaponName: ""},
		},
		SelectedSlot:     0,
		ScreenDimensions: ScreenDimensions{Width: 1024, Height: 768},
	}

	ws.Draw(cv, data, world)

	// 各スロットに背景をNineSliceで描く
	assert.Equal(t, 3, cv.nineSlices, "スロット数ぶんの背景を描く")
	// 選択中スロットにだけ枠線を重ねる
	assert.Len(t, cv.strokeRects, 1, "選択スロットに枠線を1つ描く")
	// スロット番号を各スロットに描く。番号は1始まり。
	// 以降は cv.texts をスロット順に添字参照するため、先に長さを固定してから内容を見る
	require.Len(t, cv.texts, 3, "各スロットに番号を描く")
	assert.Equal(t, "1", cv.texts[0].str, "先頭スロットの番号は1")
	assert.Equal(t, "3", cv.texts[2].str, "末尾スロットの番号は3")
}
