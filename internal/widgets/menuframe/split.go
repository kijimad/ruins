package menuframe

import (
	"image"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// 見出しと左右2枠を持つ全画面の寸法。画面はこれらを知らない
const (
	splitTitleTop     = 24  // 見出しの上端
	splitTitleH       = 36  // 見出し行の高さ
	splitBodyTop      = 80  // 左右の枠の上端
	splitSideMargin   = 40  // 画面左右の余白
	splitLeftW        = 180 // 左枠の幅
	splitGutter       = 20  // 左枠と右枠の間隔
	splitFooterH      = 72  // 下に空ける高さ。説明とヒントの2行ぶん
	splitLineH        = 16  // 下部の1行の高さ
	splitLineGap      = 8   // 下部の行と行の間隔
	splitBottomMargin = 16  // 画面下端の余白
)

// SplitScreen は見出し・左の一覧・右の詳細枠・下の説明とヒントを持つ全画面を組む。
// 左右の枠は上端をそろえ、下は説明の手前で止まる。寸法と余白は部品が持ち、
// 画面は中身の widget と文言だけを渡す。
func SplitScreen(world w.World, res resources.UIResources, title string, left, right ui.Widget, description, hint string) ui.Widget {
	sd := world.Resources.ScreenDimensions

	titleText := newCenteredText(title, res.Text.TitleFontFace, theme.TextPrimary)
	titleText.Layout(image.Rect(0, splitTitleTop, sd.Width, splitTitleTop+splitTitleH))

	bodyBottom := sd.Height - splitFooterH
	leftX := splitSideMargin
	left.Layout(image.Rect(leftX, splitBodyTop, leftX+splitLeftW, bodyBottom))
	right.Layout(image.Rect(leftX+splitLeftW+splitGutter, splitBodyTop, sd.Width-splitSideMargin, bodyBottom))

	// 説明とヒントは下端から積み上げる
	descTop := sd.Height - splitLineH*2 - splitLineGap - splitBottomMargin
	descText := newCenteredText(description, res.Text.SmallFace, theme.TextAccent)
	descText.Layout(image.Rect(0, descTop, sd.Width, descTop+splitLineH))
	hintTop := descTop + splitLineH + splitLineGap
	hintText := newCenteredText(hint, res.Text.SmallFace, theme.TextAccent)
	hintText.Layout(image.Rect(0, hintTop, sd.Width, hintTop+splitLineH))

	root := ui.NewGroup(titleText, left, right, descText, hintText)
	root.Layout(image.Rect(0, 0, sd.Width, sd.Height))
	return root
}

// SplitList は SplitScreen の左枠に載せる一覧を組む。単一ページの一覧で、ページ表示は持たない。
func SplitList(itemIndex int, rows []Row, res resources.UIResources) ui.Widget {
	listRows, _ := RenderList(itemIndex, rows, styled.Cols(styled.Name()), ListOpts{}, res)
	return ui.VBox(theme.MenuPanelRowH, ui.Placeable(listRows)...)
}
