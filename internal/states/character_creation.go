package states

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// CharacterCreationState はキャラクター作成画面のステート
type CharacterCreationState struct {
	es.BaseState[w.World]
	ui        *ebitenui.UI
	textInput *widget.TextInput
}

func (st CharacterCreationState) String() string {
	return "CharacterCreation"
}

// State interface ================

var _ es.State[w.World] = &CharacterCreationState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CharacterCreationState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CharacterCreationState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *CharacterCreationState) OnStart(world w.World) error {
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CharacterCreationState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CharacterCreationState) Update(world w.World) (es.Transition[w.World], error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		st.confirmName(world)
		return st.ConsumeTransition(), nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		st.cancel(world)
		return st.ConsumeTransition(), nil
	}

	if st.ui != nil {
		st.ui.Update()
	}

	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *CharacterCreationState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(consts.BlackColor)

	if st.ui != nil {
		st.ui.Draw(screen)
	}
	return nil
}

// ================

// initUI はUIを初期化する
func (st *CharacterCreationState) initUI(world w.World) *ebitenui.UI {
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
		widget.TextOpts.Text("名前を入力してください", &res.Text.TitleFontFace, consts.PrimaryColor),
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

	// 既存プレイヤーの名前を初期値として設定
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err == nil {
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			st.textInput.SetText(nameComp.(*gc.Name).Name)
		}
	}

	st.textInput.Focus(true)

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
	centerContainer.AddChild(hintLabel)

	rootContainer.AddChild(centerContainer)

	return &ebitenui.UI{Container: rootContainer}
}

// confirmName は名前を確定する
func (st *CharacterCreationState) confirmName(world w.World) {
	name := st.textInput.GetText()
	if name == "" {
		name = "セレスティン"
	}

	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err == nil {
		// 既存プレイヤーの名前を変更
		if nameComp := world.Components.Name.Get(playerEntity); nameComp != nil {
			nameComp.(*gc.Name).Name = name
		}
		st.SetTransition(es.Transition[w.World]{
			Type: es.TransPop,
		})
	} else {
		// プレイヤーを生成して新規ゲーム開始
		_, _ = worldhelper.SpawnPlayer(world, 5, 5, name)
		st.SetTransition(es.Transition[w.World]{
			Type:          es.TransReplace,
			NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
		})
	}
}

// cancel はキャンセルする
func (st *CharacterCreationState) cancel(world w.World) {
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
