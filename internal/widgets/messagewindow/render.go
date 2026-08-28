package messagewindow

import (
	"image"
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

const (
	titleBarHeight = 25           // タイトルバーの高さ
	choiceRowH     = 26           // 選択肢1行の高さ。本文 + 上下パディング + 区切り線
	choiceTopPad   = 0            // 選択肢一覧の上パディング
	choicePadL     = theme.Space3 // 選択肢の左パディング
	choiceLabelGap = theme.Space2 // ラベルと追加ラベルの間隔
	pageRowH       = 24           // ページ表示の高さ
	choiceMinWidth = 120          // 選択肢の塊の最小幅
	// messageAreaHeight は本文に確保する高さ。行数で伸縮させず、窓の大きさを一定に保つ
	messageAreaHeight = 150
)

// chromeHeight は選択肢の塊を除いた窓の取り分を返す。タイトルバー・本文欄・上下の余白の合計で、
// 選択肢が何件でも変わらない。窓の高さの算出と、1ページに入る件数の逆算が同じ値を使う。
func (win *Window) chromeHeight() int {
	h := theme.Space6 + theme.Space5 // 本文上の余白と、選択肢の塊の下の余白
	if win.content.SpeakerName != "" {
		h += titleBarHeight
	}
	if win.hasMessage() {
		h += messageAreaHeight + theme.Space5 // 本文欄と、その下の間隔
	}
	return h
}

// requiredHeight は内容を収めるのに要る窓の高さを返す。実際に描くときと同じ寸法で積み上げるので、
// 計算した高さと描いた中身がずれない。窓の高さはこれと設定の最小高の大きいほうになる。
func (win *Window) requiredHeight() int {
	// ページ分割が決まっていれば実際に描く件数を、決まる前は全件を見込む。
	// 選択肢が無くても Enter プロンプトが1行ぶん要る
	rows := len(win.content.Choices)
	if items, _ := getVisibleItems(win.choiceConfig, win.choiceState); len(items) > 0 {
		rows = len(items)
	}
	h := win.chromeHeight() + choiceTopPad + max(rows, 1)*choiceRowH
	if win.hasChoices && pageIndicatorText(win.choiceConfig, win.choiceState) != "" {
		h += pageRowH
	}
	return h
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

	frame := ui.NewNineSlice(res.WindowBG.Image, res.WindowBG.BX, res.WindowBG.BY)
	frame.Layout(rect)
	children = append(children, frame)

	// タイトルバー
	contentTop := rect.Min.Y
	if win.content.SpeakerName != "" {
		bar := ui.NewNineSlice(res.TitleBar.Image, res.TitleBar.BX, res.TitleBar.BY)
		bar.Layout(image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+titleBarHeight))
		children = append(children, bar)
		name := ui.NewText(win.content.SpeakerName, res.Text.SmallFace, theme.TextPrimary)
		name.Align = ui.AlignCenter
		name.VCenter = true
		name.Layout(image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Min.Y+titleBarHeight))
		children = append(children, name)
		contentTop = rect.Min.Y + titleBarHeight
	}

	padL := rect.Min.X + theme.Space6
	padR := rect.Max.X - theme.Space6

	// メッセージ本文
	if win.hasMessage() {
		children = append(children, win.segmentedLineWidgets(padL, padR, contentTop+theme.Space6, res.Text.BodyFace)...)
	}

	// 選択肢またはEnterプロンプトを下端寄りに置く。一覧は自然幅の塊を中央へ寄せる
	cx := (rect.Min.X + rect.Max.X) / 2
	if win.hasChoices {
		items, _ := getVisibleItems(win.choiceConfig, win.choiceState)
		blockH := choiceTopPad + len(items)*choiceRowH
		if pageIndicatorText(win.choiceConfig, win.choiceState) != "" {
			blockH += pageRowH
		}
		blockTop := rect.Max.Y - theme.Space5 - blockH
		cw := choiceBlockWidth(win.choiceConfig, win.choiceState, win.world)
		choiceRect := image.Rect(cx-cw/2, blockTop, cx-cw/2+cw, rect.Max.Y-theme.Space5)
		children = append(children, renderChoiceList(win.choiceConfig, win.choiceState, win.world, choiceRect))
	} else {
		lw, th := ui.MeasureText("Enter", res.Text.BodyFace)
		// 選択肢の塊と同じ規則で幅を決め、文字幅ぴったりでボタンが痩せないようにする
		cw := ui.FitWidth([]int{lw}, choicePadL*2, choiceMinWidth)
		bx := cx - cw/2
		py := rect.Max.Y - theme.Space5 - choiceRowH
		hl := ui.NewNineSlice(res.SelectionBar.Image, res.SelectionBar.BX, res.SelectionBar.BY)
		hl.Layout(image.Rect(bx, py, bx+cw, py+choiceRowH))
		children = append(children, hl)
		prompt := ui.NewText("Enter", res.Text.BodyFace, theme.TextPrimary)
		prompt.Align = ui.AlignCenter
		prompt.Layout(image.Rect(bx, py+(choiceRowH-th)/2, bx+cw, py+choiceRowH))
		children = append(children, prompt)
	}

	root := ui.NewGroup(children...)
	root.Layout(image.Rect(0, 0, sd.Width, sd.Height))
	return root
}

// segmentedLineWidgets は TextSegmentLines を色付きの1行ずつへ組む。
// 各行は左端から測定幅ぶん右へセグメントを連ね、空行は行高ぶん送る。背景色付きは帯を敷く
func (win *Window) segmentedLineWidgets(padL, padR, top int, face text.Face) []ui.Widget {
	// 行送りはフェイスの自然な行高から取る。固定値だと本文の行間が字面に対して間延びする
	lineH := ui.LineHeight(face)
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
			wpx, _ := ui.MeasureText(seg.Text, face)
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

	var children []ui.Widget
	yy := rect.Min.Y
	if pageText := pageIndicatorText(config, state); pageText != "" {
		pi := ui.NewText(pageText, res.Text.SmallFace, theme.TextPrimary)
		pi.Align = ui.AlignCenter
		pi.Layout(image.Rect(rect.Min.X, yy, rect.Max.X, yy+pageRowH))
		children = append(children, pi)
		yy += pageRowH
	}
	yy += choiceTopPad

	items, indices := getVisibleItems(config, state)
	for i, it := range items {
		focused := indices[i] == state.ItemIndex
		_, textH := ui.MeasureText(it.Label, face)
		off := (choiceRowH - textH) / 2

		// 選択の強調と下端の区切り線は一覧の行と同じ意匠を使う
		chrome := menuframe.SelectionRow(res, focused)
		chrome.Layout(image.Rect(rect.Min.X, yy, rect.Max.X, yy+choiceRowH))
		children = append(children, chrome)

		col := theme.TextSecondary
		if focused {
			col = theme.TextSelected
		}
		// ラベルは左寄せ。追加ラベルはラベルの右に間隔を空けて連ねる
		x := rect.Min.X + choicePadL
		lbl := ui.NewText(it.Label, face, col)
		lbl.Layout(image.Rect(x, yy+off, rect.Max.X, yy+choiceRowH))
		children = append(children, lbl)
		lw, _ := ui.MeasureText(it.Label, face)
		x += lw
		for _, a := range it.AdditionalLabels {
			x += choiceLabelGap
			at := ui.NewText(a, face, col)
			at.Layout(image.Rect(x, yy+off, rect.Max.X, yy+choiceRowH))
			children = append(children, at)
			aw, _ := ui.MeasureText(a, face)
			x += aw
		}

		yy += choiceRowH
	}

	root := ui.NewGroup(children...)
	root.Layout(rect)
	return root
}

// choiceLabelsWidth は1項目のラベルと追加ラベルを連ねた送り幅を返す。
// ラベルの右へ間隔を空けて追加ラベルを並べる、という描画のしかたと同じ足し方をする。
func choiceLabelsWidth(it item, face text.Face) int {
	w := ui.MeasureTextWidth(it.Label, face)
	for _, a := range it.AdditionalLabels {
		w += choiceLabelGap + ui.MeasureTextWidth(a, face)
	}
	return w
}

// choiceBlockWidth は選択肢一覧の自然幅を返す。窓では中央寄せ、テストでは左寄せの塊の幅に使う。
// 各項目のラベルと追加ラベルの合計に左右パディングを足し、最大を採る
func choiceBlockWidth(config tabMenuConfig, state viewState, world w.World) int {
	res := world.Resources.UIResources
	face := res.Text.BodyFace
	items, _ := getVisibleItems(config, state)
	contents := make([]int, len(items))
	for i, it := range items {
		contents[i] = choiceLabelsWidth(it, face)
	}
	return ui.FitWidth(contents, choicePadL*2, choiceMinWidth)
}
