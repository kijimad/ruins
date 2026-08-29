package menuframe

import (
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// ListCapacity は一覧が1ページに収められる行数を返す。大モーダル ModalRect の高さから算出する。
// カーソルの改ページと描画のページングの両方がこれを呼ぶため、両者は同じ値で揃う。
// タブ画面はこの高さに収まり、パネル画面 choice はこの件数へ合わせて中身の高さまで伸びる。
// 構成は見出しの有無とタブ帯の有無で決まる。フッタ・フッタ前の余白を差し引く。
// 行高と余白は theme のメニュートークンを使い、描画側と同じ値で揃える。
func ListCapacity(world w.World, hasHeader, hasTabs bool) int {
	total := (ModalRect(world).Dy() - theme.MenuPad*2) / theme.MenuTabRowH
	overhead := 2 + theme.MenuFooterGap // ページ表示行 + フッタ + フッタ前の余白
	if hasHeader {
		overhead++
	}
	if hasTabs {
		overhead++
	}
	return max(total-overhead, 1)
}
