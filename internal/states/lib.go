package states

import (
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// menuRow は一覧の1行。Cells は各列のセルで、アイコンと文字列が混ざってよい。
// Header が真なら見出し行でカーソルが止まらない
type menuRow struct {
	Cells  []styled.Cell
	Header bool
}

// menuListOpts は一覧描画の方針。Spaced はコマンドメニュー向けに行間を空けるか、
// HeaderRow は表の先頭に置く列見出し、選択やページ送りの対象には含めない、
// EmptyText は行が無いときに表の下へ出す説明。ページ表示はフッタ右端に出すので一覧側では扱わない
type menuListOpts struct {
	Spaced    bool
	HeaderRow []string
	EmptyText string
	// ItemsPerPage は1ページの行数の上書き。0 ならタブ帯つきモーダルの実測容量。
	// 見出しなど追加の枠部品を持つ画面は menuframe.ListCapacity で自分の容量を求めて渡す
	ItemsPerPage int
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

// itemMenuColumns は アイコン列と名前列の共通の先頭2列に、メニュー固有の trailing 列を足した
// 列定義を返す。アイコン列と名前列を揃え、対象メニュー間でアイコンと名前の見た目を統一する。
// trailing は数値列など画面ごとの追加列を意味的に渡す
func itemMenuColumns(nameWidth int, trailing ...styled.Col) []styled.Col {
	return append([]styled.Col{styled.Icon(), styled.Name(nameWidth)}, trailing...)
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

// ================
