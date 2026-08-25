package states

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// MainMenuState はメインメニューのゲームステート
type MainMenuState struct {
	es.BaseState[w.World]
	screen *menuloop.Screen[MainMenuProps]
}

// State interface ================

var _ es.State[w.World] = &MainMenuState{}

// OnStart はステート開始時の処理を行う。world には触れない。
// 前のゲームの後片付けは新しいゲームを始める側が world.ResetForNewGame で行う
func (st *MainMenuState) OnStart(_ w.World) error {
	st.screen = menuloop.NewScreen[MainMenuProps](st)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *MainMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw はスクリーンに描画する
func (st *MainMenuState) Draw(world w.World, screen *ebiten.Image) error {
	// 背景画像を描画
	bgImage, err := loadBackgroundImage(world, "title1")
	if err != nil {
		return err
	}
	screen.DrawImage(bgImage, nil)

	st.screen.Draw(screen)
	return nil
}

// DoAction はActionを実行する
func (st *MainMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransQuit}, nil
	case inputmapper.ActionMenuSelect:
		return st.handleSelection(world)
	default:
		return es.Transition[w.World]{}, fmt.Errorf("mainMenu: unsupported action: %s", action)
	}
}

// ================
// Props
// ================

// MainMenuProps はメインメニューのProps
type MainMenuProps struct {
	Items []mainMenuItem
}

// mainMenuItem はメインメニューの項目。ResetsWorld が真の項目は遷移の前に前のゲームの
// 全実体を消す。新しいゲームを始める Start・Demo が立てる。ロードは save 側が担うので立てない
type mainMenuItem struct {
	Label       string
	Transition  es.Transition[w.World]
	ResetsWorld bool
}

// Fetch は世界から表示 props を構築する。menuloop.Model の Model 部にあたる
func (st *MainMenuState) Fetch(world w.World) (MainMenuProps, error) {
	var startFuncs []es.StateFactory[w.World]
	if world.Resources.Config.SkipOpening {
		startFuncs = []es.StateFactory[w.World]{NewCharacterNamingState}
	} else {
		startFuncs = []es.StateFactory[w.World]{NewCharacterNamingState, NewOpeningState}
	}

	t := func(msgid string) string { return query.T(world, msgid) }
	return MainMenuProps{
		Items: []mainMenuItem{
			{Label: t("Start"), Transition: es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: startFuncs}, ResetsWorld: true},
			{Label: t("Demo"), Transition: es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{NewDemoStartState}}, ResetsWorld: true},
			{Label: t("Load"), Transition: es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewLoadMenuState}}},
			{Label: t("Settings"), Transition: es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewSettingsMenuState}}},
			{Label: t("Quit"), Transition: es.Transition[w.World]{Type: es.TransQuit}},
		},
	}, nil
}

// Menu は一覧の構成を返す。menuloop.Model の Menu 部にあたる
func (st *MainMenuState) Menu(props MainMenuProps) menuloop.MenuConfig {
	return menuloop.MenuConfig{Key: "menu", TabCount: 1, ItemCounts: []int{len(props.Items)}}
}

func (st *MainMenuState) handleSelection(world w.World) (es.Transition[w.World], error) {
	props := st.screen.Props()
	itemIndex := st.screen.Selection().ItemIndex
	if itemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	item := props.Items[itemIndex]
	if item.ResetsWorld {
		world.ResetForNewGame()
	}
	return item.Transition, nil
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menuloop.Model の View 部にあたる
func (st *MainMenuState) View(_ w.World, props MainMenuProps, cursor menuloop.Selection, res resources.UIResources) *ebitenui.UI {
	itemIndex := cursor.ItemIndex

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// タイトルは選択肢を背景へ浮かせる。パネル背景を付けず透明にし、下部スクリムで可読性を担保する。
	menuContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Stretch: true,
			}),
			widget.WidgetOpts.MinSize(200, 0),
		),
	)
	// 行は共通ヘルパで組み、縦幅・行間・幅を他メニューと揃える
	rows := make([]menuRow, len(props.Items))
	for i, item := range props.Items {
		rows[i] = menuRow{Cells: styled.TextCells(item.Label)}
	}
	menuContainer.AddChild(renderMenuList(itemIndex, rows, []int{menuRowWidth}, []styled.TextAlign{styled.AlignLeft}, menuListOpts{Spaced: true}, res))

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
		widget.TextOpts.Text(strings.Join(versionInfo, "\n"), &res.Text.SmallFace, theme.TextAccent),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding: &widget.Insets{
					Right:  theme.Space6,
					Bottom: theme.Space6,
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
			// シネマ配置。ロゴは背景左上、メニューは左下へ左寄せで置き、右のキューブを主役に残す
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionEnd,
				Padding: &widget.Insets{
					Left:   64,
					Bottom: 72,
				},
			}),
		),
	)

	wrapperContainer.AddChild(menuContainer)
	rootContainer.AddChild(wrapperContainer)
	rootContainer.AddChild(versionText)

	return &ebitenui.UI{Container: rootContainer}
}
