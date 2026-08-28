package messagewindow

import (
	"image"
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/internal/ui"
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
		lw, th := measure("Enter", res.Text.BodyFace)
		cw := choicePadL*2 + lw
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
	_, lineH := measure("Ag", face)
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
		_, textH := measure(it.Label, face)
		off := (choiceRowH - textH) / 2

		// 選択中は行いっぱいに金色の選択バーを敷く
		if focused {
			hl := ui.NewNineSlice(res.SelectionBar.Image, res.SelectionBar.BX, res.SelectionBar.BY)
			hl.Layout(image.Rect(rect.Min.X, yy, rect.Max.X, yy+choiceRowH))
			children = append(children, hl)
		}

		col := theme.TextSecondary
		if focused {
			col = theme.TextSelected
		}
		// ラベルは左寄せ。追加ラベルはラベルの右に間隔を空けて連ねる
		x := rect.Min.X + choicePadL
		lbl := ui.NewText(it.Label, face, col)
		lbl.Layout(image.Rect(x, yy+off, rect.Max.X, yy+choiceRowH))
		children = append(children, lbl)
		lw, _ := measure(it.Label, face)
		x += lw
		for _, a := range it.AdditionalLabels {
			x += choiceLabelGap
			at := ui.NewText(a, face, col)
			at.Layout(image.Rect(x, yy+off, rect.Max.X, yy+choiceRowH))
			children = append(children, at)
			aw, _ := measure(a, face)
			x += aw
		}

		// 行の下に薄いグラデーションの区切り線を敷く。一覧の行と同じ意匠。
		// RowDivider は非乗算済みの値なので NRGBA として色を掛ける
		if res.GradientLine != nil {
			dv := ui.Row(nil).SetBottomLine(res.GradientLine, color.NRGBA(theme.RowDivider))
			dv.Layout(image.Rect(rect.Min.X, yy, rect.Max.X, yy+choiceRowH))
			children = append(children, dv)
		}
		yy += choiceRowH
	}

	root := ui.NewGroup(children...)
	root.Layout(rect)
	return root
}

// choiceBlockWidth は選択肢一覧の自然幅を返す。窓では中央寄せ、テストでは左寄せの塊の幅に使う。
// 各項目のラベルと追加ラベルの合計に左右パディングを足し、最大を採る
func choiceBlockWidth(config tabMenuConfig, state viewState, world w.World) int {
	res := world.Resources.UIResources
	face := res.Text.BodyFace
	items, _ := getVisibleItems(config, state)
	maxW := 0
	for _, it := range items {
		lw, _ := measure(it.Label, face)
		w := choicePadL*2 + lw
		for _, a := range it.AdditionalLabels {
			aw, _ := measure(a, face)
			w += choiceLabelGap + aw
		}
		if w > maxW {
			maxW = w
		}
	}
	if maxW < choiceMinWidth {
		maxW = choiceMinWidth
	}
	return maxW
}
