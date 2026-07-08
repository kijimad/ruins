package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
)

// SettingsMenuState はグローバル設定を変更するゲームステート。
// メインメニューから push される。現状は設定項目が無く、将来の設定（音量など）を追加する土台。
type SettingsMenuState struct {
	es.BaseState[w.World]
	menuMount *hooks.Mount[settingsMenuProps]
	widget    *ebitenui.UI
}

func (st SettingsMenuState) String() string {
	return "SettingsMenu"
}

// State interface ================

var _ es.State[w.World] = &SettingsMenuState{}
var _ es.ActionHandler[w.World] = &SettingsMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *SettingsMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *SettingsMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う。メインメニューの上に重なるためワールドは操作しない
func (st *SettingsMenuState) OnStart(_ w.World) error {
	st.menuMount = hooks.NewMount[settingsMenuProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *SettingsMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *SettingsMenuState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
	}

	st.menuMount.SetProps(st.fetchProps(world))
	props := st.menuMount.GetProps()
	hooks.UseTabMenu(st.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	if st.menuMount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *SettingsMenuState) Draw(world w.World, screen *ebiten.Image) error {
	bgImage, err := loadBackgroundImage(world, "title1")
	if err != nil {
		return err
	}
	screen.DrawImage(bgImage, nil)

	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *SettingsMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *SettingsMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(), nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("settingsMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// settingsMenuProps は設定メニューのProps
type settingsMenuProps struct {
	Items []settingsMenuItem
}

// settingsMenuItem は設定メニューの1項目
type settingsMenuItem struct {
	Label      string
	Transition es.Transition[w.World]
}

func (st *SettingsMenuState) fetchProps(_ w.World) settingsMenuProps {
	return settingsMenuProps{
		Items: []settingsMenuItem{
			{Label: "戻る", Transition: es.Transition[w.World]{Type: es.TransPop}},
		},
	}
}

func (st *SettingsMenuState) handleSelection() es.Transition[w.World] {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "menu")
	if !ok || menuState.ItemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}
	}
	return props.Items[menuState.ItemIndex].Transition
}

// ================
// buildUI
// ================

func (st *SettingsMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "menu")
	itemIndex := menuState.ItemIndex

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	menuContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
			widget.WidgetOpts.MinSize(300, 0),
		),
	)

	titleText := widget.NewText(
		widget.TextOpts.Text("設定", &res.Text.BodyFace, theme.TextPrimary),
	)
	menuContainer.AddChild(titleText)

	for i, item := range props.Items {
		isSelected := i == itemIndex
		itemWidget := styled.NewListItemText(item.Label, theme.TextSecondary, isSelected, res)
		menuContainer.AddChild(itemWidget)
	}

	rootContainer.AddChild(menuContainer)
	return &ebitenui.UI{Container: rootContainer}
}
