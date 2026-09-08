package menuframe_test

import (
	"fmt"
	"testing"

	"github.com/kijimaD/ruins/internal/resources"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/uicore"
)

// menuframe.RenderList の検証はツリー構造で行う。CollectLabels は描画せず Value を集めるだけなので
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
	plain, _ := menuframe.RenderList(-1, []menuframe.Row{{Cells: styled.TextCells("項目")}}, cols, menuframe.ListOpts{ItemsPerPage: 10}, res)
	indented, _ := menuframe.RenderList(-1, []menuframe.Row{{Cells: styled.TextCells("項目"), Indent: 1}}, cols, menuframe.ListOpts{ItemsPerPage: 10}, res)

	// 字下げ行は先頭に空トラックが1つ増える。ラベルは変わらない
	plainChildren := len(uicore.Placeable(plain)[0].Children())
	indentedChildren := len(uicore.Placeable(indented)[0].Children())
	assert.Equal(t, plainChildren+1, indentedChildren, "字下げは先頭に空トラックを1つ足す")
	assert.Equal(t, labelsOf(plain), labelsOf(indented), "字下げしてもラベルは変わらない")
}

func TestRenderMenuListUI_単一ページは見出しと行を並べる(t *testing.T) {
	t.Parallel()
	rows := []menuframe.Row{
		{Cells: styled.TextCells("見出し"), Header: true},
		{Cells: styled.TextCells("項目A")},
		{Cells: styled.TextCells("項目B")},
	}
	items, _ := menuframe.RenderList(1, rows, styled.Cols(styled.Name()), menuframe.ListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	labels := labelsOf(items)

	assert.Equal(t, []string{"見出し", "項目A", "項目B"}, labels)
}

func TestRenderMenuListUI_多数行はページ送りし空行で高さを保つ(t *testing.T) {
	t.Parallel()
	rows := make([]menuframe.Row, 30)
	for i := range rows {
		rows[i] = menuframe.Row{Cells: styled.TextCells(fmt.Sprintf("Item %d", i+1))}
	}
	items, pager := menuframe.RenderList(0, rows, styled.Cols(styled.Name()), menuframe.ListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	labels := labelsOf(items)

	// 先頭ページの10件だけが出て、2ページ目の行は出ない
	assert.Equal(t, []string{
		"Item 1", "Item 2", "Item 3", "Item 4", "Item 5",
		"Item 6", "Item 7", "Item 8", "Item 9", "Item 10",
	}, labels)
	assert.Contains(t, pager, "/", "複数ページはページ表示をフッタ向けに返す")
}
