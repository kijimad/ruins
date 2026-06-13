package styled

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// TextAlign はテーブルセル内のテキスト揃え方向を表す
type TextAlign int

// テキスト揃え方向の定数
const (
	AlignLeft TextAlign = iota
	AlignRight
)

// NewTableContainer はテーブルのコンテナを作成する
// 各行がコンテナとなる縦並びレイアウトで、行単位の背景色設定が可能
func NewTableContainer(_ []int, _ resources.UIResources, opts ...widget.ContainerOpt) *widget.Container {
	defaultOpts := []widget.ContainerOpt{
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(0),
			),
		),
		// 親の RowLayout 内で横幅いっぱいに伸長する
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	}

	allOpts := make([]widget.ContainerOpt, 0, len(defaultOpts)+len(opts))
	allOpts = append(allOpts, defaultOpts...)
	allOpts = append(allOpts, opts...)

	return widget.NewContainer(allOpts...)
}

// NewTableHeaderRow はヘッダー行のセル群を作成してコンテナに追加する
func NewTableHeaderRow(container *widget.Container, columnWidths []int, headers []string, res resources.UIResources) {
	row := newRowContainer(columnWidths, image.NewNineSliceColor(color.NRGBA{}))
	for i, header := range headers {
		width := 80
		if i < len(columnWidths) {
			width = columnWidths[i]
		}

		cell := widget.NewText(
			widget.TextOpts.Text(header, &res.Text.SmallFace, theme.TextSecondary),
			widget.TextOpts.Position(widget.TextPositionStart, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.GridLayoutData{}),
				widget.WidgetOpts.MinSize(width, 20),
			),
		)
		row.AddChild(cell)
	}
	container.AddChild(row)
}

// NewTableRow はテーブル行を作成する
// isSelectedがnilの場合は通常行、非nilの場合は最初の列にカーソルを表示する選択可能行になる
// alignsがnilの場合は全て左揃えになる
func NewTableRow(container *widget.Container, columnWidths []int, values []string, aligns []TextAlign, isSelected *bool, res resources.UIResources) {
	if isSelected != nil {
		addSelectableRow(container, columnWidths, values, aligns, *isSelected, res)
		return
	}
	addDataRow(container, columnWidths, values, aligns, res)
}

// ================
// 内部関数
// ================

// newRowContainer は行コンテナを作成する。背景画像を指定でき、横幅は親に合わせて伸びる
func newRowContainer(columnWidths []int, bgImage *image.NineSlice) *widget.Container {
	columns := len(columnWidths)
	if columns == 0 {
		columns = 1
	}

	stretch := make([]bool, columns)
	// 最後の列を伸縮させて親コンテナの幅を埋める
	if columns > 0 {
		stretch[columns-1] = true
	}

	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(bgImage),
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(columns),
				widget.GridLayoutOpts.Spacing(theme.SpaceXS, 0),
				widget.GridLayoutOpts.Stretch(stretch, []bool{false}),
				widget.GridLayoutOpts.Padding(&widget.Insets{}),
			),
		),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
		),
	)
}

func addSelectableRow(container *widget.Container, columnWidths []int, values []string, aligns []TextAlign, isSelected bool, res resources.UIResources) {
	bgImage := image.NewNineSliceColor(color.NRGBA{})
	textColor := theme.TextSecondary
	if isSelected {
		bgImage = res.Panel.SelectionBar
		textColor = theme.TextSelected
	}

	row := newRowContainer(columnWidths, bgImage)

	for i := 0; i < len(values); i++ {
		width := 80
		if i < len(columnWidths) {
			width = columnWidths[i]
		}

		textPos := widget.TextPositionStart
		gridData := widget.GridLayoutData{}
		if aligns != nil && i < len(aligns) && aligns[i] == AlignRight {
			textPos = widget.TextPositionEnd
			gridData.HorizontalPosition = widget.GridLayoutPositionEnd
		}

		textWidget := widget.NewText(
			widget.TextOpts.Text(values[i], &res.Text.BodyFace, textColor),
			widget.TextOpts.Position(textPos, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(gridData),
				widget.WidgetOpts.MinSize(width, 24),
			),
		)
		row.AddChild(textWidget)
	}

	container.AddChild(row)

	// 白線は常にグラデーションの上に表示
	separator := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.SeparatorLine),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
			widget.WidgetOpts.MinSize(0, 1),
		),
	)
	container.AddChild(separator)
}

func addDataRow(container *widget.Container, columnWidths []int, values []string, aligns []TextAlign, res resources.UIResources) {
	row := newRowContainer(columnWidths, image.NewNineSliceColor(color.NRGBA{}))

	for i, value := range values {
		width := 80
		if i < len(columnWidths) {
			width = columnWidths[i]
		}

		textPos := widget.TextPositionStart
		gridData := widget.GridLayoutData{}
		if aligns != nil && i < len(aligns) && aligns[i] == AlignRight {
			textPos = widget.TextPositionEnd
			gridData.HorizontalPosition = widget.GridLayoutPositionEnd
		}

		textWidget := widget.NewText(
			widget.TextOpts.Text(value, &res.Text.BodyFace, theme.TextPrimary),
			widget.TextOpts.Position(textPos, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(gridData),
				widget.WidgetOpts.MinSize(width, 24),
			),
		)
		row.AddChild(textWidget)
	}

	container.AddChild(row)
}
