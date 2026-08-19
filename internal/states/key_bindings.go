package states

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menuinput"
)

// detailOpenBindings は x で選択中の詳細モーダルを開く束縛表。詳細を持つメニューが共有する。
// Shift+x は調べる動詞とキーを共有するので、Shift 無しに限って詳細に割り当てる
var detailOpenBindings = []menuinput.Binding{
	{Key: ebiten.KeyX, Shift: menuinput.ShiftForbidden, Action: inputmapper.ActionOpenItemDetail},
}
