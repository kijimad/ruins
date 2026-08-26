package states

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kijimaD/ruins/internal/resources"

	"github.com/stretchr/testify/assert"

	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
)

// renderMenuListUI の検証はツリー構造で行う。CollectLabels は描画せず Value を集めるだけなので
// フェイスも ebiten も要らず、完全に並列でよい。選択の背景強調はピクセルなので golden 側で見る。

func labelsOf(items []ui.Widget) []string {
	labels := make([]string, 0, len(items))
	for _, it := range items {
		labels = append(labels, ui.CollectLabels(it)...)
	}
	return labels
}

func TestRenderMenuListUI_単一ページは見出しと行を並べる(t *testing.T) {
	t.Parallel()
	rows := []menuRow{
		{Cells: styled.TextCells("見出し"), Header: true},
		{Cells: styled.TextCells("項目A")},
		{Cells: styled.TextCells("項目B")},
	}
	items := renderMenuListUI(1, rows, []int{200}, []styled.TextAlign{styled.AlignLeft}, menuListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	labels := labelsOf(items)

	assert.Contains(t, labels, "見出し", "見出し行が出る")
	assert.Contains(t, labels, "項目A")
	assert.Contains(t, labels, "項目B")
}

func TestRenderMenuListUI_多数行はページ送りし空行で高さを保つ(t *testing.T) {
	t.Parallel()
	rows := make([]menuRow, 30)
	for i := range rows {
		rows[i] = menuRow{Cells: styled.TextCells(fmt.Sprintf("Item %d", i+1))}
	}
	items := renderMenuListUI(0, rows, []int{200}, []styled.TextAlign{styled.AlignLeft}, menuListOpts{ItemsPerPage: 10}, resources.UIResources{Text: &resources.TextResources{}})
	labels := labelsOf(items)

	assert.Contains(t, labels, "Item 1", "先頭ページの先頭が出る")
	assert.NotContains(t, labels, "Item 11", "2ページ目の行は出さない")
	assert.Contains(t, strings.Join(labels, " "), "/", "複数ページはページ表示を出す")
}
