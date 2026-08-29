package states

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	"github.com/kijimaD/ruins/internal/widgets/ui"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
)

const (
	nameMinLength    = 1
	nameMaxLength    = 10
	errorDisplayTime = 2 * time.Second
)

// CharacterNamingState はキャラクター名前入力画面のステート。
//
// 入力はテキストチャンネルの例外で、束縛表と keybind.ReadInput を通さずキーを直読みする。
// 文字入力は ebiten.AppendInputChars と Backspace を直読みして props の名前へ反映する。
// IME の変換途中状態やカーソル編集は離散的な Action の語彙に還元できないため、
// Action チャンネルへは統一しない。再生が必要になれば文字列を別口で注入する
type CharacterNamingState struct {
	es.BaseState[w.World]
	mount *hooks.Mount[namingProps]
	body  ui.Widget
}

// NewCharacterNamingState は名付けステートのファクトリを返す
func NewCharacterNamingState() (es.State[w.World], error) {
	return &CharacterNamingState{}, nil
}

// State interface ================

var _ es.State[w.World] = &CharacterNamingState{}
var _ es.ActionHandler[w.World] = &CharacterNamingState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CharacterNamingState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CharacterNamingState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *CharacterNamingState) OnStart(world w.World) error {
	st.mount = hooks.NewMount[namingProps]()

	// 既存プレイヤーの名前を初期値として設定
	initialName := ""
	playerEntity, err := query.GetPlayerEntity(world)
	if err == nil {
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			initialName = nameComp.Name
		}
	}

	st.mount.SetProps(namingProps{
		CurrentName:  initialName,
		ErrorMessage: "",
	})
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CharacterNamingState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CharacterNamingState) Update(world w.World) (es.Transition[w.World], error) {
	props := st.mount.GetProps()

	// エラーメッセージの自動クリア
	expired, _, resetTimer := hooks.UseTimer(st.mount.Store(), "errorTimer", errorDisplayTime)
	if expired && props.ErrorMessage != "" {
		st.mount.SetProps(namingProps{
			CurrentName:  props.CurrentName,
			ErrorMessage: "",
		})
		resetTimer()
	}

	// 文字入力を名前へ取り込む
	if newName, changed := readTextInput(props.CurrentName); changed {
		st.mount.SetProps(namingProps{
			CurrentName:  newName,
			ErrorMessage: props.ErrorMessage,
		})
	}

	// 入力処理
	if action, ok := st.HandleInput(world.Resources.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	// dirty判定とUI再構築
	if st.mount.Update() || st.body == nil {
		st.body = st.buildUI(world)
	}
	return st.ConsumeTransition(), nil
}

// readTextInput は今フレームの文字入力と Backspace を名前へ反映する。
// 印字文字は AppendInputChars で受け取り、最大長までを追記する。Backspace は末尾1文字を消す
func readTextInput(current string) (string, bool) {
	name := current
	changed := false
	for _, r := range ebiten.AppendInputChars(nil) {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if utf8.RuneCountInString(name) >= nameMaxLength {
			break
		}
		name += string(r)
		changed = true
	}
	if input.GetSharedKeyboardInput().IsKeyPressedWithRepeat(ebiten.KeyBackspace) && name != "" {
		runes := []rune(name)
		name = string(runes[:len(runes)-1])
		changed = true
	}
	return name, changed
}

// Draw はスクリーンに描画する
func (st *CharacterNamingState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(theme.ScreenBackground)
	if st.body != nil {
		st.body.Draw(ui.NewEbitenCanvas(screen))
	}
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *CharacterNamingState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionMenuSelect, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}
	return "", false
}

// DoAction はActionを実行する
func (st *CharacterNamingState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return st.cancel(world), nil
	case inputmapper.ActionMenuSelect:
		return st.confirmName(world), nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("characterNaming: unsupported action: %s", action)
	}
}

// ================
// Props
// ================

// namingProps は名前入力画面のProps
type namingProps struct {
	CurrentName  string
	ErrorMessage string
}

// confirmName は名前を確定する
func (st *CharacterNamingState) confirmName(world w.World) es.Transition[w.World] {
	props := st.mount.GetProps()
	name := props.CurrentName
	nameLen := utf8.RuneCountInString(name)

	if nameLen < nameMinLength || nameLen > nameMaxLength {
		st.mount.SetProps(namingProps{
			CurrentName:  props.CurrentName,
			ErrorMessage: query.T(world, "Enter a name of 1 to 10 characters"),
		})
		_, startTimer, _ := hooks.UseTimer(st.mount.Store(), "errorTimer", errorDisplayTime)
		startTimer()
		return es.Transition[w.World]{Type: es.TransNone}
	}

	playerEntity, err := query.GetPlayerEntity(world)
	if err == nil {
		// 既存プレイヤーの名前を変更した
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			nameComp.Name = name
		}
		return es.Transition[w.World]{Type: es.TransPop}
	}

	// 職業選択画面へ遷移
	return es.Transition[w.World]{
		Type:          es.TransPush,
		NewStateFuncs: []es.StateFactory[w.World]{NewCharacterJobState(name)},
	}
}

// cancel はキャンセルする
func (st *CharacterNamingState) cancel(world w.World) es.Transition[w.World] {
	_, err := query.GetPlayerEntity(world)
	if err == nil {
		return es.Transition[w.World]{Type: es.TransPop}
	}
	return es.Transition[w.World]{
		Type:          es.TransReplace,
		NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState},
	}
}

// ================
// buildUI
// ================

// buildUI は名前入力画面を internal/uicore のツリーとして組む。
// タイトル・入力枠・エラー・ヒントを画面中央へ縦に並べる。入力枠には現在名とキャレットを描く
func (st *CharacterNamingState) buildUI(world w.World) ui.Widget {
	res := world.Resources.UIResources
	props := st.mount.GetProps()

	// 入力枠の中身。空なら placeholder を薄色で、入力中は名前にキャレットを添える。
	// 縦位置は VCenter で入力枠の中央へ合わせる
	var content *ui.Text
	if props.CurrentName == "" {
		content = ui.NewText(query.T(world, "Name"), res.Text.BodyFace, theme.TextSecondary)
	} else {
		content = ui.NewText(props.CurrentName+"|", res.Text.BodyFace, theme.TextPrimary)
	}
	content.VCenter = true

	hint := consts.IconKeyEnter + " " + query.T(world, "Confirm") + " / " + consts.IconKeyEsc + " " + query.T(world, "Back")
	return menuframe.FormScreen(world, res, query.T(world, "Name"), content, props.ErrorMessage, hint)
}
