package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// newMenuListTable はコマンドメニュー系の一覧用テーブルを作る。行間を少し空け、項目が少ない
// 簡易メニューが詰まって見えないようにする。データ一覧系の密なテーブルとは行間だけを変える
func newMenuListTable(columnWidths []int, res resources.UIResources) *widget.Container {
	return styled.NewTableContainer(columnWidths, res,
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(theme.Space2),
		)),
	)
}

// menuRowWidth は全幅の一覧の行の総幅。全幅メニューで揃えて、画面ごとにエントリ幅がぶれないようにする。
// 列の内訳は画面ごとに変えてよいが、全幅の一覧では合計をこの値にする。
// 分割レイアウト内の小さい一覧、能力タブや職業一覧、は独自の幅にするのでこの限りでない
const menuRowWidth = 340

// itemIconColumnWidth はアイテム一覧のアイコン列の幅。テーブル行高と同じ正方にしてアイコンを収める。
// 行高 styled.tableRowHeight は 20 で、アイコンもその大きさへ縮小して描く
const itemIconColumnWidth = 20

// menuRow は一覧の1行。Cells は各列のセルで、アイコンと文字列が混ざってよい。
// Header が真なら見出し行でカーソルが止まらない
type menuRow struct {
	Cells  []styled.Cell
	Header bool
}

// menuListOpts は一覧描画の方針。Spaced はコマンドメニュー向けに行間を空けるか、
// AlwaysIndicator は1ページでもページ表示行を確保しタブ切替で開始位置を揃えるか、
// HeaderRow は表の先頭に置く列見出し、選択やページ送りの対象には含めない、
// EmptyText は行が無いときに表の下へ出す説明
type menuListOpts struct {
	Spaced          bool
	AlwaysIndicator bool
	HeaderRow       []string
	EmptyText       string
	// ItemsPerPage は1ページの行数の上書き。0 ならタブ帯つきモーダルの実測容量。
	// 見出しなど追加のチェームを持つ画面は menuframe.ListCapacity で自分の容量を求めて渡す
	ItemsPerPage int
}

// renderMenuList は一覧を共通の作法で組む唯一の入口。ページ送り・ページ表示・空行埋め・
// 行間をここに集約し、各メニューは行データ menuRow と列幅を渡すだけにする。これにより
// ページ送り忘れ・行間ずれを構造的に防ぐ。
// 列幅と行の値は呼び出し側が対で用意する。全幅の一覧では列幅の合計を menuRowWidth に揃える
func renderMenuList(itemIndex int, rows []menuRow, colWidths []int, aligns []styled.TextAlign, opts menuListOpts, res resources.UIResources) *widget.Container {
	// 列幅とセル数の対応を検査する。ずれると列が既定幅へ無言で落ちて崩れるため、内部の呼び出し
	// 不整合として早期に panic させる。アイコン列も普通の1列なのでセル数は常に列幅数と一致する
	if opts.HeaderRow != nil && len(opts.HeaderRow) != len(colWidths) {
		panic(fmt.Sprintf("renderMenuList: HeaderRow column count %d does not match column widths %d", len(opts.HeaderRow), len(colWidths)))
	}
	for i, r := range rows {
		if len(r.Cells) != len(colWidths) {
			panic(fmt.Sprintf("renderMenuList: row %d column count %d does not match column widths %d", i, len(r.Cells), len(colWidths)))
		}
	}

	perPage := opts.ItemsPerPage
	if perPage == 0 {
		perPage = menuframe.ListCapacity(res, false, true)
	}
	container := styled.NewVerticalContainer()
	pg := pagination.New(itemIndex, len(rows), perPage)
	if opts.AlwaysIndicator || pg.IsEnabled() {
		container.AddChild(newPageIndicator(pg, res))
	}

	table := styled.NewTableContainer(colWidths, res)
	if opts.Spaced {
		table = newMenuListTable(colWidths, res)
	}
	if opts.HeaderRow != nil {
		styled.NewTableHeaderRow(table, colWidths, opts.HeaderRow, res)
	}
	visible := pagination.VisibleEntries(rows, pg)
	for _, entry := range visible {
		if entry.Item.Header {
			// 見出し行はアイコンを持たない前提で、セルの文字列を並べて見出し専用行に渡す
			styled.NewTableHeaderRow(table, colWidths, cellTexts(entry.Item.Cells), res)
			continue
		}
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, colWidths, entry.Item.Cells, aligns, &isSelected, res)
	}
	// 複数ページの画面は各ページを1ページ件数ぶんの空行で埋め、ページを繰っても高さを一定にする
	if len(rows) > perPage {
		blank := make([]string, len(colWidths))
		for i := range blank {
			blank[i] = " "
		}
		blankCells := styled.TextCells(blank...)
		for i := len(visible); i < perPage; i++ {
			notSelected := false
			styled.NewTableRow(table, colWidths, blankCells, aligns, &notSelected, res)
		}
	}
	container.AddChild(table)
	// 行が無いときの空表示を一覧側で持つ。各メニューが同じ後処理を書かずに済む
	if len(rows) == 0 && opts.EmptyText != "" {
		container.AddChild(styled.NewDescriptionText(opts.EmptyText, res))
	}
	return container
}

// cellTexts は見出し行のセルから文字列を取り出す。見出しはアイコンを持たない前提で Text を並べる
func cellTexts(cells []styled.Cell) []string {
	texts := make([]string, len(cells))
	for i, c := range cells {
		texts[i] = c.Text
	}
	return texts
}

// itemRowData はアイテム一覧の1行の元データ。名前×個数で並べる複数のアイテムメニューで共通に使う。
// 表示の名前・個数・アイコンは itemMenuRow が entity から解決するので、Name・Count は元データの
// 保持にとどまる。Desc は詳細用途で、収納メニューなど使わない画面ではゼロ値のままにする
type itemRowData struct {
	Entity ecs.Entity
	Name   string
	Weight string
	Count  int
	Desc   string
}

// menuColumn は一覧の1列の幅と揃え。アイテムメニュー固有の trailing 列の指定に使う
type menuColumn struct {
	Width int
	Align styled.TextAlign
}

// itemMenuColumns は アイコン列と名前列の共通の先頭2列に、メニュー固有の trailing 列を足した
// 列幅と揃えを返す。アイコン列は行高と同じ正方、名前列は nameWidth で左揃え。
// これで対象メニュー間でアイコンと名前の見た目が揃う
func itemMenuColumns(nameWidth int, trailing ...menuColumn) ([]int, []styled.TextAlign) {
	colWidths := make([]int, 0, 2+len(trailing))
	aligns := make([]styled.TextAlign, 0, 2+len(trailing))
	colWidths = append(colWidths, itemIconColumnWidth, nameWidth)
	aligns = append(aligns, styled.AlignLeft, styled.AlignLeft)
	for _, c := range trailing {
		colWidths = append(colWidths, c.Width)
		aligns = append(aligns, c.Align)
	}
	return colWidths, aligns
}

// itemMenuRow は アイテム entity から共通の先頭部 [アイコンセル + 名前×個数セル] を組み、
// メニュー固有の trailing 文字列セルを後続に付けた menuRow を返す。
// 個数は呼び出し側が GroupStacks で確定済みの値を渡す。ここで数え直すと一覧の行ごとに
// 全走査が走るため、束ねた結果を持ち回って表示する
func itemMenuRow(world w.World, e ecs.Entity, count int, trailing ...string) menuRow {
	name := query.GetEntityName(e, world)
	icon, _ := resources.SpriteImage(world.Resources.SpriteSheets, world.Components.SpriteRender.Get(e))
	label := query.FormatNameCount(name, count)
	// 腐敗食は新鮮以外のとき鮮度を名前に添え、状態を見て食べるか捨てるか判断できるようにする
	if marker := query.FreshnessMarker(world, e); marker != "" {
		label += " (" + marker + ")"
	}
	cells := append([]styled.Cell{styled.IconCell(icon), styled.TextCell(label)}, styled.TextCells(trailing...)...)
	return menuRow{Cells: cells}
}

// newPageIndicator はページ位置を示す行を組み立てる。1ページに収まり表示が空になるときも
// 高さを一定に保つため半角空白を置く。ページングを持つ一覧で共通に使う
func newPageIndicator(pg pagination.Pagination, res resources.UIResources) *widget.Container {
	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " "
	}
	return styled.NewPageIndicator(pageText, res)
}

// ================
