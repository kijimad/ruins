package hud

import (
	"image"
	"image/color"
	"testing"

	theme "github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/stretchr/testify/assert"
)

func TestStatusBadges_Draw_無効または空なら何も描かない(t *testing.T) {
	t.Parallel()

	t.Run("無効なら描かない", func(t *testing.T) {
		t.Parallel()
		badges := NewStatusBadges(nil)
		badges.enabled = false
		cv := &fakeCanvas{}
		badges.Draw(cv, StatusBadgesData{Badges: []StatusBadge{{Text: "毒"}}})
		assert.Empty(t, cv.texts)
	})

	t.Run("バッジが空なら描かない", func(t *testing.T) {
		t.Parallel()
		badges := NewStatusBadges(nil)
		cv := &fakeCanvas{}
		badges.Draw(cv, StatusBadgesData{Badges: nil})
		assert.Empty(t, cv.texts)
	})
}

func TestStatusBadges_Draw_バッジを下から積み上げて描く(t *testing.T) {
	t.Parallel()
	badges := NewStatusBadges(nil)
	cv := &fakeCanvas{}

	data := StatusBadgesData{
		Badges: []StatusBadge{
			{Text: "毒", Color: color.RGBA{R: 1, G: 0, B: 0, A: 255}},
			{Text: "麻痺", Color: color.RGBA{R: 0, G: 1, B: 0, A: 255}},
		},
		MessageAreaHeight: 40,
		ScreenDimensions:  ScreenDimensions{Width: 1024, Height: 768},
	}
	badges.Draw(cv, data)

	// 後ろのバッジから下から積み上げるので、後ろの「麻痺」がベースラインに近い下側に、
	// 先頭の「毒」がその上に来る
	poison := findText(t, cv.texts, "毒")
	paralysis := findText(t, cv.texts, "麻痺")
	assert.Greater(t, paralysis.pos.Y, poison.pos.Y, "後ろのバッジほど下に来る")
}

func TestStatusBadges_Draw_5個を超えると残数を表示する(t *testing.T) {
	t.Parallel()
	badges := NewStatusBadges(nil)
	cv := &fakeCanvas{}

	data := StatusBadgesData{
		Badges: []StatusBadge{
			{Text: "1"}, {Text: "2"}, {Text: "3"}, {Text: "4"}, {Text: "5"}, {Text: "6"}, {Text: "7"},
		},
		MessageAreaHeight: 40,
		ScreenDimensions:  ScreenDimensions{Width: 1024, Height: 768},
	}
	badges.Draw(cv, data)

	findText(t, cv.texts, "+2") // 上限を超えたぶんは+N表示になる。見つからなければ内部でFailする
	assert.False(t, hasText(cv.texts, "6"), "上限を超えたバッジ自体は描かない")
	assert.False(t, hasText(cv.texts, "7"), "上限を超えたバッジ自体は描かない")
}

// hasText は指定した文字列を描いたかどうかを返す
func hasText(texts []textCall, s string) bool {
	for _, tc := range texts {
		if tc.str == s {
			return true
		}
	}
	return false
}

func TestBadgeChrome_枠と塗りと上下のラインを描く(t *testing.T) {
	t.Parallel()
	cv := &fakeCanvas{}
	r := image.Rect(0, 0, 40, 20)

	badgeChrome(cv, r, theme.HUDBadgeBg)

	assert.Len(t, cv.strokeRects, 1, "外枠を描く")
	assert.Equal(t, r, cv.strokeRects[0])
	assert.Len(t, cv.fillRects, 3, "背景と上辺ハイライト・下辺シャドウの3つを塗る")
}
