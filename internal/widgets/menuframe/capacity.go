package menuframe

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// 一覧モーダルの行高と内側余白。states の buildTabScreenUI・buildPanelScreenUI の
// panelScreenRowH・panelScreenPad に対応する。ずれると描画が枠からはみ出すので合わせる。
const (
	capacityRowH = 26
	capacityPad  = 12
	// footerMargin は最終エントリとフッタの間に空ける行数。近すぎないよう余白を確保する
	footerMargin = 2
)

// ListCapacity はタブ画面が1ページに収められる行数を返す。大モーダル ModalRect の高さから算出する。
// カーソルの改ページと描画のページングの両方がこれを呼ぶため、両者は同じ値で揃う。
// 構成は見出しの有無とタブ帯の有無で決まる。ページ表示行とフッタは常に1行ずつ差し引く。
func ListCapacity(world w.World, hasHeader, hasTabs bool) int {
	// タブ画面は下端まで広い大モーダル。最終エントリとフッタが接しないよう余白を確保する
	return capacityForHeight(ModalRect(world).Dy(), hasHeader, hasTabs, footerMargin)
}

// PanelCapacity はパネル画面が1ページに収められる行数を返す。項目数相応の小さなパネルは
// CenterWindowRect に収まる高さで算出する。選択メニュー、choice、などで使う。
// パネルは中身の高さに合わせて縮むので、フッタ前の余白は要らない。
func PanelCapacity(world w.World, hasHeader, hasTabs bool) int {
	return capacityForHeight(CenterWindowRect(world).Dy(), hasHeader, hasTabs, 0)
}

// capacityForHeight はモーダル高さから枠部品と余白を引いてデータ行数を求める。
func capacityForHeight(modalHeight int, hasHeader, hasTabs bool, margin int) int {
	total := (modalHeight - capacityPad*2) / capacityRowH
	overhead := 2 + margin // ページ表示行 + フッタ + フッタ前の余白
	if hasHeader {
		overhead++
	}
	if hasTabs {
		overhead++
	}
	return max(total-overhead, 1)
}
