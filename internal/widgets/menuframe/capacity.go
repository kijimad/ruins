package menuframe

import (
	"github.com/kijimaD/ruins/internal/resources"
)

// ListCapacity は一覧を持つモーダルが1ページに収められる行数を返す。
// 本体は internal/ui の固定枠モーダルで描くので、CenterWindowRect の高さと行高から算出する。
// カーソルの改ページと描画のページングの両方がこれを呼ぶため、両者は同じ値で揃う。
// 構成は見出しの有無とタブ帯の有無で決まる。ページ表示行とフッタは常に1行ずつ差し引く。
//
// 幾何は states 側の buildTabScreenUI と揃える。panelScreenRowH と panelScreenPad に対応する。
// ずれると描画が枠からはみ出すので、変えるときは両方を合わせる。
func ListCapacity(_ resources.UIResources, hasHeader, hasTabs bool) int {
	// modalHeight は CenterWindowRect の高さ。rowH と pad は internal/ui タブ画面の行高と内側余白
	const (
		modalHeight = 400
		rowH        = 26
		pad         = 12
	)
	total := (modalHeight - pad*2) / rowH
	overhead := 2 // ページ表示行 + フッタ
	if hasHeader {
		overhead++
	}
	if hasTabs {
		overhead++
	}
	n := max(total-overhead, 1)
	return n
}
