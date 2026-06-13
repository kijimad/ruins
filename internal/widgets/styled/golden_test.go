package styled_test

import (
	"os"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
)

// TestMain はebitenグラフィックスコンテキスト内で全テストを実行する。
// これにより各テスト関数で RenderWidget + ReadPixels が使える
func TestMain(m *testing.M) {
	os.Exit(vrt.RunTestMain(m))
}

func TestGolden_ListItemText_Selected(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("選択中のアイテム", theme.TextPrimary, true, res))
		return root
	}, 300, 30)
}

func TestGolden_ListItemText_Unselected(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("非選択のアイテム", theme.TextPrimary, false, res))
		return root
	}, 300, 30)
}

func TestGolden_ListItemText_WithLabels(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("回復薬", theme.TextPrimary, true, res, "x3", "1.5kg"))
		return root
	}, 400, 30)
}

func TestGolden_ListItemText_Multiple(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("選択中", theme.TextPrimary, true, res))
		root.AddChild(styled.NewListItemText("非選択1", theme.TextPrimary, false, res))
		root.AddChild(styled.NewListItemText("非選択2", theme.TextPrimary, false, res))
		return root
	}, 300, 90)
}

func TestGolden_TableRow_Selected(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{100, 200, 80}
		container := styled.NewTableContainer(widths, res)
		selected := true
		styled.NewTableRow(container, widths,
			[]string{"回復薬", "HPを回復する", "3"},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight},
			&selected, res)
		return container
	}, 400, 30)
}

func TestGolden_TableRow_Unselected(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{100, 200, 80}
		container := styled.NewTableContainer(widths, res)
		unselected := false
		styled.NewTableRow(container, widths,
			[]string{"鉄鉱石", "合成素材", "12"},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight},
			&unselected, res)
		return container
	}, 400, 30)
}

func TestGolden_TableHeaderRow(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{100, 200, 80}
		container := styled.NewTableContainer(widths, res)
		styled.NewTableHeaderRow(container, widths, []string{"名前", "説明", "数量"}, res)
		return container
	}, 400, 30)
}

func TestGolden_TableWithHeaderAndRows(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{120, 180, 60}
		container := styled.NewTableContainer(widths, res)
		styled.NewTableHeaderRow(container, widths, []string{"名前", "説明", "数量"}, res)
		selected := true
		unselected := false
		styled.NewTableRow(container, widths,
			[]string{"回復薬", "HPを回復する", "3"}, nil, &selected, res)
		styled.NewTableRow(container, widths,
			[]string{"鉄鉱石", "合成素材", "12"}, nil, &unselected, res)
		return container
	}, 400, 80)
}

func TestGolden_MenuText(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewMenuText("メニューテキスト", res))
		return root
	}, 300, 30)
}

func TestGolden_DescriptionText(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewDescriptionText("補助テキスト: 小さめのフォント", res))
		return root
	}, 400, 25)
}

func TestGolden_PageIndicator(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewPageIndicator("1/3", res))
		return root
	}, 300, 25)
}

// ================
// ListItemText エッジケース
// ================

func TestGolden_ListItemText_EmptyText(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("", theme.TextPrimary, true, res))
		return root
	}, 300, 30)
}

func TestGolden_ListItemText_LongText(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("とても長いアイテム名が入ったリスト項目のテスト用テキスト", theme.TextPrimary, true, res))
		return root
	}, 600, 30)
}

func TestGolden_ListItemText_ManyLabels(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("装備品", theme.TextPrimary, true, res, "x5", "2.3kg", "500G", "Lv3"))
		return root
	}, 500, 30)
}

func TestGolden_ListItemText_UnselectedWithLabels(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("回復薬", theme.TextPrimary, false, res, "x3", "1.5kg"))
		return root
	}, 400, 30)
}

// ================
// TableRow エッジケース
// ================

func TestGolden_TableRow_SingleColumn(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{300}
		container := styled.NewTableContainer(widths, res)
		selected := true
		styled.NewTableRow(container, widths,
			[]string{"単一カラムの行"},
			[]styled.TextAlign{styled.AlignLeft},
			&selected, res)
		return container
	}, 320, 30)
}

func TestGolden_TableRow_ManyColumns(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{80, 80, 60, 60, 60}
		container := styled.NewTableContainer(widths, res)
		selected := true
		styled.NewTableRow(container, widths,
			[]string{"名前", "種別", "攻撃", "防御", "重量"},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignRight, styled.AlignRight},
			&selected, res)
		return container
	}, 380, 30)
}

func TestGolden_TableRow_AllRightAligned(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{100, 100, 100}
		container := styled.NewTableContainer(widths, res)
		unselected := false
		styled.NewTableRow(container, widths,
			[]string{"100", "200", "300"},
			[]styled.TextAlign{styled.AlignRight, styled.AlignRight, styled.AlignRight},
			&unselected, res)
		return container
	}, 320, 30)
}

func TestGolden_TableRow_NonSelectable(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{100, 200}
		container := styled.NewTableContainer(widths, res)
		styled.NewTableRow(container, widths,
			[]string{"重量", "2.50kg"},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignRight},
			nil, res)
		return container
	}, 320, 30)
}

func TestGolden_TableFull(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		widths := []int{100, 100, 60}
		container := styled.NewTableContainer(widths, res)
		styled.NewTableHeaderRow(container, widths, []string{"名前", "説明", "数量"}, res)
		s0, s1, s2, s3 := true, false, false, false
		styled.NewTableRow(container, widths, []string{"回復薬", "HPを回復", "3"}, nil, &s0, res)
		styled.NewTableRow(container, widths, []string{"鉄鉱石", "合成素材", "12"}, nil, &s1, res)
		styled.NewTableRow(container, widths, []string{"聖水", "状態回復", "1"}, nil, &s2, res)
		styled.NewTableRow(container, widths, []string{"毒消し", "解毒", "5"}, nil, &s3, res)
		return container
	}, 300, 160)
}

// ================
// テキスト系バリエーション
// ================

func TestGolden_TitleText(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewTitleText("タイトルテキスト", res))
		return root
	}, 300, 25)
}

func TestGolden_BodyText(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewBodyText("本文テキストのサンプル", theme.TextPrimary, res))
		return root
	}, 400, 30)
}

func TestGolden_FragmentText_Colors(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(0),
			)),
		)
		root.AddChild(styled.NewFragmentText("赤テキスト", theme.StatusDanger, res))
		root.AddChild(styled.NewFragmentText("と", theme.TextPrimary, res))
		root.AddChild(styled.NewFragmentText("緑テキスト", theme.StatusSuccess, res))
		return root
	}, 400, 30)
}

func TestGolden_WindowHeaderContainer(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewWindowHeaderContainer("ウィンドウタイトル", res))
		return root
	}, 400, 35)
}

func TestGolden_PageIndicator_WithArrows(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewPageIndicator("↑ 2/5 ↓", res))
		return root
	}, 300, 25)
}

// ================
// 複合パターン
// ================

func TestGolden_ListItemText_MixedSelection(t *testing.T) {
	t.Parallel()
	res := vrt.SharedUIResources(t)
	vrt.AssertGolden(t, func() *widget.Container {
		root := verticalRoot()
		root.AddChild(styled.NewListItemText("非選択1", theme.TextPrimary, false, res))
		root.AddChild(styled.NewListItemText("非選択2", theme.TextPrimary, false, res))
		root.AddChild(styled.NewListItemText("選択中", theme.TextPrimary, true, res))
		root.AddChild(styled.NewListItemText("非選択3", theme.TextPrimary, false, res))
		root.AddChild(styled.NewListItemText("非選択4", theme.TextPrimary, false, res))
		return root
	}, 300, 150)
}

func verticalRoot() *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		)),
	)
}
