package states

import (
	"fmt"
	"image"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
)

// MainMenuState はメインメニューのゲームステート
type MainMenuState struct {
	es.BaseState[w.World]
	menuMount *hooks.Mount[mainMenuProps]
	widget    *ebitenui.UI
}

func (st MainMenuState) String() string {
	return "MainMenu"
}

// State interface ================

var _ es.State[w.World] = &MainMenuState{}
var _ es.ActionHandler[w.World] = &MainMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *MainMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *MainMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステート開始時の処理を行う
func (st *MainMenuState) OnStart(world w.World) error {
	// ワールドをクリアする。前のゲーム状態を削除する
	world.Manager.DeleteAllEntities()

	st.menuMount = hooks.NewMount[mainMenuProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *MainMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *MainMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
	}

	// Props更新
	st.menuMount.SetProps(st.fetchProps())
	props := st.menuMount.GetProps()
	hooks.UseTabMenu(st.menuMount.Store(), "menu", hooks.TabMenuConfig{
		TabCount:   1,
		ItemCounts: []int{len(props.Items)},
	})

	// dirty判定とUI再構築
	if st.menuMount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *MainMenuState) Draw(world w.World, screen *ebiten.Image) error {
	// 背景画像を描画
	bgSheet := (*world.Resources.SpriteSheets)["bg"]
	bgSprite := bgSheet.Sprites["title1"]
	rect := image.Rect(
		bgSprite.X,
		bgSprite.Y,
		bgSprite.X+bgSprite.Width,
		bgSprite.Y+bgSprite.Height,
	)
	bgImage := bgSheet.Texture.Image.SubImage(rect).(*ebiten.Image)
	screen.DrawImage(bgImage, nil)

	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *MainMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *MainMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransQuit}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection()
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("mainMenu: 未対応のアクション: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// mainMenuProps はメインメニューのProps
type mainMenuProps struct {
	Items []mainMenuItem
}

// mainMenuItem はメインメニューの項目
type mainMenuItem struct {
	Label      string
	Transition es.Transition[w.World]
}

func (st *MainMenuState) fetchProps() mainMenuProps {
	return mainMenuProps{
		Items: []mainMenuItem{
			{Label: "開始", Transition: es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{NewCharacterNamingState}}},
			{Label: "読込", Transition: es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewLoadMenuState}}},
			{Label: "終了", Transition: es.Transition[w.World]{Type: es.TransQuit}},
		},
	}
}

func (st *MainMenuState) handleSelection() (es.Transition[w.World], error) {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "menu")
	if !ok {
		return es.Transition[w.World]{}, fmt.Errorf("menuの取得に失敗")
	}
	itemIndex := menuState.ItemIndex

	if itemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	return props.Items[itemIndex].Transition, nil
}

// ================
// buildUI
// ================

func (st *MainMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "menu")
	itemIndex := menuState.ItemIndex

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// メニューコンテナを構築
	menuContainer := styled.NewVerticalContainer()
	for i, item := range props.Items {
		isSelected := i == itemIndex
		itemWidget := styled.NewListItemText(item.Label, consts.TextColor, isSelected, res)
		menuContainer.AddChild(itemWidget)
	}

	// バージョン表示テキストを作成
	versionInfo := []string{}
	if consts.AppVersion != "v0.0.0" {
		versionInfo = append(versionInfo, consts.AppVersion)
	}
	if consts.AppCommit != "0000000" {
		versionInfo = append(versionInfo, consts.AppCommit)
	}
	if consts.AppDate != "0000-00-00" {
		versionInfo = append(versionInfo, consts.AppDate)
	}
	versionText := widget.NewText(
		widget.TextOpts.Text(strings.Join(versionInfo, "\n"), &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding: &widget.Insets{
					Right:  20,
					Bottom: 20,
				},
			}),
		),
	)

	// ラッパーコンテナを作成
	wrapperContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				Padding: &widget.Insets{
					Top: 400,
				},
			}),
		),
	)

	wrapperContainer.AddChild(menuContainer)
	rootContainer.AddChild(wrapperContainer)
	rootContainer.AddChild(versionText)

	return &ebitenui.UI{Container: rootContainer}
}
