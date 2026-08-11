package messagewindow

import (
	"github.com/ebitenui/ebitenui/widget"
	w "github.com/kijimaD/ruins/internal/world"
)

// view はタブメニューの描画を担当する。状態管理は外部（hooks）が行う
type view struct {
	config    tabMenuConfig
	state     viewState
	uiBuilder *uiBuilder
}

// newView は view を作成する
func newView(config tabMenuConfig, world w.World) *view {
	return &view{
		config:    config,
		uiBuilder: newUIBuilder(world),
	}
}

// SetState は外部から描画状態を設定する
func (v *view) SetState(state viewState) {
	v.state = state
}

// BuildUI はメニューのUIを構築する
func (v *view) BuildUI() *widget.Container {
	return v.uiBuilder.BuildUI(v.config, v.state)
}

// UpdateFocus はフォーカス表示を更新する
func (v *view) UpdateFocus() {
	v.uiBuilder.UpdateFocus(v.config, v.state)
}

// GetCurrentPage は現在のページ番号を返す（1ベース、表示用）
func (v *view) GetCurrentPage() int {
	return currentPage(v.config, v.state) + 1
}
