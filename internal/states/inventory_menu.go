package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// inventorySubState はインベントリメニュー内のサブステート
type inventorySubState int

const (
	invSubStateMenu   inventorySubState = iota // メニュー選択
	invSubStateWindow                          // アクションウィンドウ
)

const inventoryItemsPerPage = 20

// InventoryMenuState はインベントリメニューのゲームステート
type InventoryMenuState struct {
	es.BaseState[w.World]
	subState    inventorySubState
	menuMount   *hooks.Mount[inventoryProps]
	windowMount *hooks.Mount[windowProps]
	widget      *ebitenui.UI
}

func (st InventoryMenuState) String() string {
	return "InventoryMenu"
}

// State interface ================

var _ es.State[w.World] = &InventoryMenuState{}
var _ es.ActionHandler[w.World] = &InventoryMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *InventoryMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *InventoryMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *InventoryMenuState) OnStart(_ w.World) error {
	st.subState = invSubStateMenu
	st.menuMount = hooks.NewMount[inventoryProps]()
	st.windowMount = hooks.NewMount[windowProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *InventoryMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *InventoryMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// InventoryChangedSystemを実行して所持重量を更新
	for _, updater := range []w.Updater{
		&gs.InventoryChangedSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		switch st.subState {
		case invSubStateMenu:
			st.menuMount.Dispatch(action)
		case invSubStateWindow:
			st.windowMount.Dispatch(action)
		}
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	// UseTabMenuを呼んでreducerを登録・更新
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(st.menuMount.Store(), "inventory", hooks.TabMenuConfig{
		TabCount:     len(props.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: inventoryItemsPerPage,
	})

	// ウィンドウ用のUseStateを登録
	st.setupWindowState(world)

	menuDirty := st.menuMount.Update()
	windowDirty := st.windowMount.Update()
	if menuDirty || windowDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// setupWindowState はウィンドウ用のUseStateを登録する
func (st *InventoryMenuState) setupWindowState(world w.World) {
	windowProps := st.windowMount.GetProps()
	actionCount := len(st.getActionItems(world, windowProps.SelectedEntity))

	hooks.UseState(st.windowMount.Store(), "focusIndex", 0, func(v int, a inputmapper.ActionID) int {
		if actionCount == 0 {
			return 0
		}
		switch a {
		case inputmapper.ActionWindowUp:
			return (v - 1 + actionCount) % actionCount
		case inputmapper.ActionWindowDown:
			return (v + 1) % actionCount
		default:
			return v
		}
	})
}

// Draw はゲームステートの描画処理を行う
func (st *InventoryMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *InventoryMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	switch st.subState {
	case invSubStateMenu:
		return HandleMenuInput()
	case invSubStateWindow:
		return HandleWindowInput()
	}
	return "", false
}

// DoAction はActionを実行する
func (st *InventoryMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.subState {
	case invSubStateWindow:
		switch action {
		case inputmapper.ActionWindowConfirm:
			if err := st.executeActionItem(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionWindowCancel:
			st.subState = invSubStateMenu
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("invSubStateWindow: 未対応のアクション: %s", action)
		}

	case invSubStateMenu:
		switch action {
		case inputmapper.ActionOpenDebugMenu:
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			return es.Transition[w.World]{Type: es.TransPop}, nil
		case inputmapper.ActionMenuSelect:
			if err := st.handleItemSelection(); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("invSubStateMenu: 未対応のアクション: %s", action)
		}
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type inventoryProps struct {
	Tabs []inventoryTabData
}

type inventoryTabData struct {
	ID    string
	Label string
	Items []inventoryItemData
}

type inventoryItemData struct {
	Entity ecs.Entity
	Name   string
	Count  string
	Desc   string
}

// windowProps はアクションウィンドウ用のProps
type windowProps struct {
	SelectedEntity ecs.Entity
}

func (st *InventoryMenuState) fetchProps(world w.World) inventoryProps {
	return inventoryProps{
		Tabs: []inventoryTabData{
			{ID: "items", Label: "道具", Items: st.createItemData(world, st.queryMenuItem(world))},
			{ID: "weapons", Label: "武器", Items: st.createItemData(world, st.queryMenuWeapon(world))},
			{ID: "wearables", Label: "防具", Items: st.createItemData(world, st.queryMenuWearable(world))},
		},
	}
}

func (st *InventoryMenuState) createItemData(world w.World, entities []ecs.Entity) []inventoryItemData {
	items := make([]inventoryItemData, len(entities))

	for i, entity := range entities {
		name := world.Components.Name.Get(entity).(*gc.Name).Name

		item := inventoryItemData{
			Entity: entity,
			Name:   name,
		}

		// Stackableコンポーネントがあれば個数を表示する
		if entity.HasComponent(world.Components.Stackable) {
			itemComp := world.Components.Item.Get(entity).(*gc.Item)
			item.Count = fmt.Sprintf("%d", itemComp.Count)
		}

		// 説明文
		if entity.HasComponent(world.Components.Description) {
			desc := world.Components.Description.Get(entity).(*gc.Description)
			item.Desc = desc.Description
		}

		items[i] = item
	}

	return items
}

func (st *InventoryMenuState) queryMenuItem(world w.World) []ecs.Entity {
	items := []ecs.Entity{}

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
		world.Components.Wearable.Not(),
		world.Components.Weapon.Not(),
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		items = append(items, entity)
	}))

	return worldhelper.SortEntities(world, items)
}

func (st *InventoryMenuState) queryMenuWeapon(world w.World) []ecs.Entity {
	items := []ecs.Entity{}

	world.Manager.Join(
		world.Components.Item,
		world.Components.Weapon,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		items = append(items, entity)
	}))

	return worldhelper.SortEntities(world, items)
}

func (st *InventoryMenuState) queryMenuWearable(world w.World) []ecs.Entity {
	items := []ecs.Entity{}

	world.Manager.Join(
		world.Components.Item,
		world.Components.Wearable,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		items = append(items, entity)
	}))

	return worldhelper.SortEntities(world, items)
}

// ================
// buildUI
// ================

func (st *InventoryMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "inventory")
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	root.AddChild(styled.NewTitleText("インベントリ", res))
	root.AddChild(st.buildCategoryContainer(props.Tabs, tabIndex, res))
	root.AddChild(widget.NewContainer())

	root.AddChild(st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(st.buildSpecContainer(world, props, tabIndex, itemIndex, res))

	root.AddChild(st.buildDescContainer(props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	result := &ebitenui.UI{Container: root}

	// アクションウィンドウが開いている場合は追加
	if st.subState == invSubStateWindow {
		actionWindow := st.buildActionWindow(world, res)
		result.AddWindow(actionWindow)
	}

	return result
}

func (st *InventoryMenuState) buildCategoryContainer(tabs []inventoryTabData, tabIndex int, res *resources.UIResources) *widget.Container {
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

func (st *InventoryMenuState) buildItemContainer(tabs []inventoryTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(tabs) {
		return container
	}

	currentTab := tabs[tabIndex]
	columnWidths := []int{20, 150, 50}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight}

	// ペジネーション
	pg := pagination.New(itemIndex, len(currentTab.Items), inventoryItemsPerPage)

	// ページインジケーター（上部固定位置、右寄せ）
	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " " // 空でも高さを確保
	}
	container.AddChild(styled.NewPageIndicator(pageText, res))

	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		isSelected := pg.IsSelectedInPage(entry.Index)
		styled.NewTableRow(table, columnWidths, []string{"", entry.Item.Name, entry.Item.Count}, aligns, &isSelected, res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		container.AddChild(styled.NewDescriptionText("(アイテムなし)", res))
	}

	return container
}

func (st *InventoryMenuState) buildSpecContainer(world w.World, props inventoryProps, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	if tabIndex >= len(props.Tabs) {
		return container
	}
	if itemIndex >= len(props.Tabs[tabIndex].Items) {
		return container
	}

	item := props.Tabs[tabIndex].Items[itemIndex]
	views.UpdateSpec(world, container, item.Entity)

	return container
}

func (st *InventoryMenuState) buildDescContainer(tabs []inventoryTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	desc := " "
	if tabIndex < len(tabs) && itemIndex < len(tabs[tabIndex].Items) {
		desc = tabs[tabIndex].Items[itemIndex].Desc
	}
	if desc == "" {
		desc = " "
	}
	container.AddChild(styled.NewMenuText(desc, res))
	return container
}

// ================
// アクションウィンドウ
// ================

func (st *InventoryMenuState) handleItemSelection() error {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "inventory")
	if !ok {
		return fmt.Errorf("inventoryの取得に失敗")
	}
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	if tabIndex >= len(props.Tabs) {
		return nil
	}
	if itemIndex >= len(props.Tabs[tabIndex].Items) {
		return nil
	}

	item := props.Tabs[tabIndex].Items[itemIndex]
	st.subState = invSubStateWindow
	st.windowMount = hooks.NewMount[windowProps]()
	st.windowMount.SetProps(windowProps{
		SelectedEntity: item.Entity,
	})
	return nil
}

// getActionItems は指定されたエンティティで利用可能なアクション一覧を返す
func (st *InventoryMenuState) getActionItems(world w.World, entity ecs.Entity) []string {
	if entity == 0 {
		return []string{}
	}

	actions := []string{}

	if entity.HasComponent(world.Components.Consumable) {
		actions = append(actions, "使う")
	}
	actions = append(actions, "捨てる")
	actions = append(actions, TextClose)

	return actions
}

func (st *InventoryMenuState) buildActionWindow(world w.World, res *resources.UIResources) *widget.Window {
	windowContainer := styled.NewWindowContainer(res)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", res)
	actionWindow := styled.NewSmallWindow(titleContainer, windowContainer)

	windowProps := st.windowMount.GetProps()
	actions := st.getActionItems(world, windowProps.SelectedEntity)
	focusIndex, _ := hooks.GetState[int](st.windowMount, "focusIndex")

	for i, action := range actions {
		isSelected := i == focusIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, res)
		windowContainer.AddChild(actionWidget)
	}

	actionWindow.SetLocation(getCenterWinRect(world))
	return actionWindow
}

func (st *InventoryMenuState) executeActionItem(world w.World) error {
	windowProps := st.windowMount.GetProps()
	entity := windowProps.SelectedEntity

	focusIndex, ok := hooks.GetState[int](st.windowMount, "focusIndex")
	if !ok {
		return fmt.Errorf("focusIndexの取得に失敗")
	}

	actions := st.getActionItems(world, entity)
	if focusIndex >= len(actions) {
		return nil
	}

	selectedAction := actions[focusIndex]

	switch selectedAction {
	case "使う":
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			st.subState = invSubStateMenu
			return err
		}

		params := activity.ActionParams{
			Actor:  playerEntity,
			Target: &entity,
		}
		_, err = activity.Execute(&activity.UseItemActivity{}, params, world)
		if err != nil {
			st.subState = invSubStateMenu
			return err
		}

		st.subState = invSubStateMenu
	case "捨てる":
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			st.subState = invSubStateMenu
			return err
		}

		params := activity.ActionParams{
			Actor:  playerEntity,
			Target: &entity,
		}
		_, err = activity.Execute(&activity.DropActivity{}, params, world)
		if err != nil {
			st.subState = invSubStateMenu
			return err
		}

		st.subState = invSubStateMenu
	case TextClose:
		st.subState = invSubStateMenu
	}

	return nil
}
