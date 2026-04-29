package states

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

const (
	nameMinLength    = 1
	nameMaxLength    = 10
	errorDisplayTime = 2 * time.Second
)

// CharacterNamingState はキャラクター名前入力画面のステート
type CharacterNamingState struct {
	es.BaseState[w.World]
	mount  *hooks.Mount[namingProps]
	widget *ebitenui.UI
}

// NewCharacterNamingState は名付けステートのファクトリを返す
func NewCharacterNamingState() es.State[w.World] {
	return &CharacterNamingState{}
}

func (st CharacterNamingState) String() string {
	return "CharacterNaming"
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
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err == nil {
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			initialName = nameComp.(*gc.Name).Name
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

	// TextInput から現在の値を同期
	if textInput, ok := hooks.GetRef[*widget.TextInput](st.mount.Store(), "textInput"); ok && textInput != nil {
		currentText := textInput.GetText()
		if currentText != props.CurrentName {
			st.mount.SetProps(namingProps{
				CurrentName:  currentText,
				ErrorMessage: props.ErrorMessage,
			})
		}
	}

	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	// dirty判定とUI再構築
	if st.mount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *CharacterNamingState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(consts.BlackColor)
	if st.widget != nil {
		st.widget.Draw(screen)
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
		return es.Transition[w.World]{}, fmt.Errorf("characterNaming: 未対応のアクション: %s", action)
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
			ErrorMessage: "名前は1〜10文字で入力してください",
		})
		_, startTimer, _ := hooks.UseTimer(st.mount.Store(), "errorTimer", errorDisplayTime)
		startTimer()
		return es.Transition[w.World]{Type: es.TransNone}
	}

	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err == nil {
		// 既存プレイヤーの名前を変更した
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			nameComp.(*gc.Name).Name = name
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
	_, err := worldhelper.GetPlayerEntity(world)
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

func (st *CharacterNamingState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	centerContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(20),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	titleLabel := widget.NewText(
		widget.TextOpts.Text("名前", &res.Text.TitleFontFace, consts.PrimaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	// テキスト入力を作成
	textInput := hooks.UseRef(st.mount.Store(), "textInput", func() *widget.TextInput {
		ti := widget.NewTextInput(
			widget.TextInputOpts.WidgetOpts(
				widget.WidgetOpts.LayoutData(widget.RowLayoutData{
					Position: widget.RowLayoutPositionCenter,
					Stretch:  true,
				}),
				widget.WidgetOpts.MinSize(300, 0),
			),
			widget.TextInputOpts.Image(res.TextInput.Image),
			widget.TextInputOpts.Face(&res.TextInput.Face),
			widget.TextInputOpts.Color(res.TextInput.Color),
			widget.TextInputOpts.Padding(&res.TextInput.Padding),
			widget.TextInputOpts.Placeholder("名前"),
		)
		ti.SetText(props.CurrentName)
		ti.Focus(true)
		return ti
	})

	// エラーメッセージ
	errorText := widget.NewText(
		widget.TextOpts.Text(props.ErrorMessage, &res.Text.SmallFace, consts.DangerColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart,
			}),
		),
	)

	// 操作ヒント
	hintLabel := widget.NewText(
		widget.TextOpts.Text(consts.IconKeyEnter+" 決定 / "+consts.IconKeyEsc+" 戻る", &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	centerContainer.AddChild(titleLabel)
	centerContainer.AddChild(textInput)
	centerContainer.AddChild(errorText)
	centerContainer.AddChild(hintLabel)

	rootContainer.AddChild(centerContainer)

	return &ebitenui.UI{Container: rootContainer}
}
