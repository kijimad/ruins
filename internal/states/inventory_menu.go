package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InventoryMenuState はインベントリメニューのゲームステート
type InventoryMenuState struct {
	es.BaseState[w.World]
	menuMount   *ui.Mount[inventoryProps]
	windowMount *ui.Mount[windowProps] // アクションウィンドウ用
	widget      *ebitenui.UI
}

func (st InventoryMenuState) String() string {
	return "InventoryMenu"
}

// State interface ================

var _ es.State[w.World] = &InventoryMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *InventoryMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *InventoryMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *InventoryMenuState) OnStart(_ w.World) error {
	st.menuMount = ui.NewMount[inventoryProps]()
	st.windowMount = ui.NewMount[windowProps]()
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

	// ウィンドウのPropsを設定
	windowProps := st.windowMount.GetProps()

	// キー入力をActionに変換
	var action inputmapper.ActionID
	var ok bool
	if windowProps.Open {
		action, ok = HandleWindowInput()
	} else {
		action, ok = HandleMenuInput()
	}

	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		if !windowProps.Open {
			st.menuMount.Dispatch(action)
		}
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	// UseTabMenuを呼んでreducerを登録・更新
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	ui.UseTabMenu(st.menuMount.Store(), "inventory", ui.TabMenuConfig{
		TabCount:   len(props.Tabs),
		ItemCounts: itemCounts,
	})

	// ウィンドウ用のUseStateを登録
	st.setupWindowState(world)

	if st.menuMount.Update() || st.windowMount.Update() {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// setupWindowState はウィンドウ用のUseStateを登録する
func (st *InventoryMenuState) setupWindowState(world w.World) {
	windowProps := st.windowMount.GetProps()
	actionCount := len(st.getActionItems(world, windowProps.SelectedEntity))

	ui.UseState(st.windowMount.Store(), "focusIndex", 0, func(v int, a inputmapper.ActionID) int {
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

// DoAction はActionを実行する
func (st *InventoryMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	windowProps := st.windowMount.GetProps()

	// ウィンドウモード時のアクション処理
	if windowProps.Open {
		switch action {
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			st.windowMount.Dispatch(action)
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowConfirm:
			if err := st.executeActionItem(world); err != nil {
				return es.Transition[w.World]{}, err
			}
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowCancel:
			st.closeActionWindow()
			return es.Transition[w.World]{Type: es.TransNone}, nil
		default:
			return es.Transition[w.World]{}, fmt.Errorf("ウィンドウモード時の未知のアクション: %s", action)
		}
	}

	switch action {
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect:
		st.handleItemSelection()
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight:
		return es.Transition[w.World]{Type: es.TransNone}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
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
	Open           bool
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
	tabIndex, _ := ui.GetState[int](st.menuMount, "inventory_tabIndex")
	itemIndex, _ := ui.GetState[int](st.menuMount, "inventory_itemIndex")

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
	windowProps := st.windowMount.GetProps()
	if windowProps.Open {
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

	table := styled.NewTableContainer(columnWidths, res)
	for i, item := range currentTab.Items {
		isSelected := i == itemIndex
		styled.NewTableRow(table, columnWidths, []string{"", item.Name, item.Count}, aligns, &isSelected, res)
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

func (st *InventoryMenuState) handleItemSelection() {
	props := st.menuMount.GetProps()
	tabIndex, _ := ui.GetState[int](st.menuMount, "inventory_tabIndex")
	itemIndex, _ := ui.GetState[int](st.menuMount, "inventory_itemIndex")

	if tabIndex >= len(props.Tabs) {
		return
	}
	if itemIndex >= len(props.Tabs[tabIndex].Items) {
		return
	}

	item := props.Tabs[tabIndex].Items[itemIndex]
	st.openActionWindow(item.Entity)
}

func (st *InventoryMenuState) openActionWindow(entity ecs.Entity) {
	// windowMountを再作成して状態をリセット
	st.windowMount = ui.NewMount[windowProps]()
	st.windowMount.SetProps(windowProps{
		Open:           true,
		SelectedEntity: entity,
	})
}

func (st *InventoryMenuState) closeActionWindow() {
	st.windowMount.SetProps(windowProps{
		Open:           false,
		SelectedEntity: 0,
	})
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
	focusIndex, _ := ui.GetState[int](st.windowMount, "focusIndex")

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

	focusIndex, _ := ui.GetState[int](st.windowMount, "focusIndex")

	actions := st.getActionItems(world, entity)
	if focusIndex >= len(actions) {
		return nil
	}

	selectedAction := actions[focusIndex]

	switch selectedAction {
	case "使う":
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		params := activity.ActionParams{
			Actor:  playerEntity,
			Target: &entity,
		}
		_, err = activity.Execute(&activity.UseItemActivity{}, params, world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		st.closeActionWindow()
	case "捨てる":
		playerEntity, err := worldhelper.GetPlayerEntity(world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		params := activity.ActionParams{
			Actor:  playerEntity,
			Target: &entity,
		}
		_, err = activity.Execute(&activity.DropActivity{}, params, world)
		if err != nil {
			st.closeActionWindow()
			return err
		}

		st.closeActionWindow()
	case TextClose:
		st.closeActionWindow()
	}

	return nil
}
