package menuframe

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// RenderList の検証はツリー構造で行う。CollectLabels は描画せず Value を集めるだけなので
// フェイスも ebiten も要らず、完全に並列でよい。選択の背景強調はピクセルなので golden 側で見る。

func labelsOf(items []uicore.Drawable) []string {
	labels := make([]string, 0, len(items))
	for _, it := range uicore.Placeable(items) {
		labels = append(labels, uicore.CollectLabels(it)...)
	}
	return labels
}

func TestRenderMenuListUI_Indentは行全体を字下げする(t *testing.T) {
	t.Parallel()
	res := resources.UIResources{Text: &resources.TextResources{}}
	cols := styled.Cols(styled.Name())

	// itemIndex に負値を渡し選択を持たせず、字下げの有無だけを見る
	plain, _ := RenderList(-1, []Row{{Cells: styled.TextCells("項目")}}, cols, ListOpts{ItemsPerPage: 10}, res)
	indented, _ := RenderList(-1, []Row{{Cells: styled.TextCells("項目"), Indent: 1}}, cols, ListOpts{ItemsPerPage: 10}, res)

	// 字下げ行は先頭に空トラックが1つ増える。ラベルは変わらない
	plainChildren := len(uicore.Placeable(plain)[0].Children())
	indentedChildren := len(uicore.Placeable(indented)[0].Children())
	assert.Equal(t, plainChildren+1, indentedChildren, "字下げは先頭に空トラックを1つ足す")
	assert.Equal(t, labelsOf(plain), labelsOf(indented), "字下げしてもラベルは変わらない")
}

func TestRenderMenuListUI_無効行も描画されラベルは残る(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{Cells: styled.TextCells("有効")},
		{Cells: styled.TextCells("無効"), Disabled: true},
	}
	items, _ := RenderList(-1, rows, styled.Cols(styled.Name()), ListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	assert.Equal(t, []string{"有効", "無効"}, labelsOf(items), "無効行も消えず淡色で描く")
}

func TestRenderMenuListUI_単一ページは見出しと行を並べる(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{Cells: styled.TextCells("見出し"), Header: true},
		{Cells: styled.TextCells("項目A")},
		{Cells: styled.TextCells("項目B")},
	}
	items, _ := RenderList(1, rows, styled.Cols(styled.Name()), ListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	labels := labelsOf(items)

	assert.Equal(t, []string{"見出し", "項目A", "項目B"}, labels)
}

func TestRenderMenuListUI_多数行はページ送りし空行で高さを保つ(t *testing.T) {
	t.Parallel()
	rows := make([]Row, 30)
	for i := range rows {
		rows[i] = Row{Cells: styled.TextCells(fmt.Sprintf("Item %d", i+1))}
	}
	items, pager := RenderList(0, rows, styled.Cols(styled.Name()), ListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	labels := labelsOf(items)

	// 先頭ページの10件だけが出て、2ページ目の行は出ない
	assert.Equal(t, []string{
		"Item 1", "Item 2", "Item 3", "Item 4", "Item 5",
		"Item 6", "Item 7", "Item 8", "Item 9", "Item 10",
	}, labels)
	assert.Contains(t, pager, "/", "複数ページはページ表示をフッタ向けに返す")
}

func TestRenderMenuListUI_最終ページの余りは空行で埋めて高さを保つ(t *testing.T) {
	t.Parallel()
	rows := make([]Row, 25)
	for i := range rows {
		rows[i] = Row{Cells: styled.TextCells(fmt.Sprintf("Item %d", i+1))}
	}
	// itemIndex=22 は3ページ目に属し、3ページ目は Item21〜25 の5件しかない
	items, pager := RenderList(22, rows, styled.Cols(styled.Name()), ListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	labels := labelsOf(items)

	assert.Equal(t, []string{
		"Item 21", "Item 22", "Item 23", "Item 24", "Item 25",
		"", "", "", "", "",
	}, labels, "余った5件は空行で埋め、ページを繰っても高さが変わらないようにする")
	assert.Equal(t, "3/3", pager, "最終ページの番号もページ表示に出る")
}

func TestRenderMenuListUI_列見出し行は各行のHeader行とは別に先頭へ1回出す(t *testing.T) {
	t.Parallel()
	rows := []Row{{Cells: styled.TextCells("項目A")}}
	opts := ListOpts{HeaderRow: []string{"名前"}, ItemsPerPage: 10}
	items, _ := RenderList(-1, rows, styled.Cols(styled.Name()), opts, resources.UIResources{Text: &resources.TextResources{}})

	assert.Equal(t, []string{"名前", "項目A"}, labelsOf(items), "opts.HeaderRowは表の先頭に一度だけ出る列見出し")
}

func TestRenderMenuListUI_行が無いときはEmptyTextを表示する(t *testing.T) {
	t.Parallel()
	opts := ListOpts{EmptyText: "アイテムがありません", ItemsPerPage: 10}
	items, _ := RenderList(-1, nil, styled.Cols(styled.Name()), opts, resources.UIResources{Text: &resources.TextResources{}})

	assert.Equal(t, []string{"アイテムがありません"}, labelsOf(items))
}

func TestRenderMenuListUI_行が無くEmptyText未指定なら何も出さない(t *testing.T) {
	t.Parallel()
	items, pager := RenderList(-1, nil, styled.Cols(styled.Name()), ListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})

	assert.Empty(t, items)
	assert.Empty(t, pager)
}

func TestSelectionRow_中身を持たず意匠だけの行を返す(t *testing.T) {
	t.Parallel()
	res := resources.UIResources{Text: &resources.TextResources{}}

	row := SelectionRow(res, false)

	assert.Empty(t, row.Children(), "中身は呼び出し側が別に重ねるので子を持たない")
}

// resolveColWidths は非公開関数のため internal test で直接呼び出す。
func TestResolveColWidths_Icon列は正方の固定幅(t *testing.T) {
	t.Parallel()

	cols := []styled.Col{styled.Icon()}
	widths := resolveColWidths(cols, nil, nil, nil)

	assert.Equal(t, []int{theme.MenuIconW}, widths, "Icon列は内容に関わらずMenuIconWの固定幅になる")
}

func TestResolveColWidths_Grow列は0のままにする(t *testing.T) {
	t.Parallel()

	cols := []styled.Col{styled.Name()}
	rows := []Row{{Cells: styled.TextCells("とても長い項目名がここに入る想定の文字列")}}
	widths := resolveColWidths(cols, nil, rows, nil)

	assert.Equal(t, []int{0}, widths, "Grow列は行のflexが余り幅を割り当てるのでここでは0のまま")
}

func TestResolveColWidths_Fit列のアイコンは画像の実幅で測る(t *testing.T) {
	t.Parallel()

	icon := ebiten.NewImage(30, 12)
	cols := []styled.Col{styled.Fit()}
	rows := []Row{{Cells: []styled.Cell{styled.IconCell(icon)}}}
	widths := resolveColWidths(cols, nil, rows, nil)

	assert.Equal(t, []int{30 + theme.Space3}, widths, "Fit列のアイコンセルは正方に丸めず横長のまま画像の実幅で測る")
}

func TestResolveColWidths_列数より少ないセルの行は読み飛ばす(t *testing.T) {
	t.Parallel()

	icon := ebiten.NewImage(25, 25)
	cols := []styled.Col{styled.Name(), styled.Fit()}
	rows := []Row{
		{Cells: styled.TextCells("項目のみ")},
		{Cells: []styled.Cell{styled.TextCell("項目"), styled.IconCell(icon)}},
	}
	widths := resolveColWidths(cols, nil, rows, nil)

	assert.Equal(t, []int{0, 25 + theme.Space3}, widths, "セルを持たない行はpanicせず読み飛ばし、2列目の幅は残る行のアイコンから決まる")
}

func TestResolveColWidths_見出しもFit列の実測に加える(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t, testutil.WithUI())
	face := world.Resources.UIResources.Text.BodyFace

	icon := ebiten.NewImage(5, 5)
	cols := []styled.Col{styled.Fit()}
	headerRow := []string{"非常に長い見出し文字列テスト"}
	rows := []Row{{Cells: []styled.Cell{styled.IconCell(icon)}}}

	widths := resolveColWidths(cols, headerRow, rows, face)

	wantHeaderW := uicore.MeasureTextWidth(headerRow[0], face) + theme.Space3
	assert.Equal(t, []int{wantHeaderW}, widths, "見出しがアイコンより広ければ見出しの実測幅が列幅を決める")
	assert.Greater(t, widths[0], 5+theme.Space3, "見出しがアイコンより広いことの前提")
}
