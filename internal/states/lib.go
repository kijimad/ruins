package states

import (
	"fmt"
	"image"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/hud"
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

// newTabScreenUI はモーダル画面の標準 UI を組み立てる。
// 行構成は 見出し（任意）/ タブ帯（任意）/ コンテンツ / 伸縮スペーサー / フッター（任意）。
// コンテンツは上詰めされ、フッターは下端でログの手前に収まる。呼び出し側は
// 返り値へ詳細モーダル等のウィンドウを AddWindow できる。
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
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
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
	logReserve := consts.GameHeight - gameLogTopY(consts.GameHeight) + theme.Space3
	root := widget.NewContainer(widget.ContainerOpts.Layout(
		widget.NewAnchorLayout(widget.AnchorLayoutOpts.Padding(&widget.Insets{Bottom: logReserve})),
	))
	root.AddChild(panel)
	return &ebitenui.UI{Container: root}
}

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
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
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

// menuRowWidth はメニュー一覧の行の総幅。全メニューで揃えて、画面ごとにエントリ幅がぶれないようにする。
// 各メニューは列の内訳を変えてもよいが、合計はこの値にする
const menuRowWidth = 340

// newThreeColContent は3列3行のメニュー本文を組み立てる。商店・合成・収納で共通に使う。
// 右上に所持金や重量、中段左に一覧、中段右に性能や参照リスト、左下に説明を置き、
// 残りのセルは空で埋めて位置を揃える。nil のセルは空コンテナにする
func newThreeColContent(topRight, midLeft, midRight, bottomLeft widget.PreferredSizeLocateableWidget) *widget.Container {
	cell := func(c widget.PreferredSizeLocateableWidget) widget.PreferredSizeLocateableWidget {
		if c == nil {
			return widget.NewContainer()
		}
		return c
	}
	content := styled.NewItemGridContainer()
	content.AddChild(widget.NewContainer(), widget.NewContainer(), cell(topRight))
	content.AddChild(cell(midLeft), widget.NewContainer(), cell(midRight))
	content.AddChild(cell(bottomLeft), widget.NewContainer(), widget.NewContainer())
	return content
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
func menuNavHint(hasTabs bool, extras ...string) string {
	parts := make([]string, 0, 4+len(extras))
	if hasTabs {
		parts = append(parts, consts.IconArrowLeft+consts.IconArrowRight+" タブ")
	}
	parts = append(parts, consts.IconArrowUp+consts.IconArrowDown+" 選択")
	parts = append(parts, consts.IconKeyEnter+" 決定")
	parts = append(parts, extras...)
	parts = append(parts, consts.IconKeyEsc+" 戻る")
	return strings.Join(parts, "   ")
}

// centerRow は子を水平中央に置くアンカーコンテナを返す。タブ帯や見出しの中央寄せに使う
func centerRow(child widget.PreferredSizeLocateableWidget) *widget.Container {
	row := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	child.GetWidget().LayoutData = widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter}
	row.AddChild(child)
	return row
}

// gameLogTopY は画面下部のゲームログのボックス上端 Y を返す。
// モーダルやウィンドウをこの上端より上に収め、ログと重ならないようにする基準に使う。
func gameLogTopY(screenHeight int) int {
	cfg := hud.DefaultMessageAreaConfig
	logHeight := cfg.LogAreaMargin*2 + cfg.MaxLogLines*cfg.LineHeight + cfg.YPadding*2
	return screenHeight - logHeight - theme.Space3
}

// wrapModalRoot は root を画面より一回り小さい中央モーダルとして包む。
// 外周は背景を持たず透明にし、周囲に後ろのフィールドを覗かせる。動詞タブ画面と各メニューで共通に使う。
// 下端はゲームログの上端より上で止め、ログと重ならないようにする。
func wrapModalRoot(root *widget.Container) *widget.Container {
	bottom := consts.GameHeight - gameLogTopY(consts.GameHeight) + theme.Space3
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

// getCenterWinRect はゲームワールドから画面サイズを取得してウィンドウ位置を計算する
// TODO: package移動する
func getCenterWinRect(world w.World) image.Rectangle {
	windowWidth, windowHeight := 400, 400 // ウィンドウサイズの設定

	// worldから実際の画面サイズを取得
	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height

	// 横は画面中央。縦はゲームログの上端より上の領域に収めて、ログと重ならないようにする
	x := screenWidth/2 - windowWidth/2
	logTop := gameLogTopY(screenHeight)
	y := max((logTop-windowHeight)/2, theme.Space3)

	rect := image.Rect(x, y, x+windowWidth, y+windowHeight)
	return rect
}

// ================

// 共通の文字列定数
const (
	// UI表示用の定数
	TextNoDescription = "説明なし" // アイテムの説明がない場合の表示文字列
	TextClose         = "閉じる"  // メニューやウィンドウを閉じる際の表示文字列
	// メニューアクションのラベル。選択肢の生成と分岐で同じ定数を使い、
	// ラベル変更で switch 分岐が黙って死ぬのを防ぐ
	TextCraft = "合成する" // 合成メニューの合成アクション
	TextBuy   = "購入する" // 商店メニューの購入アクション
	TextSell  = "売却する" // 商店メニューの売却アクション
	TextHire  = "雇用する" // 酒場メニューの雇用アクション
)
