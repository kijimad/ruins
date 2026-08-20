package menuloop

import (
	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// KeyHelpState は現在の文脈のキー束縛一覧を表示するステート。? でどの画面からも開ける。
// 表示は開いた画面の束縛表から導出するので、キーを変えれば一覧が追随する。
// メニュー画面では Screen の入力ゲートが ActionOpenKeyHelp を吸ってこれを push し、
// メニュー外の画面は自分の DoAction から push する
type KeyHelpState struct {
	es.BaseState[w.World]
	tables [][]keybind.Binding
	widget *ebitenui.UI
}

var _ es.State[w.World] = &KeyHelpState{}

// NewKeyHelpState は tables のキー一覧を表示するヘルプのファクトリを返す
func NewKeyHelpState(tables ...[]keybind.Binding) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &KeyHelpState{tables: tables}, nil
	}
}

// keyHelpBindings はヘルプ画面自身の束縛表。Esc か開いたときと同じ ? で閉じる
var keyHelpBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
	{Key: ebiten.KeySlash, Shift: keybind.ShiftRequired, Action: inputmapper.ActionCloseMenu},
}

// OnStart は一覧の UI を組む。束縛表は state の寿命の間変わらないので1度だけ組めばよい
func (st *KeyHelpState) OnStart(world w.World) error {
	res := world.Resources.UIResources
	list := styled.NewVerticalContainer()
	for _, e := range keybind.HintEntries(world, st.tables...) {
		list.AddChild(styled.NewBodyText(e.Keys+"  "+e.Label, theme.TextPrimary, res))
	}
	st.widget = menuframe.NewPanelScreen(res, query.T(world, "Key bindings"), list,
		keybind.NavHint(world, false, keyHelpBindings))
	return nil
}

// Update は閉じる入力だけを読む。ヘルプ表示中は時間を進めない
func (st *KeyHelpState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, keyHelpBindings); ok && action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}
	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw は保持中の一覧を描く
func (st *KeyHelpState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}
