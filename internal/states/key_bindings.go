package states

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
)

// detailOpenBindings は x で選択中の詳細モーダルを開く束縛表。詳細を持つメニューが共有する。
// Shift+x は調べる動詞とキーを共有するので、Shift 無しに限って詳細に割り当てる
var detailOpenBindings = []keybind.Binding{
	{Key: ebiten.KeyX, Shift: keybind.ShiftForbidden, Action: inputmapper.ActionOpenItemDetail},
}
