package styled

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
)

// TextAlign はテーブルセル内のテキスト揃え方向を表す
type TextAlign int

// テキスト揃え方向の定数
const (
	AlignLeft TextAlign = iota
	AlignRight
)

// NewTableContainer はテーブルのコンテナを作成する
// columnWidths で各列の幅を指定する
func NewTableContainer(columnWidths []int, _ *resources.UIResources, opts ...widget.ContainerOpt) *widget.Container {
	columns := len(columnWidths)
	if columns == 0 {
		columns = 1
	}

	stretch := make([]bool, columns)
	for i := range stretch {
		stretch[i] = false
	}

	defaultOpts := []widget.ContainerOpt{
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(columns),
				widget.GridLayoutOpts.Spacing(2, 2),
				widget.GridLayoutOpts.Stretch(stretch, []bool{false}),
			),
		),
	}

	allOpts := make([]widget.ContainerOpt, 0, len(defaultOpts)+len(opts))
	allOpts = append(allOpts, defaultOpts...)
	allOpts = append(allOpts, opts...)

	return widget.NewContainer(allOpts...)
}

// NewTableHeaderRow はヘッダー行のセル群を作成してコンテナに追加する
func NewTableHeaderRow(container *widget.Container, columnWidths []int, headers []string, res *resources.UIResources) {
	for i, header := range headers {
		width := 80
		if i < len(columnWidths) {
			width = columnWidths[i]
		}

		cell := widget.NewText(
			widget.TextOpts.Text(header, &res.Text.SmallFace, consts.ForegroundColor),
			widget.TextOpts.Position(widget.TextPositionStart, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.GridLayoutData{}),
				widget.WidgetOpts.MinSize(width, 20),
			),
		)
		container.AddChild(cell)
	}
}

// NewTableRow はテーブル行を作成する
// isSelectedがnilの場合は通常行、非nilの場合は最初の列にカーソルを表示する選択可能行になる
// alignsがnilの場合は全て左揃えになる
func NewTableRow(container *widget.Container, columnWidths []int, values []string, aligns []TextAlign, isSelected *bool, res *resources.UIResources) {
	if isSelected != nil {
		addSelectableRow(container, columnWidths, values, aligns, *isSelected, res)
		return
	}
	addDataRow(container, columnWidths, values, aligns, res)
}

// ================
// 内部関数
// ================

func addSelectableRow(container *widget.Container, columnWidths []int, values []string, aligns []TextAlign, isSelected bool, res *resources.UIResources) {
	cursorColor := color.RGBA{}
	if isSelected {
		cursorColor = consts.PrimaryColor
	}

	cursorText := widget.NewText(
		widget.TextOpts.Text(consts.IconCursor, &res.Text.BodyFace, cursorColor),
		widget.TextOpts.Position(widget.TextPositionStart, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{}),
			widget.WidgetOpts.MinSize(columnWidths[0], 24),
		),
	)
	container.AddChild(cursorText)

	for i := 1; i < len(values); i++ {
		width := 80
		if i < len(columnWidths) {
			width = columnWidths[i]
		}

		textPos := widget.TextPositionStart
		if aligns != nil && i < len(aligns) && aligns[i] == AlignRight {
			textPos = widget.TextPositionEnd
		}

		textWidget := widget.NewText(
			widget.TextOpts.Text(values[i], &res.Text.BodyFace, consts.TextColor),
			widget.TextOpts.Position(textPos, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.GridLayoutData{}),
				widget.WidgetOpts.MinSize(width, 24),
			),
		)
		container.AddChild(textWidget)
	}
}

func addDataRow(container *widget.Container, columnWidths []int, values []string, aligns []TextAlign, res *resources.UIResources) {
	for i, value := range values {
		width := 80
		if i < len(columnWidths) {
			width = columnWidths[i]
		}

		textPos := widget.TextPositionStart
		if aligns != nil && i < len(aligns) && aligns[i] == AlignRight {
			textPos = widget.TextPositionEnd
		}

		textWidget := widget.NewText(
			widget.TextOpts.Text(value, &res.Text.BodyFace, consts.TextColor),
			widget.TextOpts.Position(textPos, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.GridLayoutData{}),
				widget.WidgetOpts.MinSize(width, 24),
			),
		)
		container.AddChild(textWidget)
	}
}
