package styled

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
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

// tableRowHeight はテーブル1行の高さ。本文は BodyFace、ヘッダは SmallFace。行を詰めて1画面に多く収める
const tableRowHeight = 20

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
	row := newRowContainer(columnWidths, image.NewNineSliceColor(theme.Transparent))
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
		addSelectableRow(container, columnWidths, nil, false, values, aligns, *isSelected, res)
		return
	}
	addDataRowColored(container, columnWidths, nil, false, values, aligns, theme.TextPrimary, res)
}

// NewTableRowWithIcon は先頭にアイコン列を持つテーブル行を作成する。
// icon が nil のときは透明なセルを置き、列数を columnWidths と揃える。
// columnWidths[0] がアイコン列で、values は後続の列に対応する
func NewTableRowWithIcon(container *widget.Container, columnWidths []int, icon *ebiten.Image, values []string, aligns []TextAlign, isSelected *bool, res resources.UIResources) {
	if isSelected != nil {
		addSelectableRow(container, columnWidths, icon, true, values, aligns, *isSelected, res)
		return
	}
	addDataRowColored(container, columnWidths, icon, true, values, aligns, theme.TextPrimary, res)
}

// NewSpriteCell は img を一辺 size のアイコン widget にする。原寸が size より大きければ縮小する。
// img が nil のときは透明な size×size のセルを返し、アイコンの無い行の桁を合わせる
func NewSpriteCell(img *ebiten.Image, size int) *widget.Graphic {
	scaled := ebiten.NewImage(size, size)
	if img != nil {
		bounds := img.Bounds()
		iw, ih := bounds.Dx(), bounds.Dy()
		if iw > 0 && ih > 0 {
			longest := max(ih, iw)
			scale := 1.0
			if longest > size {
				scale = float64(size) / float64(longest)
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scale, scale)
			// 縮小後のアイコンをセル中央へ寄せる
			op.GeoM.Translate((float64(size)-float64(iw)*scale)/2, (float64(size)-float64(ih)*scale)/2)
			scaled.DrawImage(img, op)
		}
	}
	return widget.NewGraphic(
		widget.GraphicOpts.Image(scaled),
		widget.GraphicOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.GridLayoutData{}),
			widget.WidgetOpts.MinSize(size, size),
		),
	)
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
	// 幅0の列があればそこを伸縮させて親コンテナの幅を埋める。無ければ最後の列を伸縮させる。
	// 幅0は「ここを伸ばす」印で、名前を伸ばして右側の数値列をまとめたい表で使う
	stretchIdx := columns - 1
	for i, cw := range columnWidths {
		if cw == 0 {
			stretchIdx = i
			break
		}
	}
	if columns > 0 {
		stretch[stretchIdx] = true
	}

	return widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(bgImage),
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(columns),
				widget.GridLayoutOpts.Spacing(theme.Space1, 0),
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

func addSelectableRow(container *widget.Container, columnWidths []int, icon *ebiten.Image, hasIcon bool, values []string, aligns []TextAlign, isSelected bool, res resources.UIResources) {
	bgImage := image.NewNineSliceColor(theme.Transparent)
	textColor := theme.TextSecondary
	if isSelected {
		bgImage = res.Panel.SelectionBar
		textColor = theme.TextSelected
	}

	row := newRowContainer(columnWidths, bgImage)
	addRowCells(row, columnWidths, icon, hasIcon, values, aligns, textColor, res)
	container.AddChild(row)

	container.AddChild(NewGradientLine(res.GradientLine, color.RGBA{255, 255, 255, 80}, 1))
}

// NewTableRowColored はデータ行を指定色の文字で描く。詳細モーダルで条件可否を色分けする用途に使う
func NewTableRowColored(container *widget.Container, columnWidths []int, values []string, aligns []TextAlign, textColor color.RGBA, res resources.UIResources) {
	addDataRowColored(container, columnWidths, nil, false, values, aligns, textColor, res)
}

func addDataRowColored(container *widget.Container, columnWidths []int, icon *ebiten.Image, hasIcon bool, values []string, aligns []TextAlign, textColor color.RGBA, res resources.UIResources) {
	row := newRowContainer(columnWidths, image.NewNineSliceColor(theme.Transparent))
	addRowCells(row, columnWidths, icon, hasIcon, values, aligns, textColor, res)
	container.AddChild(row)
}

// addRowCells は行コンテナへセルを並べる。hasIcon なら先頭にアイコンセルを1つ置き、
// values を後続の列に対応させる。columnWidths とアイコンぶんのずれをここで吸収する
func addRowCells(row *widget.Container, columnWidths []int, icon *ebiten.Image, hasIcon bool, values []string, aligns []TextAlign, textColor color.RGBA, res resources.UIResources) {
	col := 0
	if hasIcon {
		row.AddChild(NewSpriteCell(icon, tableRowHeight))
		col = 1
	}

	for i, value := range values {
		ci := col + i
		width := 80
		if ci < len(columnWidths) {
			width = columnWidths[ci]
		}

		textPos := widget.TextPositionStart
		gridData := widget.GridLayoutData{}
		if aligns != nil && ci < len(aligns) && aligns[ci] == AlignRight {
			textPos = widget.TextPositionEnd
			gridData.HorizontalPosition = widget.GridLayoutPositionEnd
		}

		textWidget := widget.NewText(
			widget.TextOpts.Text(value, &res.Text.BodyFace, textColor),
			widget.TextOpts.Position(textPos, widget.TextPositionCenter),
			widget.TextOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(gridData),
				widget.WidgetOpts.MinSize(width, tableRowHeight),
			),
		)
		row.AddChild(textWidget)
	}
}
