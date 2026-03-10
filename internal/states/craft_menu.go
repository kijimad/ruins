package states

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
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
	ecs "github.com/x-hgg-x/goecs/v2"
)

const craftItemsPerPage = 20

// craftSubState はクラフトメニュー内のサブステート
type craftSubState int

const (
	craftSubStateMenu   craftSubState = iota // メニュー選択
	craftSubStateWindow                      // アクションウィンドウ
	craftSubStateResult                      // 結果表示
)

// CraftMenuState はクラフトメニューのゲームステート
type CraftMenuState struct {
	es.BaseState[w.World]
	subState    craftSubState
	menuMount   *hooks.Mount[craftProps]
	windowMount *hooks.Mount[craftWindowProps]
	resultMount *hooks.Mount[craftResultProps]
	widget      *ebitenui.UI
}

func (st CraftMenuState) String() string {
	return "CraftMenu"
}

// State interface ================

var _ es.State[w.World] = &CraftMenuState{}
var _ es.ActionHandler[w.World] = &CraftMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *CraftMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *CraftMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *CraftMenuState) OnStart(_ w.World) error {
	st.subState = craftSubStateMenu
	st.menuMount = hooks.NewMount[craftProps]()
	st.windowMount = hooks.NewMount[craftWindowProps]()
	st.resultMount = hooks.NewMount[craftResultProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *CraftMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *CraftMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		switch st.subState {
		case craftSubStateMenu:
			st.menuMount.Dispatch(action)
		case craftSubStateWindow:
			st.windowMount.Dispatch(action)
		case craftSubStateResult:
			st.resultMount.Dispatch(action)
		}
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	// UseTabMenuでreducerを登録・更新
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(st.menuMount.Store(), "craft", hooks.TabMenuConfig{
		TabCount:     len(props.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: craftItemsPerPage,
	})

	// ウィンドウ用のステート
	st.setupWindowState(world)
	st.setupResultState()

	// 短絡評価を避け、全てのmountのdirtyフラグをクリアする
	menuDirty := st.menuMount.Update()
	windowDirty := st.windowMount.Update()
	resultDirty := st.resultMount.Update()
	if menuDirty || windowDirty || resultDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *CraftMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *CraftMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	switch st.subState {
	case craftSubStateMenu:
		return HandleMenuInput()
	case craftSubStateWindow, craftSubStateResult:
		return HandleWindowInput()
	}
	return "", false
}

// DoAction はActionを実行する
func (st *CraftMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.subState {
	case craftSubStateResult:
		switch action {
		case inputmapper.ActionWindowConfirm, inputmapper.ActionWindowCancel:
			st.subState = craftSubStateMenu
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("craftSubStateResult: 未対応のアクション: %s", action)
		}

	case craftSubStateWindow:
		switch action {
		case inputmapper.ActionWindowConfirm:
			if err := st.executeActionItem(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionWindowCancel:
			st.subState = craftSubStateMenu
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("craftSubStateWindow: 未対応のアクション: %s", action)
		}

	case craftSubStateMenu:
		switch action {
		case inputmapper.ActionOpenDebugMenu:
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			return es.Transition[w.World]{Type: es.TransPop}, nil
		case inputmapper.ActionMenuSelect:
			if err := st.handleItemSelection(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("craftSubStateMenu: 未対応のアクション: %s", action)
		}
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type craftProps struct {
	Tabs []craftTabData
}

type craftTabData struct {
	ID    string
	Label string
	Items []craftItemData
}

type craftItemData struct {
	RecipeName string
	CanCraft   bool
}

type craftWindowProps struct {
	RecipeName string
}

type craftResultProps struct {
	ResultEntity ecs.Entity
}

func (st *CraftMenuState) fetchProps(world w.World) craftProps {
	return craftProps{
		Tabs: st.createTabs(world),
	}
}

func (st *CraftMenuState) createTabs(world w.World) []craftTabData {
	return []craftTabData{
		{ID: "consumables", Label: "道具", Items: st.createMenuItems(world, st.queryMenuConsumable(world))},
		{ID: "weapons", Label: "武器", Items: st.createMenuItems(world, st.queryMenuWeapon(world))},
		{ID: "wearables", Label: "装備", Items: st.createMenuItems(world, st.queryMenuWearable(world))},
	}
}

func (st *CraftMenuState) createMenuItems(world w.World, recipeNames []string) []craftItemData {
	items := make([]craftItemData, len(recipeNames))

	for i, recipeName := range recipeNames {
		canCraft, _ := worldhelper.CanCraft(world, recipeName)
		items[i] = craftItemData{
			RecipeName: recipeName,
			CanCraft:   canCraft,
		}
	}

	return items
}

func (st *CraftMenuState) queryMenuConsumable(world w.World) []string {
	rawMaster := world.Resources.RawMaster
	var items []string

	for recipeName := range rawMaster.RecipeIndex {
		spec, err := rawMaster.NewRecipeSpec(recipeName)
		if err != nil {
			continue
		}
		if spec.Consumable != nil {
			items = append(items, recipeName)
		}
	}

	sort.Strings(items)
	return items
}

func (st *CraftMenuState) queryMenuWeapon(world w.World) []string {
	rawMaster := world.Resources.RawMaster
	var items []string

	for recipeName := range rawMaster.RecipeIndex {
		spec, err := rawMaster.NewRecipeSpec(recipeName)
		if err != nil {
			continue
		}
		if spec.Weapon != nil {
			items = append(items, recipeName)
		}
	}

	sort.Strings(items)
	return items
}

func (st *CraftMenuState) queryMenuWearable(world w.World) []string {
	rawMaster := world.Resources.RawMaster
	var items []string

	for recipeName := range rawMaster.RecipeIndex {
		spec, err := rawMaster.NewRecipeSpec(recipeName)
		if err != nil {
			continue
		}
		if spec.Wearable != nil {
			items = append(items, recipeName)
		}
	}

	sort.Strings(items)
	return items
}

// ================
// Action Window
// ================

func (st *CraftMenuState) setupWindowState(world w.World) {
	windowProps := st.windowMount.GetProps()
	actionItems := st.getActionItems(world, windowProps.RecipeName)

	hooks.UseState(st.windowMount.Store(), "craft_window_index", 0, func(v int, action inputmapper.ActionID) int {
		switch action {
		case inputmapper.ActionWindowUp:
			if v > 0 {
				return v - 1
			}
			return len(actionItems) - 1
		case inputmapper.ActionWindowDown:
			if v < len(actionItems)-1 {
				return v + 1
			}
			return 0
		default:
			return v
		}
	})
}

func (st *CraftMenuState) getActionItems(world w.World, recipeName string) []string {
	if recipeName == "" {
		return []string{TextClose}
	}

	actionItems := []string{}

	if canCraft, _ := worldhelper.CanCraft(world, recipeName); canCraft {
		actionItems = append(actionItems, "合成する")
	}
	actionItems = append(actionItems, TextClose)

	return actionItems
}

func (st *CraftMenuState) handleItemSelection(_ w.World) error {
	props := st.menuMount.GetProps()
	tabIndex, ok := hooks.GetState[int](st.menuMount, "craft_tabIndex")
	if !ok {
		return fmt.Errorf("craft_tabIndexの取得に失敗")
	}
	itemIndex, ok := hooks.GetState[int](st.menuMount, "craft_itemIndex")
	if !ok {
		return fmt.Errorf("craft_itemIndexの取得に失敗")
	}

	if tabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[tabIndex]
	if itemIndex >= len(tab.Items) {
		return nil
	}
	item := tab.Items[itemIndex]

	st.subState = craftSubStateWindow
	st.windowMount = hooks.NewMount[craftWindowProps]()
	st.windowMount.SetProps(craftWindowProps{
		RecipeName: item.RecipeName,
	})
	return nil
}

func (st *CraftMenuState) executeActionItem(world w.World) error {
	windowProps := st.windowMount.GetProps()
	actionIndex, ok := hooks.GetState[int](st.windowMount, "craft_window_index")
	if !ok {
		return fmt.Errorf("craft_window_indexの取得に失敗")
	}
	actionItems := st.getActionItems(world, windowProps.RecipeName)

	if actionIndex >= len(actionItems) {
		return nil
	}

	selectedAction := actionItems[actionIndex]

	switch selectedAction {
	case "合成する":
		resultEntity, err := worldhelper.Craft(world, windowProps.RecipeName)
		if err != nil {
			return err
		}
		st.subState = craftSubStateMenu
		st.subState = craftSubStateResult
		st.resultMount = hooks.NewMount[craftResultProps]()
		st.resultMount.SetProps(craftResultProps{
			ResultEntity: *resultEntity,
		})
	case TextClose:
		st.subState = craftSubStateMenu
	}
	return nil
}

// ================
// Result Window
// ================

func (st *CraftMenuState) setupResultState() {
	resultItems := []string{TextClose}

	hooks.UseState(st.resultMount.Store(), "craft_result_index", 0, func(v int, action inputmapper.ActionID) int {
		switch action {
		case inputmapper.ActionWindowUp:
			if v > 0 {
				return v - 1
			}
			return len(resultItems) - 1
		case inputmapper.ActionWindowDown:
			if v < len(resultItems)-1 {
				return v + 1
			}
			return 0
		default:
			return v
		}
	})
}

// ================
// buildUI
// ================

func (st *CraftMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	tabIndex, _ := hooks.GetState[int](st.menuMount, "craft_tabIndex")
	itemIndex, _ := hooks.GetState[int](st.menuMount, "craft_itemIndex")

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 1行目: タイトル、カテゴリ、空
	root.AddChild(styled.NewTitleText("合成", res))
	root.AddChild(st.buildCategoryContainer(props.Tabs, tabIndex, res))
	root.AddChild(widget.NewContainer())

	// 2行目: アイテム一覧、空、性能+レシピ表示
	root.AddChild(st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(st.buildDetailContainer(world, props, tabIndex, itemIndex, res))

	// 3行目: 説明文
	root.AddChild(st.buildDescContainer(world, props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	eui := &ebitenui.UI{Container: root}

	// ウィンドウを追加
	switch st.subState {
	case craftSubStateMenu:
		// ウィンドウなし
	case craftSubStateWindow:
		actionWindow := st.buildActionWindow(world, st.windowMount.GetProps())
		eui.AddWindow(actionWindow)
	case craftSubStateResult:
		resultWindow := st.buildResultWindow(world, st.resultMount.GetProps())
		eui.AddWindow(resultWindow)
	}

	return eui
}

func (st *CraftMenuState) buildCategoryContainer(tabs []craftTabData, tabIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	for i, tab := range tabs {
		isSelected := i == tabIndex
		color := consts.ForegroundColor
		if isSelected {
			color = consts.TextColor
		}
		container.AddChild(styled.NewListItemText(tab.Label, color, isSelected, res))
	}
	return container
}

func (st *CraftMenuState) buildItemContainer(tabs []craftTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(tabs) {
		return container
	}

	currentTab := tabs[tabIndex]
	pg := pagination.New(itemIndex, len(currentTab.Items), craftItemsPerPage)

	// ページインジケーター
	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " "
	}
	container.AddChild(styled.NewPageIndicator(pageText, res))

	columnWidths := []int{20, 180}

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, columnWidths, []string{"", entry.Item.RecipeName}, nil, &isSelected, res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(レシピなし)", res))
	}

	return container
}

func (st *CraftMenuState) buildDetailContainer(world w.World, props craftProps, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	specContainer := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	recipeContainer := styled.NewVerticalContainer()

	if tabIndex >= len(props.Tabs) {
		return styled.NewVSplitContainer(specContainer, recipeContainer)
	}
	tab := props.Tabs[tabIndex]
	if itemIndex >= len(tab.Items) {
		return styled.NewVSplitContainer(specContainer, recipeContainer)
	}
	item := tab.Items[itemIndex]

	// 性能表示
	rawMaster := world.Resources.RawMaster
	spec, err := rawMaster.NewRecipeSpec(item.RecipeName)
	if err == nil {
		views.UpdateSpecFromSpec(world, specContainer, spec)
	}

	// レシピ表示
	if err == nil && spec.Recipe != nil {
		st.buildRecipeList(world, recipeContainer, spec.Recipe, res)
	}

	return styled.NewVSplitContainer(specContainer, recipeContainer)
}

func (st *CraftMenuState) buildRecipeList(world w.World, container *widget.Container, recipe *gc.Recipe, res *resources.UIResources) {
	for _, input := range recipe.Inputs {
		var currentAmount int
		if stackableEntity, found := worldhelper.FindStackableInInventory(world, input.Name); found {
			item := world.Components.Item.Get(stackableEntity).(*gc.Item)
			currentAmount = item.Count
		}
		str := fmt.Sprintf("%s %d pcs\n    所持: %d pcs", input.Name, input.Amount, currentAmount)
		var textColor color.RGBA
		if currentAmount >= input.Amount {
			textColor = consts.SuccessColor
		} else {
			textColor = consts.DangerColor
		}

		container.AddChild(styled.NewBodyText(str, textColor, res))
	}
}

func (st *CraftMenuState) buildDescContainer(world w.World, tabs []craftTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	desc := " "

	if tabIndex < len(tabs) && itemIndex < len(tabs[tabIndex].Items) {
		item := tabs[tabIndex].Items[itemIndex]
		rawMaster := world.Resources.RawMaster
		spec, err := rawMaster.NewRecipeSpec(item.RecipeName)
		if err == nil && spec.Description != nil {
			desc = spec.Description.Description
		}
	}

	if desc == "" {
		desc = " "
	}
	container.AddChild(styled.NewMenuText(desc, res))
	return container
}

func (st *CraftMenuState) buildActionWindow(world w.World, windowProps craftWindowProps) *widget.Window {
	res := world.Resources.UIResources
	actionIndex, _ := hooks.GetState[int](st.windowMount, "craft_window_index")
	actionItems := st.getActionItems(world, windowProps.RecipeName)

	windowContainer := styled.NewWindowContainer(res)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", res)
	window := styled.NewSmallWindow(titleContainer, windowContainer)

	for i, action := range actionItems {
		isSelected := i == actionIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, res)
		windowContainer.AddChild(actionWidget)
	}

	window.SetLocation(getCenterWinRect(world))
	return window
}

func (st *CraftMenuState) buildResultWindow(world w.World, resultProps craftResultProps) *widget.Window {
	res := world.Resources.UIResources
	resultIndex, _ := hooks.GetState[int](st.resultMount, "craft_result_index")
	resultItems := []string{TextClose}

	windowContainer := styled.NewWindowContainer(res)
	titleContainer := styled.NewWindowHeaderContainer("合成結果", res)
	window := styled.NewSmallWindow(titleContainer, windowContainer)

	// アイテム詳細を表示
	views.UpdateSpec(world, windowContainer, resultProps.ResultEntity)

	// ボタン項目を表示
	for i, action := range resultItems {
		isSelected := i == resultIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, res)
		windowContainer.AddChild(actionWidget)
	}

	window.SetLocation(getCenterWinRect(world))
	return window
}
