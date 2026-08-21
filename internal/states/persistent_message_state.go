package states

import (
	"github.com/hajimehoshi/ebiten/v2"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/widgets/messagewindow"
	w "github.com/kijimaD/ruins/internal/world"
)

// PersistentMessageState は選択肢実行後もウィンドウを開いたままにするMessageStateラッパー
type PersistentMessageState struct {
	MessageState
}

var _ es.State[w.World] = &PersistentMessageState{}

// persistentMessageBindings は常設メッセージの束縛表。Esc で明示的に閉じる
var persistentMessageBindings = []keybind.Binding{
	{Key: ebiten.KeyEscape, Action: inputmapper.ActionCloseMenu},
}

// Update はゲームステートの更新処理を行う
func (st *PersistentMessageState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := keybind.ReadInput(world, persistentMessageBindings); ok && action == inputmapper.ActionCloseMenu {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	if st.messageWindow != nil {
		if err := st.messageWindow.Update(); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}

		if st.messageWindow.IsClosed() {
			// BaseStateで設定された遷移を優先確認
			if transition := st.ConsumeTransition(); transition.Type != es.TransNone {
				return transition, nil
			}
			// PersistentMessageStateは自動Popしない
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		// MessageWindowがアクティブな間は何もしない
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	return st.ConsumeTransition(), nil
}

// OnResume はステートが再開される際に呼ばれる。
// messageData は OnStart で build 済みなので再構築のみ行い build は呼ばない。
// 表示中の言語切り替えには追従せず、切り替えるには OnStart から開き直す
func (st *PersistentMessageState) OnResume(world w.World) error {
	// メッセージウィンドウを強制的に再構築
	if st.messageData != nil {
		st.messageWindow = messagewindow.NewWindow(world, st.messageData)
	}
	return nil
}

// NewPersistentMessageState は組み立て済みメッセージから永続メッセージステートを作成する。
// 構築を build へ一本化するため、受け取った messageData を返すだけの build で包む。
// world 依存の構築が要る呼び出しは &PersistentMessageState{} を直接組んで build を設定する
func NewPersistentMessageState(messageData *messagedata.MessageData) *PersistentMessageState {
	return &PersistentMessageState{
		build: func(_ w.World) *messagedata.MessageData { return messageData },
	}
}
