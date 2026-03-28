package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

const autoSellItemsPerPage = 20

// AutoSellState はラン終了時の精算画面。
// OnStartでプレビューを生成し、Enterで売却を実行してから町に遷移する。
type AutoSellState struct {
	es.BaseState[w.World]
	mount   *hooks.Mount[autoSellProps]
	widget  *ebitenui.UI
	preview worldhelper.AutoSellResult
}

type autoSellProps struct {
	Items []worldhelper.SoldItem
	Total int
}

// NewAutoSellState は帰還報告画面のStateを作成するファクトリー関数
func NewAutoSellState() es.StateFactory[w.World] {
	return func() es.State[w.World] {
		return &AutoSellState{}
	}
}

func (st AutoSellState) String() string {
	return "AutoSell"
}

// State interface ================

var _ es.State[w.World] = &AutoSellState{}
var _ es.ActionHandler[w.World] = &AutoSellState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *AutoSellState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *AutoSellState) OnResume(_ w.World) error { return nil }

// OnStart はプレビューを生成する。売却はまだ実行しない。
func (st *AutoSellState) OnStart(world w.World) error {
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return fmt.Errorf("プレイヤーの取得に失敗: %w", err)
	}

	result, err := worldhelper.PreviewEndRun(world, playerEntity)
	if err != nil {
		return fmt.Errorf("プレビュー生成に失敗: %w", err)
	}
	st.preview = result
	st.mount = hooks.NewMount[autoSellProps]()

	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *AutoSellState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *AutoSellState) Update(world w.World) (es.Transition[w.World], error) {
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.mount.Dispatch(action)
	}

	props := autoSellProps{
		Items: st.preview.Items,
		Total: st.preview.Total,
	}
	st.mount.SetProps(props)

	hooks.UseTabMenu(st.mount.Store(), "autosell", hooks.TabMenuConfig{
		TabCount:     1,
		ItemCounts:   []int{len(props.Items)},
		ItemsPerPage: autoSellItemsPerPage,
	})

	if st.mount.Update() || st.widget == nil {
		st.widget = st.buildUI(world)
	}
	st.widget.Update()

	return st.ConsumeTransition(), nil
}

// Draw はスクリーンに描画する
func (st *AutoSellState) Draw(_ w.World, screen *ebiten.Image) error {
	screen.Fill(consts.BlackColor)
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *AutoSellState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *AutoSellState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuSelect, inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		// 売却を実行してから町に遷移する
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			return es.Transition[w.World]{}, fmt.Errorf("プレイヤーの取得に失敗: %w", err)
		}
		if err := worldhelper.ExecuteEndRun(world, playerEntity, st.preview.Total); err != nil {
			return es.Transition[w.World]{}, fmt.Errorf("売却実行に失敗: %w", err)
		}
		return es.Transition[w.World]{
			Type:          es.TransReplace,
			NewStateFuncs: []es.StateFactory[w.World]{NewTownState()},
		}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight:
		// Dispatchで処理される
	default:
		// 他のアクションは無視する
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// buildUI
// ================

func (st *AutoSellState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.mount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.mount, "autosell")
	itemIndex := menuState.ItemIndex

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// row1: タイトル | 空 | 空
	root.AddChild(styled.NewTitleText("帰還報告", res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	// row2: アイテムリスト | 空 | スペック
	root.AddChild(st.buildItemContainer(props, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(st.buildSpecContainer(world, props, itemIndex, res))

	// row3: フッター | 空 | 空
	root.AddChild(st.buildFooterContainer(props, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	return &ebitenui.UI{Container: root}
}

func (st *AutoSellState) buildItemContainer(props autoSellProps, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	if len(props.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(収穫なし)", res))
		return container
	}

	columnWidths := []int{20, 120, 40, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignRight}

	pg := pagination.New(itemIndex, len(props.Items), autoSellItemsPerPage)

	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " "
	}
	container.AddChild(styled.NewPageIndicator(pageText, res))

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(props.Items, pg) {
		isSelected := pg.IsSelectedInPage(entry.Index)
		countStr := ""
		if entry.Item.Count > 1 {
			countStr = fmt.Sprintf("%d", entry.Item.Count)
		}
		styled.NewTableRow(table, columnWidths, []string{"", entry.Item.Name, countStr, worldhelper.FormatCurrency(entry.Item.Price)}, aligns, &isSelected, res)
	}
	container.AddChild(table)

	return container
}

func (st *AutoSellState) buildSpecContainer(world w.World, props autoSellProps, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	if itemIndex >= len(props.Items) {
		return container
	}

	item := props.Items[itemIndex]
	if item.Entity != 0 {
		views.UpdateSpec(world, container, item.Entity)
	}

	return container
}

func (st *AutoSellState) buildFooterContainer(props autoSellProps, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()

	totalText := fmt.Sprintf("合計  %s", worldhelper.FormatCurrency(props.Total))
	container.AddChild(widget.NewText(
		widget.TextOpts.Text(totalText, &res.Text.BodyFace, consts.PrimaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionEnd,
			}),
		),
	))

	hintText := consts.IconArrowUp + consts.IconArrowDown + " 選択 / " + consts.IconKeyEnter + " 決定"
	container.AddChild(widget.NewText(
		widget.TextOpts.Text(hintText, &res.Text.SmallFace, consts.SecondaryColor),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{
				Position: widget.RowLayoutPositionCenter,
			}),
		),
	))

	return container
}
