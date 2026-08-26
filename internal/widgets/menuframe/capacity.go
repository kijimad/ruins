package menuframe

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// 一覧モーダルの行高と内側余白。states の buildTabScreenUI・buildPanelScreenUI の
// panelScreenRowH・panelScreenPad に対応する。ずれると描画が枠からはみ出すので合わせる。
const (
	capacityRowH = 21 // 一覧行の高さ。元の ebitenui の行密度に合わせる
	capacityPad  = 12
	// footerMargin は最終エントリとフッタの間に空ける行数。近すぎないよう余白を確保する
	footerMargin = 2
)

// ListCapacity は一覧が1ページに収められる行数を返す。大モーダル ModalRect の高さから算出する。
// カーソルの改ページと描画のページングの両方がこれを呼ぶため、両者は同じ値で揃う。
// タブ画面はこの高さに収まり、パネル画面 choice はこの件数へ合わせて中身の高さまで伸びる。
// 構成は見出しの有無とタブ帯の有無で決まる。ページ表示行・フッタ・フッタ前の余白を差し引く。
func ListCapacity(world w.World, hasHeader, hasTabs bool) int {
	total := (ModalRect(world).Dy() - capacityPad*2) / capacityRowH
	overhead := 2 + footerMargin // ページ表示行 + フッタ + フッタ前の余白
	if hasHeader {
		overhead++
	}
	if hasTabs {
		overhead++
	}
	return max(total-overhead, 1)
}
