package states

import (
	"time"
	"unicode/utf8"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
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
	ui             *ebitenui.UI
	textInput      *widget.TextInput
	errorText      *widget.Text
	errorClearTime time.Time
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

// OnPause はステートが一時停止される際に呼ばれる
func (st *CharacterNamingState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CharacterNamingState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *CharacterNamingState) OnStart(world w.World) error {
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CharacterNamingState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CharacterNamingState) Update(world w.World) (es.Transition[w.World], error) {
	// エラーメッセージの自動クリア
	if st.errorText != nil && st.errorText.Label != "" && time.Now().After(st.errorClearTime) {
		st.errorText.Label = ""
	}

	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsEnterJustPressedOnce() {
		st.confirmName(world)
		return st.ConsumeTransition(), nil
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		st.cancel(world)
		return st.ConsumeTransition(), nil
	}

	if st.ui != nil {
		st.ui.Update()
	}

	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *CharacterNamingState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(consts.BlackColor)

	if st.ui != nil {
		st.ui.Draw(screen)
	}
	return nil
}

// ================

// initUI はUIを初期化する
func (st *CharacterNamingState) initUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources

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

	st.textInput = widget.NewTextInput(
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

	// 既存プレイヤーの名前を初期値として設定する。プレイヤーが存在しない場合は空欄のまま
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err == nil {
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			st.textInput.SetText(nameComp.(*gc.Name).Name)
		}
	}

	st.textInput.Focus(true)

	// エラーメッセージ
	st.errorText = widget.NewText(
		widget.TextOpts.Text("", &res.Text.SmallFace, consts.DangerColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionStart,
			}),
		),
	)

	// 操作ヒント
	hintLabel := widget.NewText(
		widget.TextOpts.Text("Enter: 決定 / Esc: 戻る", &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	)

	centerContainer.AddChild(titleLabel)
	centerContainer.AddChild(st.textInput)
	centerContainer.AddChild(st.errorText)
	centerContainer.AddChild(hintLabel)

	rootContainer.AddChild(centerContainer)

	return &ebitenui.UI{Container: rootContainer}
}

// confirmName は名前を確定する
func (st *CharacterNamingState) confirmName(world w.World) {
	name := st.textInput.GetText()

	nameLen := utf8.RuneCountInString(name)
	if nameLen < nameMinLength || nameLen > nameMaxLength {
		st.errorText.Label = "名前は1〜10文字で入力してください"
		st.errorClearTime = time.Now().Add(errorDisplayTime)
		return
	}

	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err == nil {
		// 既存プレイヤーの名前を変更した
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			nameComp.(*gc.Name).Name = name
		}
		st.SetTransition(es.Transition[w.World]{
			Type: es.TransPop,
		})
	} else {
		// 職業選択画面へ遷移
		st.SetTransition(es.Transition[w.World]{
			Type:          es.TransPush,
			NewStateFuncs: []es.StateFactory[w.World]{NewCharacterJobState(name)},
		})
	}
}

// cancel はキャンセルする
func (st *CharacterNamingState) cancel(world w.World) {
	_, err := worldhelper.GetPlayerEntity(world)
	if err == nil {
		st.SetTransition(es.Transition[w.World]{
			Type: es.TransPop,
		})
	} else {
		st.SetTransition(es.Transition[w.World]{
			Type:          es.TransReplace,
			NewStateFuncs: []es.StateFactory[w.World]{NewMainMenuState},
		})
	}
}
