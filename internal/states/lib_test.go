package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/vrt"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMenuList_実測容量が画面に収まる は ListCapacity の返す行数の満杯ページが
// 実際の renderMenuList と TabScreen で組んでも画面へ収まることを固定する。
// ListCapacity の測定はダミー構成なので、実物との部品ずれはこのテストが検出する
func TestMenuList_実測容量が画面に収まる(t *testing.T) {
	t.Parallel()

	res := vrt.SharedUIResources(t)
	cases := []struct {
		name      string
		hasHeader bool
	}{
		{"タブ帯のみ", false},
		{"見出しとタブ帯", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			capacity := menuframe.ListCapacity(res, tc.hasHeader, true)
			require.Positive(t, capacity)

			assert.LessOrEqual(t, realListHeight(res, tc.hasHeader, capacity), consts.GameHeight,
				"容量ぶんの満杯ページは画面に収まる")
			assert.Greater(t, realListHeight(res, tc.hasHeader, capacity+1), consts.GameHeight,
				"1行増やすと収まらない。つまり容量は目一杯の値になっている")
		})
	}
}

// realListHeight は実物の renderMenuList で1ページ n 行の一覧を組み、TabScreen で
// 包んだモーダル全体の preferred 高さを返す
func realListHeight(res resources.UIResources, hasHeader bool, n int) int {
	rows := make([]menuRow, n+1)
	for i := range rows {
		rows[i] = menuRow{Cells: styled.TextCells(" ", " ")}
	}
	list := renderMenuList(0, rows, []int{itemIconColumnWidth, menuRowWidth}, nil,
		menuListOpts{AlwaysIndicator: true, ItemsPerPage: n}, res)
	p := menuframe.TabScreen{Content: list, Footer: " ", TabLabels: []string{" "}}
	if hasHeader {
		p.Header = " "
	}
	_, h := menuframe.NewTabScreen(res, p).Container.PreferredSize()
	return h
}
