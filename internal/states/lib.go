package states

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

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

// itemMenuColumns はアイコン列と伸縮する名前列の共通の先頭2列に、メニュー固有の trailing 列を
// 足した列定義を返す。対象メニュー間でアイコンと名前の見た目を統一する。
// trailing は数値列など画面ごとの追加列を意味的に渡す
func itemMenuColumns(trailing ...styled.Col) []styled.Col {
	return append([]styled.Col{styled.Icon(), styled.Name()}, trailing...)
}

// menuIcon は entity のスプライトを一覧のアイコン列の大きさで返す。縮小はリソース側が
// 一度だけ済ませるので、描画は等倍になる
func menuIcon(world w.World, e ecs.Entity) *ebiten.Image {
	return world.Resources.Icons.Sized(world.Resources.SpriteSheets, world.Components.SpriteRender.Get(e), theme.MenuIconW)
}

// itemMenuRow はアイテム entity から共通の先頭部 [アイコンセル + 名前×個数セル] を組み、
// メニュー固有の trailing 文字列セルを後続に付けた menuframe.Row を返す。
// 個数は呼び出し側が GroupStacks で確定済みの値を渡す。ここで数え直すと一覧の行ごとに
// 全走査が走るため、束ねた結果を持ち回って表示する
func itemMenuRow(world w.World, e ecs.Entity, count int, trailing ...string) menuframe.Row {
	name := query.GetEntityName(e, world)
	icon := menuIcon(world, e)
	label := query.FormatNameCount(name, count)
	// 腐敗食は新鮮以外のとき鮮度を名前に添え、状態を見て食べるか捨てるか判断できるようにする
	if marker := query.FreshnessMarker(world, e); marker != "" {
		label += " (" + marker + ")"
	}
	cells := append([]styled.Cell{styled.IconCell(icon), styled.TextCell(label)}, styled.TextCells(trailing...)...)
	return menuframe.Row{Cells: cells}
}

// ================
