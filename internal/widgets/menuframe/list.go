package menuframe

import (
	"image/color"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// Row は一覧の1行。Cells は各列のセルで、アイコンと文字列が混ざってよい。
// Header が真なら見出し行でカーソルが止まらない。
type Row struct {
	Cells  []styled.Cell
	Header bool
}

// ListOpts は一覧描画の方針。HeaderRow は表の先頭に置く列見出し、選択やページ送りの対象には
// 含めない、EmptyText は行が無いときに表の下へ出す説明。ページ表示はフッタ右端に出すので
// 一覧側では扱わない。行の高さと間隔は並べる先の TabScreen・PanelScreen が決める。
type ListOpts struct {
	HeaderRow []string
	EmptyText string
	// ItemsPerPage は1ページの行数の上書き。0 なら全行を1ページに収める。
	// 見出しやタブ帯を持つ画面は ListCapacity で自分の容量を求めて渡す。
	ItemsPerPage int
}

// RenderList は行データから見出し・選択強調・空行埋めを備えた行ウィジェット列と、
// フッタ右端へ出すページ表示文字列を返す。ページ表示は複数ページのときだけ非空になる。
// 呼び出し側は行を TabScreen/PanelScreen へ並べ、ページ表示をそのフッタへ渡す。
// 1ページの件数は呼び出し側が opts.ItemsPerPage に解決して渡す。0 なら全行を1ページに収める。
// itemIndex に負値を渡すとどの行も選択されず、カーソルを持たない表になる。
// 列幅は CSS Grid のトラックと同じ規則で解決する。Fit の列は全行の内容の実測から、
// Grow の列は余り幅から決まり、表全体で列が揃う。呼び出し側は幅の数値を書かない。
func RenderList(itemIndex int, rows []Row, cols []styled.Col, opts ListOpts, res resources.UIResources) ([]uicore.Drawable, string) {
	face := res.Text.BodyFace
	colWidths := resolveColWidths(cols, opts.HeaderRow, rows, face)
	aligns := styled.Aligns(cols)
	perPage := opts.ItemsPerPage
	if perPage <= 0 {
		perPage = max(len(rows), 1)
	}
	pg := pagination.New(itemIndex, len(rows), perPage)

	var items []uicore.Drawable
	if opts.HeaderRow != nil {
		items = append(items, headerRow(opts.HeaderRow, colWidths, face))
	}
	visible := pagination.VisibleEntries(rows, pg)
	for _, entry := range visible {
		if entry.Item.Header {
			items = append(items, headerRow(cellTexts(entry.Item.Cells), colWidths, face))
			continue
		}
		items = append(items, dataRow(entry.Item.Cells, colWidths, aligns, pg.IsSelectedInPage(entry.Index), face, res))
	}
	// 複数ページの画面は各ページを1ページ件数ぶんの空行で埋め、ページを繰っても高さを一定にする
	if len(rows) > perPage {
		for i := len(visible); i < perPage; i++ {
			items = append(items, blankRow(colWidths, face))
		}
	}
	// 行が無いときの空表示を一覧側で持つ。各メニューが同じ後処理を書かずに済む
	if len(rows) == 0 && opts.EmptyText != "" {
		items = append(items, uicore.NewText(opts.EmptyText, face, theme.TextSecondary))
	}
	return items, pg.GetPageText()
}

// resolveColWidths は列のトラック幅を解決する。Fit の列は見出しと全行の内容の実測の最大値に
// 列間の間隔を足した幅、Icon の列は正方の固定幅にする。Grow の列は 0 のままにし、
// 行の flex が余り幅を割り当てる。
func resolveColWidths(cols []styled.Col, headerRow []string, rows []Row, face text.Face) []int {
	widths := make([]int, len(cols))
	for i, c := range cols {
		switch c.Mode {
		case styled.ColIcon:
			widths[i] = theme.MenuIconW
		case styled.ColFit:
			// 見出しと全行のこの列の中身を並べ、いちばん広いものに列間の間隔を足す
			contents := make([]int, 0, len(rows)+1)
			if i < len(headerRow) {
				contents = append(contents, uicore.MeasureTextWidth(headerRow[i], face))
			}
			for _, r := range rows {
				if i >= len(r.Cells) {
					continue
				}
				if cell := r.Cells[i]; cell.Icon != nil {
					// アイコンセルは画像の実幅で測る。正方の一覧アイコンも
					// 横長のキーキャップも同じ規則で測れる
					contents = append(contents, cell.Icon.Bounds().Dx())
				} else {
					contents = append(contents, uicore.MeasureTextWidth(cell.Text, face))
				}
			}
			widths[i] = uicore.FitWidth(contents, theme.Space3, 0)
		case styled.ColGrow:
			// 0 のままにする。行の flex が余り幅を割り当てる
		}
	}
	return widths
}

// cellTexts は見出し行のセルから文字列を取り出す。見出しはアイコンを持たない前提で Text を並べる。
func cellTexts(cells []styled.Cell) []string {
	texts := make([]string, len(cells))
	for i, c := range cells {
		texts[i] = c.Text
	}
	return texts
}

// toAlign は styled のそろえを internal/uicore のそろえへ写す。
func toAlign(a styled.TextAlign) uicore.Align {
	if a == styled.AlignRight {
		return uicore.AlignRight
	}
	return uicore.AlignLeft
}

// headerRow は見出し行を組む。カーソルは止まらず、補助色で描く。
func headerRow(texts []string, colWidths []int, face text.Face) *uicore.Container {
	cells := make([]uicore.Widget, len(texts))
	for i, s := range texts {
		t := uicore.NewText(s, face, theme.TextSecondary)
		t.VCenter = true
		cells[i] = t
	}
	return uicore.Row(colWidths, cells...)
}

// dataRow はデータ行を組む。選択中なら金色の選択バーを敷き文字色を選択色にする。アイコンセルは画像で描く。
func dataRow(cells []styled.Cell, colWidths []int, aligns []styled.TextAlign, selected bool, face text.Face, res resources.UIResources) *uicore.Container {
	// 非選択は暗く、選択は明るくして、カーソル位置を際立たせる
	var textColor color.Color = theme.TextSecondary
	if selected {
		textColor = theme.TextSelected
	}
	cellWidgets := make([]uicore.Widget, len(cells))
	for i, c := range cells {
		if c.Icon != nil {
			cellWidgets[i] = uicore.NewGraphic(c.Icon)
			continue
		}
		t := uicore.NewText(c.Text, face, textColor)
		t.VCenter = true
		if i < len(aligns) {
			t.Align = toAlign(aligns[i])
		}
		cellWidgets[i] = t
	}
	return rowChrome(uicore.Row(colWidths, cellWidgets...), res, selected)
}

// SelectionRow は中身を持たない1行ぶんの意匠を返す。選択中の強調と下端の区切り線だけを持ち、
// 中身は呼び出し側が別に重ねる。行の内容を列で表せない一覧が、意匠だけを共有するのに使う。
func SelectionRow(res resources.UIResources, selected bool) uicore.Widget {
	return rowChrome(uicore.Row(nil), res, selected)
}

// rowChrome は一覧の1行に共通の意匠を着せる。選択中は金色のバーを敷き、下端に区切り線を引く。
// この2つが一覧の行の見た目を決めるので、定義をここ1箇所に置く。
func rowChrome(row *uicore.Container, res resources.UIResources, selected bool) *uicore.Container {
	if selected && res.SelectionBar != nil {
		row.SetBackgroundNineSlice(res.SelectionBar.Image, res.SelectionBar.BX, res.SelectionBar.BY)
	}
	// RowDivider は非乗算済みの値なので NRGBA として色を掛ける
	if res.GradientLine != nil {
		row.SetBottomLine(res.GradientLine, color.NRGBA(theme.RowDivider))
	}
	return row
}

// blankRow は高さを揃えるための空行を組む。高さは行高が確保するので文字は持たせず、
// フォント測定と描画を省く。
func blankRow(colWidths []int, face text.Face) *uicore.Container {
	cells := make([]uicore.Widget, len(colWidths))
	for i := range cells {
		cells[i] = uicore.NewText("", face, theme.TextPrimary)
	}
	return uicore.Row(colWidths, cells...)
}
