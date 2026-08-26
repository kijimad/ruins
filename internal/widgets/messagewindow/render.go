package messagewindow

import (
	"image"
	"image/color"
	"strings"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// windowFrameStyle はメッセージウィンドウの枠。背景と細い枠線を描く。
var windowFrameStyle = ui.BoxStyle{Fill: theme.WindowBackground, Border: theme.WindowBorder, BorderWidth: 2}

// titleBarStyle は話者名を載せるタイトルバーの帯。本体より暗くしてヘッダと分かる。
var titleBarStyle = ui.BoxStyle{Fill: color.RGBA{R: 15, G: 15, B: 20, A: 255}}

// choiceHighlightStyle は選択中の選択肢やプロンプトの背景強調。
var choiceHighlightStyle = ui.BoxStyle{Fill: theme.ChoiceSelectedBg}

const (
	titleBarHeight = 40 // タイトルバーの高さ
	choiceRowH     = 34 // 選択肢1行の高さ
	pageRowH       = 24 // ページ表示の高さ
)

// measure はフェイスでの文字列の幅と高さを返す。水平フローの送り幅と縦中央寄せに使う
func measure(s string, face text.Face) (int, int) {
	wpx, hpx := text.Measure(s, face, 0)
	return int(wpx), int(hpx)
}

// buildTree は現在の状態からウィンドウ全体を internal/ui のツリーへ組み、画面上に配置して返す。
// 枠・タイトルバー・メッセージ・選択肢またはEnterプロンプトを絶対座標で重ねる
func (win *Window) buildTree() ui.Widget {
	res := win.world.Resources.UIResources
	sd := win.world.Resources.ScreenDimensions

	size := win.calculateWindowSize()
	x, y := win.calculateWindowPosition(size)
	rect := image.Rect(x, y, x+size.Width, y+size.Height)

	var children []ui.Widget

	frame := ui.Panel(windowFrameStyle, size.Height)
	frame.Layout(rect)
	children = append(children, frame)

	// タイトルバー
	contentTop := rect.Min.Y
	if win.content.SpeakerName != "" {
		bar := ui.Panel(titleBarStyle, titleBarHeight)
		bar.Layout(image.Rect(rect.Min.X+2, rect.Min.Y+2, rect.Max.X-2, rect.Min.Y+titleBarHeight))
		children = append(children, bar)
		name := ui.NewText(win.content.SpeakerName, res.Text.SmallFace, theme.TextPrimary)
		name.Align = ui.AlignCenter
		name.Layout(image.Rect(rect.Min.X, rect.Min.Y+12, rect.Max.X, rect.Min.Y+titleBarHeight))
		children = append(children, name)
		contentTop = rect.Min.Y + titleBarHeight
	}

	padL := rect.Min.X + theme.Space6
	padR := rect.Max.X - theme.Space6

	// メッセージ本文
	if win.hasMessage() {
		children = append(children, win.segmentedLineWidgets(padL, padR, contentTop+theme.Space6, res.Text.BodyFace)...)
	}

	// 選択肢またはEnterプロンプトを下端寄りに置く
	if win.hasChoices {
		items, _ := getVisibleItems(win.choiceConfig, win.choiceState)
		blockH := len(items)*choiceRowH + theme.Space2
		if pageIndicatorText(win.choiceConfig, win.choiceState) != "" {
			blockH += pageRowH
		}
		blockTop := rect.Max.Y - theme.Space5 - blockH
		choiceRect := image.Rect(rect.Min.X, blockTop, rect.Max.X, rect.Max.Y-theme.Space5)
		children = append(children, renderChoiceList(win.choiceConfig, win.choiceState, win.world, choiceRect))
	} else {
		promptY := rect.Max.Y - theme.Space5 - choiceRowH
		cx := (rect.Min.X + rect.Max.X) / 2
		children = append(children, choiceRowWidgets(item{Label: "Enter"}, true, cx, promptY, padL, padR, res.Text.BodyFace)...)
	}

	root := ui.NewGroup(children...)
	root.Layout(image.Rect(0, 0, sd.Width, sd.Height))
	return root
}

// segmentedLineWidgets は TextSegmentLines を色付きの1行ずつへ組む。
// 各行は左端から測定幅ぶん右へセグメントを連ね、空行は行高ぶん送る。背景色付きは帯を敷く
func (win *Window) segmentedLineWidgets(padL, padR, top int, face text.Face) []ui.Widget {
	lineH := win.config.textStyle.LineHeight
	var out []ui.Widget
	yy := top
	for _, line := range win.content.TextSegmentLines {
		if lineIsBlank(line) {
			yy += lineH
			continue
		}
		xx := padL
		for _, seg := range line {
			if seg.Text == "" {
				continue
			}
			wpx, _ := measure(seg.Text, face)
			if seg.BackgroundColor != nil {
				bg := ui.Panel(ui.BoxStyle{Fill: *seg.BackgroundColor}, lineH)
				bg.Layout(image.Rect(xx, yy, xx+wpx, yy+lineH))
				out = append(out, bg)
			}
			var col color.Color = win.config.textStyle.Color
			if seg.Color != nil {
				col = *seg.Color
			}
			t := ui.NewText(seg.Text, face, col)
			t.Layout(image.Rect(xx, yy, padR, yy+lineH))
			out = append(out, t)
			xx += wpx
		}
		yy += lineH
	}
	return out
}

// lineIsBlank は行の全セグメントが空白文字だけかを返す
func lineIsBlank(line []messagedata.TextSegment) bool {
	for _, seg := range line {
		for _, r := range seg.Text {
			if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
				return false
			}
		}
	}
	return true
}

// renderChoiceList は選択肢一覧を rect 内へ組む。上端にページ表示、以降を1行ずつ下へ並べる。
// 各行はウィンドウ中央に文字を寄せ、選択中は背景を強調する。窓とテスト双方から使う
func renderChoiceList(config tabMenuConfig, state viewState, world w.World, rect image.Rectangle) ui.Widget {
	res := world.Resources.UIResources
	face := res.Text.BodyFace
	padL := rect.Min.X + theme.Space4
	padR := rect.Max.X - theme.Space4
	cx := (rect.Min.X + rect.Max.X) / 2

	var children []ui.Widget
	yy := rect.Min.Y
	if pageText := pageIndicatorText(config, state); pageText != "" {
		pi := ui.NewText(pageText, res.Text.SmallFace, theme.TextPrimary)
		pi.Align = ui.AlignCenter
		pi.Layout(image.Rect(rect.Min.X, yy, rect.Max.X, yy+pageRowH))
		children = append(children, pi)
		yy += pageRowH
	}

	items, indices := getVisibleItems(config, state)
	for i, it := range items {
		focused := indices[i] == state.ItemIndex
		children = append(children, choiceRowWidgets(it, focused, cx, yy, padL, padR, face)...)
		yy += choiceRowH
	}

	root := ui.NewGroup(children...)
	root.Layout(rect)
	return root
}

// choiceRowWidgets は選択肢1行分のウィジェットを返す。ラベルは中央寄せ、追加ラベルは右寄せ。
// 選択中は文字を明るくし、中央に背景帯を敷く
func choiceRowWidgets(it item, focused bool, cx, y, padL, padR int, face text.Face) []ui.Widget {
	var out []ui.Widget
	labelW, textH := measure(it.Label, face)
	off := (choiceRowH - textH) / 2

	textColor := theme.TextSecondary
	if focused {
		textColor = theme.TextSelected
		hw := labelW + 2*theme.Space5
		hx := cx - hw/2
		hl := ui.Panel(choiceHighlightStyle, choiceRowH)
		hl.Layout(image.Rect(hx, y, hx+hw, y+choiceRowH))
		out = append(out, hl)
	}

	if len(it.AdditionalLabels) > 0 {
		add := ui.NewText(strings.Join(it.AdditionalLabels, "  "), face, theme.TextSecondary)
		add.Align = ui.AlignRight
		add.Layout(image.Rect(padL, y+off, padR, y+choiceRowH))
		out = append(out, add)
	}

	lbl := ui.NewText(it.Label, face, textColor)
	lbl.Align = ui.AlignCenter
	lbl.Layout(image.Rect(padL, y+off, padR, y+choiceRowH))
	out = append(out, lbl)
	return out
}
