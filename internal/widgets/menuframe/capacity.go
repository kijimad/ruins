package menuframe

import (
	"sync"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
)

// ListCapacity は一覧を持つモーダルが1ページに収められる行数を実測で返す。
// チェーム込みのダミー UI を組んで preferred 高さを測り、画面へ収まる最大の行数を求める。
// レイアウトの padding や部品の高さを式に写さないため、それらの変更に自動で追随する。
// 構成は見出しの有無とタブ帯の有無で決まる。画面寸法とフォントは実行中変わらないため、
// 構成ごとに1度だけ測って使い回す
func ListCapacity(res resources.UIResources, hasHeader, hasTabs bool) int {
	listCapacityMu.Lock()
	defer listCapacityMu.Unlock()
	key := [2]bool{hasHeader, hasTabs}
	if n, ok := listCapacityCache[key]; ok {
		return n
	}
	n := 1
	for n < listCapacityLimit && listHeight(res, hasHeader, hasTabs, n+1) <= consts.GameHeight {
		n++
	}
	listCapacityCache[key] = n
	return n
}

// listCapacityLimit は探索の上限。測定の不具合で高さが増えない場合の無限ループを防ぐ
const listCapacityLimit = 100

var (
	listCapacityMu    sync.Mutex
	listCapacityCache = map[[2]bool]int{}
)

// listHeight は n 行の満杯ページを持つモーダル全体の preferred 高さを返す。
// 一覧の中身はページ表示行とテーブル行の縦積みで、renderMenuList と部品構成を揃えている。
// 高さだけが要るので中身はダミーで組む。実物との一致は states 側のテストが
// 実際の一覧を組んで固定する
func listHeight(res resources.UIResources, hasHeader, hasTabs bool, n int) int {
	content := styled.NewVerticalContainer()
	content.AddChild(styled.NewPageIndicator(" ", res))
	table := styled.NewTableContainer(nil, res)
	notSelected := false
	for range n {
		styled.NewTableRow(table, []int{20, 300}, styled.TextCells(" ", " "), nil, &notSelected, res)
	}
	content.AddChild(table)
	p := TabScreen{Content: content, Footer: " "}
	if hasHeader {
		p.Header = " "
	}
	if hasTabs {
		p.TabLabels = []string{" "}
	}
	_, h := NewTabScreen(res, p).Container.PreferredSize()
	return h
}
