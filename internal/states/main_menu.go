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
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// MainMenuState はメインメニューのゲームステート
type MainMenuState struct {
	es.BaseState[w.World]
	screen *menurt.Screen[MainMenuProps]
}

// State interface ================

var _ es.State[w.World] = &MainMenuState{}

// OnStart はステート開始時の処理を行う
func (st *MainMenuState) OnStart(world w.World) error {
	// ワールドをクリアする。前のゲーム状態を削除する
	var clearEntities []ecs.Entity
	clearQuery := ecs.NewUnsafeFilter(world.ECS).Query()
	for clearQuery.Next() {
		clearEntities = append(clearEntities, clearQuery.Entity())
	}
	for _, e := range clearEntities {
		world.ECS.RemoveEntity(e)
	}
	// シングルトンエンティティを再構築する
	world.InitSingleton()

	st.screen = menurt.NewScreen[MainMenuProps](st)
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

// MainMenuProps はメインメニューのProps
type MainMenuProps struct {
	Items []mainMenuItem
}

// mainMenuItem はメインメニューの項目
type mainMenuItem struct {
	Label      string
	Transition es.Transition[w.World]
}

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *MainMenuState) Fetch(world w.World) MainMenuProps {
	var startFuncs []es.StateFactory[w.World]
	if world.Config.SkipOpening {
		startFuncs = []es.StateFactory[w.World]{NewCharacterNamingState}
	} else {
		startFuncs = []es.StateFactory[w.World]{NewCharacterNamingState, NewOpeningState}
	}

	t := func(msgid string) string { return query.T(world, msgid) }
	return MainMenuProps{
		Items: []mainMenuItem{
			{Label: t("Start"), Transition: es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: startFuncs}},
			{Label: t("Demo"), Transition: es.Transition[w.World]{Type: es.TransReplace, NewStateFuncs: []es.StateFactory[w.World]{NewDemoStartState}}},
			{Label: t("Load"), Transition: es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewLoadMenuState}}},
			{Label: t("Settings"), Transition: es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewSettingsMenuState}}},
			{Label: t("Quit"), Transition: es.Transition[w.World]{Type: es.TransQuit}},
		},
	}
}

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *MainMenuState) Menu(props MainMenuProps) menurt.MenuConfig {
	return menurt.MenuConfig{Key: "menu", TabCount: 1, ItemCounts: []int{len(props.Items)}}
}

func (st *MainMenuState) handleSelection() (es.Transition[w.World], error) {
	props := st.screen.Props()
	itemIndex := st.screen.Selection().ItemIndex
	if itemIndex >= len(props.Items) {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}
	return props.Items[itemIndex].Transition, nil
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *MainMenuState) View(_ w.World, props MainMenuProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
	itemIndex := cursor.ItemIndex

	rootContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	menuContainer := styled.NewVerticalContainer(
		// メインメニューはベース state すなわちスタック[0]で、メニュー層でなく target に直接描く。
		// 層アルファが効かないので不透明の ImageOpaque にせず、独自背景を透かす Image のままにする。
		// 将来スタック[1..]へ移して層に載せるなら、下メニューを透けさせないため ImageOpaque へ替える。
		widget.ContainerOpts.BackgroundImage(res.Panel.Image),
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
		rows[i] = menuRow{Cells: []string{item.Label}}
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
