package tabmenu

import (
	"github.com/ebitenui/ebitenui/widget"
	w "github.com/kijimaD/ruins/internal/world"
)

// View はタブメニューの描画を担当する。状態管理は外部（hooks）が行う
type View struct {
	config    Config
	state     ViewState
	uiBuilder *uiBuilder
}

// NewView は View を作成する
func NewView(config Config, world w.World) *View {
	return &View{
		config:    config,
		uiBuilder: newUIBuilder(world),
	}
}

// SetState は外部から描画状態を設定する
func (v *View) SetState(state ViewState) {
	v.state = state
}

// BuildUI はメニューのUIを構築する
func (v *View) BuildUI() *widget.Container {
	return v.uiBuilder.BuildUI(v.config, v.state)
}

// UpdateFocus はフォーカス表示を更新する
func (v *View) UpdateFocus() {
	v.uiBuilder.UpdateFocus(v.config, v.state)
}

// GetCurrentPage は現在のページ番号を返す（1ベース、表示用）
func (v *View) GetCurrentPage() int {
	return currentPage(v.config, v.state) + 1
}
