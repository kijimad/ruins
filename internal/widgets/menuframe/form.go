package menuframe

import (
	"image"
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/internal/uicore"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// 名前入力のような1項目の入力画面の寸法。画面はこれらを知らない
const (
	formWidth  = 320          // 入力欄と見出しの幅
	formTitleH = 36           // 見出し行の高さ
	formInputH = 44           // 入力欄の高さ
	formRowGap = theme.Space6 // 行と行の間隔
)

// FormScreen は1項目の入力画面を組む。見出し・入力欄・エラー・操作ヒントを画面中央の
// 縦並びに置く。errText が空ならエラー行は高さを取らない。
// 幅・行高・間隔は部品が持ち、画面は文言と入力欄の中身だけを渡す。
func FormScreen(world w.World, res resources.UIResources, title string, body uicore.Widget, errText, hint string) uicore.Widget {
	sd := world.Resources.ScreenDimensions

	items := []uicore.FlexItem{
		{W: newCenteredText(title, res.Text.TitleFontFace, theme.TextPrimary), Height: formTitleH},
		{Height: formRowGap},
		{W: InputBox(res, body), Height: formInputH},
		{Height: formRowGap},
	}
	if errText != "" {
		items = append(items,
			uicore.FlexItem{W: newCenteredText(errText, res.Text.SmallFace, theme.StatusDanger), Height: noteRowH},
			uicore.FlexItem{Height: formRowGap},
		)
	}
	items = append(items, uicore.FlexItem{W: newCenteredText(hint, res.Text.SmallFace, theme.TextAccent), Height: noteRowH})

	// 縦並びの総高を出し、画面の中央へ置く
	total := 0
	for _, it := range items {
		total += it.Height
	}
	rect := image.Rect(
		(sd.Width-formWidth)/2,
		(sd.Height-total)/2,
		(sd.Width-formWidth)/2+formWidth,
		(sd.Height-total)/2+total,
	)
	uicore.FlexColumn(rect, items)

	widgets := make([]uicore.Widget, 0, len(items))
	for _, it := range items {
		if it.W != nil {
			widgets = append(widgets, it.W)
		}
	}
	root := uicore.NewGroup(widgets...)
	root.Layout(image.Rect(0, 0, sd.Width, sd.Height))
	return root
}

// newCenteredText は中央寄せの1行を作る。入力画面の各行はすべて中央へそろえる
func newCenteredText(s string, face text.Face, c color.Color) *uicore.Text {
	t := uicore.NewText(s, face, c)
	t.Align = uicore.AlignCenter
	return t
}
