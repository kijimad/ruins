package states

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// tabScreen はモーダル画面の共通レイアウト入力。
// 各画面はこの入力を渡すだけで、見出し・タブ帯・コンテンツ・フッターの配置と
// モーダル枠、ログ回避、上詰めが標準化される。目視での位置合わせを不要にする。
// TabLabels・Footer・Header は空なら該当行を置かず、タブの無いメニューにも使える。
type tabScreen struct {
	// Header は上部中央の見出し。空なら見出し行を置かない
	Header string
	// TabLabels はタブ帯の見出し一覧。空ならタブ帯を置かない。TabIndex を強調表示する
	TabLabels []string
	TabIndex  int
	// Content は画面の中身。ページ表示行を先頭に含めると全画面で開始位置が揃う
	Content *widget.Container
	// Footer は下部のキー案内。空なら置かない。小さめの補助テキストで表示する
	Footer string
}

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

// newPanelScreenUI は中央に寄せた、内容サイズに縮む小さめパネルのメニュー画面を組む。
// タイトル・本体・フッターを縦に積む。メインメニューやセーブロードのような簡易コマンドメニューの見た目に揃える。
// 大きめモーダルの newTabScreenUI と違い、項目数が少ない画面がエントリ数相応の大きさに収まる
func newPanelScreenUI(res resources.UIResources, title string, content *widget.Container, footer string) *ebitenui.UI {
	panel := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.Image),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 0),
		),
	)
	if title != "" {
		panel.AddChild(styled.NewMenuText(title, res))
	}
	panel.AddChild(content)
	if footer != "" {
		panel.AddChild(styled.NewDescriptionText(footer, res))
	}
	// 下部にログ領域ぶんの余白を確保し、その上の領域で中央寄せする。データ一覧のモーダルと
	// 同じくログに被らないようにする
	logReserve := consts.GameHeight - menuscreen.LogTopY(consts.GameHeight) + theme.Space3
	root := widget.NewContainer(widget.ContainerOpts.Layout(
		widget.NewAnchorLayout(widget.AnchorLayoutOpts.Padding(&widget.Insets{Bottom: logReserve})),
	))
	root.AddChild(panel)
	return &ebitenui.UI{Container: root}
}

// newTabScreenUI はモーダル画面の標準 UI を組み立てる。
// 行構成は 見出し（任意）/ タブ帯（任意）/ コンテンツ / 伸縮スペーサー / フッター（任意）。
// コンテンツは上詰めされ、フッターは下端でログの手前に収まる。呼び出し側は
// 返り値へ詳細モーダル等のウィンドウを AddWindow できる。
func newTabScreenUI(res resources.UIResources, p tabScreen) *ebitenui.UI {
	children := make([]widget.PreferredSizeLocateableWidget, 0, 5)
	rowStretch := make([]bool, 0, 5)
	add := func(c widget.PreferredSizeLocateableWidget, stretch bool) {
		children = append(children, c)
		rowStretch = append(rowStretch, stretch)
	}

	if p.Header != "" {
		add(centerRow(styled.NewMenuText(p.Header, res)), false)
	}
	if len(p.TabLabels) > 0 {
		add(centerRow(styled.NewTabBar(p.TabLabels, p.TabIndex, res)), false)
	}
	add(p.Content, false)
	add(widget.NewContainer(), true) // 伸縮スペーサー。フッターを下端へ押す
	if p.Footer != "" {
		footer := styled.NewRowContainer()
		footer.AddChild(styled.NewDescriptionText(p.Footer, res))
		add(footer, false)
	}

	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.Image),
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Spacing(0, theme.Space2),
			widget.GridLayoutOpts.Stretch([]bool{true}, rowStretch),
			widget.GridLayoutOpts.Padding(&widget.Insets{Top: theme.Space3, Bottom: theme.Space3, Left: theme.Space3, Right: theme.Space3}),
		)),
	)
	for _, c := range children {
		root.AddChild(c)
	}
	return &ebitenui.UI{Container: wrapModalRoot(root)}
}

// menuItemsPerPage は一覧1ページの表示件数。全メニュー共通。モーダルの高さに収まり
// ログ領域へはみ出さない上限にする。テーブル行20px + ページ表示 + タブ帯 + フッターが
// モーダル領域に収まる件数を基準にする
const menuItemsPerPage = 18

// menuRowWidth は全幅の一覧の行の総幅。全幅メニューで揃えて、画面ごとにエントリ幅がぶれないようにする。
// 列の内訳は画面ごとに変えてよいが、全幅の一覧では合計をこの値にする。
// 分割レイアウト内の小さい一覧、能力タブや職業一覧、は独自の幅にするのでこの限りでない
const menuRowWidth = 340

// menuRow は一覧の1行。Cells は各列の文字列、Header が真なら見出し行でカーソルが止まらない
type menuRow struct {
	Cells  []string
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
}

// renderMenuList は一覧を共通の作法で組む唯一の入口。ページ送り・ページ表示・空行埋め・
// 行間をここに集約し、各メニューは行データ menuRow と列幅を渡すだけにする。これにより
// ページ送り忘れ・行間ずれを構造的に防ぐ。
// 列幅と行の値は呼び出し側が対で用意する。全幅の一覧では列幅の合計を menuRowWidth に揃える
func renderMenuList(itemIndex int, rows []menuRow, colWidths []int, aligns []styled.TextAlign, opts menuListOpts, res resources.UIResources) *widget.Container {
	// 列幅と行の値の対応を検査する。ずれると列が既定幅へ無言で落ちて崩れるため、内部の呼び出し
	// 不整合として早期に panic させる
	if opts.HeaderRow != nil && len(opts.HeaderRow) != len(colWidths) {
		panic(fmt.Sprintf("renderMenuList: HeaderRow column count %d does not match column widths %d", len(opts.HeaderRow), len(colWidths)))
	}
	for i, r := range rows {
		if len(r.Cells) != len(colWidths) {
			panic(fmt.Sprintf("renderMenuList: row %d column count %d does not match column widths %d", i, len(r.Cells), len(colWidths)))
		}
	}

	container := styled.NewVerticalContainer()
	pg := pagination.New(itemIndex, len(rows), menuItemsPerPage)
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
			styled.NewTableHeaderRow(table, colWidths, entry.Item.Cells, res)
			continue
		}
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, colWidths, entry.Item.Cells, aligns, &isSelected, res)
	}
	// 複数ページの画面は各ページを1ページ件数ぶんの空行で埋め、ページを繰っても高さを一定にする
	if len(rows) > menuItemsPerPage {
		blank := make([]string, len(colWidths))
		for i := range blank {
			blank[i] = " "
		}
		for i := len(visible); i < menuItemsPerPage; i++ {
			notSelected := false
			styled.NewTableRow(table, colWidths, blank, aligns, &notSelected, res)
		}
	}
	container.AddChild(table)
	// 行が無いときの空表示を一覧側で持つ。各メニューが同じ後処理を書かずに済む
	if len(rows) == 0 && opts.EmptyText != "" {
		container.AddChild(styled.NewDescriptionText(opts.EmptyText, res))
	}
	return container
}

// nameWithCount は個数が2以上のとき名前に ×個数 を添える。1個や非スタックは名前だけを返す
func nameWithCount(name string, count int) string {
	if count > 1 {
		return fmt.Sprintf("%s %s%d", name, consts.IconTimes, count)
	}
	return name
}

// newCurrencyRow は所持金を表示する行を組み立てる。酒場で使う
func newCurrencyRow(currency int, res resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	container.AddChild(styled.NewMenuText(query.FormatCurrency(currency), res))
	return container
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

// menuNavHint はメニュー共通のキー操作案内を組み立てる。全メニューのフッターに常設し、
// どの画面でも同じキーで同じ操作ができることを示す。矢印や Enter/Esc は素の記号がフォントに
// 無く文字化けするため FontAwesome のアイコンを使う。hasTabs が true のときタブ切替を含め、
// extras に画面固有の案内を後ろへ足す
func menuNavHint(world w.World, hasTabs bool, extras ...string) string {
	parts := make([]string, 0, 4+len(extras))
	if hasTabs {
		parts = append(parts, consts.IconArrowLeft+consts.IconArrowRight+" "+query.T(world, "Tab"))
	}
	parts = append(parts, consts.IconArrowUp+consts.IconArrowDown+" "+query.T(world, "Select"))
	parts = append(parts, consts.IconKeyEnter+" "+query.T(world, "Confirm"))
	parts = append(parts, extras...)
	parts = append(parts, consts.IconKeyEsc+" "+query.T(world, "Back"))
	return strings.Join(parts, "   ")
}

// centerRow は子を水平中央に置くアンカーコンテナを返す。タブ帯や見出しの中央寄せに使う
func centerRow(child widget.PreferredSizeLocateableWidget) *widget.Container {
	row := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	child.GetWidget().LayoutData = widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter}
	row.AddChild(child)
	return row
}

// wrapModalRoot は root を画面より一回り小さい中央モーダルとして包む。
// 外周は背景を持たず透明にし、周囲に後ろのフィールドを覗かせる。動詞タブ画面と各メニューで共通に使う。
// 下端はゲームログの上端より上で止め、ログと重ならないようにする。
func wrapModalRoot(root *widget.Container) *widget.Container {
	bottom := consts.GameHeight - menuscreen.LogTopY(consts.GameHeight) + theme.Space3
	outer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true}),
			widget.GridLayoutOpts.Padding(&widget.Insets{Top: 48, Bottom: bottom, Left: 96, Right: 96}),
		)),
	)
	outer.AddChild(root)
	return outer
}

// ================
