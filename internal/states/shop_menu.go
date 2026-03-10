package states

import (
	"fmt"

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

// shopSubState はショップメニュー内のサブステート
type shopSubState int

const (
	shopSubStateMenu   shopSubState = iota // メニュー選択
	shopSubStateWindow                     // アクションウィンドウ
)

const shopItemsPerPage = 20

// ShopMenuState はショップメニューのゲームステート
type ShopMenuState struct {
	es.BaseState[w.World]
	subState    shopSubState
	menuMount   *hooks.Mount[shopProps]
	windowMount *hooks.Mount[shopWindowProps]
	widget      *ebitenui.UI
}

func (st ShopMenuState) String() string {
	return "ShopMenu"
}

// State interface ================

var _ es.State[w.World] = &ShopMenuState{}
var _ es.ActionHandler[w.World] = &ShopMenuState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *ShopMenuState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *ShopMenuState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *ShopMenuState) OnStart(_ w.World) error {
	st.subState = shopSubStateMenu
	st.menuMount = hooks.NewMount[shopProps]()
	st.windowMount = hooks.NewMount[shopWindowProps]()
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *ShopMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *ShopMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 入力処理
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		switch st.subState {
		case shopSubStateMenu:
			st.menuMount.Dispatch(action)
		case shopSubStateWindow:
			st.windowMount.Dispatch(action)
		}
	}

	props := st.fetchProps(world)
	st.menuMount.SetProps(props)

	// UseTabMenuでreducerを登録・更新
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	hooks.UseTabMenu(st.menuMount.Store(), "shop", hooks.TabMenuConfig{
		TabCount:     len(props.Tabs),
		ItemCounts:   itemCounts,
		ItemsPerPage: shopItemsPerPage,
	})

	// ウィンドウ用のステート
	st.setupWindowState(world)

	menuDirty := st.menuMount.Update()
	windowDirty := st.windowMount.Update()
	if menuDirty || windowDirty || st.widget == nil {
		st.widget = st.buildUI(world)
	}

	st.widget.Update()
	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *ShopMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.widget.Draw(screen)
	return nil
}

// HandleInput はキー入力をActionに変換する
func (st *ShopMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	switch st.subState {
	case shopSubStateMenu:
		return HandleMenuInput()
	case shopSubStateWindow:
		return HandleWindowInput()
	}
	return "", false
}

// DoAction はActionを実行する
func (st *ShopMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch st.subState {
	case shopSubStateWindow:
		switch action {
		case inputmapper.ActionWindowConfirm:
			if err := st.executeActionItem(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionWindowCancel:
			st.subState = shopSubStateMenu
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("shopSubStateWindow: 未対応のアクション: %s", action)
		}

	case shopSubStateMenu:
		switch action {
		case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
			return es.Transition[w.World]{Type: es.TransPop}, nil
		case inputmapper.ActionMenuSelect:
			if err := st.handleItemSelection(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
			// Dispatchで処理される
		default:
			return es.Transition[w.World]{}, fmt.Errorf("shopSubStateMenu: 未対応のアクション: %s", action)
		}
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

type shopProps struct {
	Tabs     []shopTabData
	Currency int
}

type shopTabData struct {
	ID    string
	Label string
	Items []shopItemData
}

type shopItemData struct {
	Label    string
	Price    int
	Count    int // 売却時のアイテム個数
	Entity   ecs.Entity
	IsBuy    bool
	Disabled bool
}

type shopWindowProps struct {
	SelectedItem shopItemData
}

func (st *ShopMenuState) fetchProps(world w.World) shopProps {
	var currency int
	worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
		currency = worldhelper.GetCurrency(world, playerEntity)
	})

	return shopProps{
		Tabs:     st.createTabs(world, currency),
		Currency: currency,
	}
}

func (st *ShopMenuState) createTabs(world w.World, currency int) []shopTabData {
	return []shopTabData{
		{ID: "buy", Label: "購入", Items: st.createBuyItems(world, currency)},
		{ID: "sell", Label: "売却", Items: st.createSellItems(world)},
	}
}

func (st *ShopMenuState) createBuyItems(world w.World, currency int) []shopItemData {
	shopInventory := worldhelper.GetShopInventory()
	items := make([]shopItemData, 0, len(shopInventory))

	for _, itemName := range shopInventory {
		price := st.getItemPrice(world, itemName, true)
		canAfford := currency >= price

		items = append(items, shopItemData{
			Label:    itemName,
			Price:    price,
			IsBuy:    true,
			Disabled: !canAfford,
		})
	}

	return items
}

func (st *ShopMenuState) createSellItems(world w.World) []shopItemData {
	var items []shopItemData

	worldhelper.QueryPlayer(world, func(_ ecs.Entity) {
		world.Manager.Join(
			world.Components.Item,
			world.Components.Name,
			world.Components.ItemLocationInPlayerBackpack,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			nameComp := world.Components.Name.Get(entity).(*gc.Name)
			itemName := nameComp.Name

			baseValue := worldhelper.GetItemValue(world, entity)
			price := worldhelper.CalculateSellPrice(baseValue)

			count := 1
			if entity.HasComponent(world.Components.Stackable) {
				itemComp := world.Components.Item.Get(entity).(*gc.Item)
				count = itemComp.Count
			}

			items = append(items, shopItemData{
				Label:  itemName,
				Price:  price,
				Count:  count,
				Entity: entity,
				IsBuy:  false,
			})
		}))
	})

	return items
}

func (st *ShopMenuState) getItemPrice(world w.World, itemName string, isBuy bool) int {
	rawMaster := world.Resources.RawMaster
	itemIdx, ok := rawMaster.ItemIndex[itemName]
	if !ok {
		return 0
	}
	itemDef := rawMaster.Raws.Items[itemIdx]
	if itemDef.Value == nil {
		return 0
	}

	baseValue := *itemDef.Value
	if isBuy {
		return worldhelper.CalculateBuyPrice(baseValue)
	}
	return worldhelper.CalculateSellPrice(baseValue)
}

// ================
// Window
// ================

func (st *ShopMenuState) setupWindowState(world w.World) {
	windowProps := st.windowMount.GetProps()
	actionItems := st.getActionItems(world, windowProps.SelectedItem)

	hooks.UseState(st.windowMount.Store(), "shop_window_index", 0, func(v int, action inputmapper.ActionID) int {
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

func (st *ShopMenuState) getActionItems(world w.World, item shopItemData) []string {
	if item.Label == "" {
		return []string{TextClose}
	}

	actionItems := []string{}

	if item.IsBuy {
		var canAfford bool
		worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
			currency := worldhelper.GetCurrency(world, playerEntity)
			canAfford = currency >= item.Price
		})
		if canAfford {
			actionItems = append(actionItems, "購入する")
		}
	} else {
		actionItems = append(actionItems, "売却する")
	}
	actionItems = append(actionItems, TextClose)

	return actionItems
}

func (st *ShopMenuState) handleItemSelection(_ w.World) error {
	props := st.menuMount.GetProps()
	tabIndex, ok := hooks.GetState[int](st.menuMount, "shop_tabIndex")
	if !ok {
		return fmt.Errorf("shop_tabIndexの取得に失敗")
	}
	itemIndex, ok := hooks.GetState[int](st.menuMount, "shop_itemIndex")
	if !ok {
		return fmt.Errorf("shop_itemIndexの取得に失敗")
	}

	if tabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[tabIndex]
	if itemIndex >= len(tab.Items) {
		return nil
	}
	item := tab.Items[itemIndex]

	st.subState = shopSubStateWindow
	st.windowMount = hooks.NewMount[shopWindowProps]()
	st.windowMount.SetProps(shopWindowProps{
		SelectedItem: item,
	})
	return nil
}

func (st *ShopMenuState) executeActionItem(world w.World) error {
	windowProps := st.windowMount.GetProps()
	actionIndex, ok := hooks.GetState[int](st.windowMount, "shop_window_index")
	if !ok {
		return fmt.Errorf("shop_window_indexの取得に失敗")
	}
	actionItems := st.getActionItems(world, windowProps.SelectedItem)

	if actionIndex >= len(actionItems) {
		return nil
	}

	selectedAction := actionItems[actionIndex]

	switch selectedAction {
	case "購入する":
		worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
			_ = worldhelper.BuyItem(world, playerEntity, windowProps.SelectedItem.Label)
		})
		st.subState = shopSubStateMenu
	case "売却する":
		worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
			_ = worldhelper.SellItem(world, playerEntity, windowProps.SelectedItem.Entity)
		})
		st.subState = shopSubStateMenu
	case TextClose:
		st.subState = shopSubStateMenu
	}
	return nil
}

// ================
// buildUI
// ================

func (st *ShopMenuState) buildUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources
	props := st.menuMount.GetProps()
	tabIndex, _ := hooks.GetState[int](st.menuMount, "shop_tabIndex")
	itemIndex, _ := hooks.GetState[int](st.menuMount, "shop_itemIndex")

	root := styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 1行目: タイトル、カテゴリ、所持金
	root.AddChild(styled.NewTitleText("店", res))
	root.AddChild(st.buildCategoryContainer(props.Tabs, tabIndex, res))
	root.AddChild(st.buildCurrencyContainer(props.Currency, res))

	// 2行目: アイテム一覧、空、性能表示
	root.AddChild(st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(st.buildSpecContainer(world, props, tabIndex, itemIndex, res))

	// 3行目: 説明文
	root.AddChild(st.buildDescContainer(world, props.Tabs, tabIndex, itemIndex, res))
	root.AddChild(widget.NewContainer())
	root.AddChild(widget.NewContainer())

	eui := &ebitenui.UI{Container: root}

	// ウィンドウを追加
	if st.subState == shopSubStateWindow {
		actionWindow := st.buildActionWindow(world, st.windowMount.GetProps())
		eui.AddWindow(actionWindow)
	}

	return eui
}

func (st *ShopMenuState) buildCategoryContainer(tabs []shopTabData, tabIndex int, res *resources.UIResources) *widget.Container {
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

func (st *ShopMenuState) buildCurrencyContainer(currency int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	container.AddChild(styled.NewMenuText(worldhelper.FormatCurrency(currency), res))
	return container
}

func (st *ShopMenuState) buildItemContainer(tabs []shopTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(tabs) {
		return container
	}

	currentTab := tabs[tabIndex]
	pg := pagination.New(itemIndex, len(currentTab.Items), shopItemsPerPage)

	// ページインジケーター（上部固定位置、右寄せ）
	pageText := pg.GetPageText()
	if pageText == "" {
		pageText = " "
	}
	container.AddChild(styled.NewPageIndicator(pageText, res))

	// 購入タブ: カーソル、名前、価格の3列
	// 売却タブ: カーソル、名前、価格、個数の4列
	if currentTab.ID == "buy" {
		columnWidths := []int{20, 150, 80}
		aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight}

		table := styled.NewTableContainer(columnWidths, res)
		for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
			isSelected := pg.IsSelectedInPage(entry.Index)
			priceStr := worldhelper.FormatCurrency(entry.Item.Price)
			styled.NewTableRow(table, columnWidths, []string{"", entry.Item.Label, priceStr}, aligns, &isSelected, res)
		}
		container.AddChild(table)
	} else {
		columnWidths := []int{20, 150, 80, 40}
		aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight, styled.AlignRight}

		table := styled.NewTableContainer(columnWidths, res)
		for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
			isSelected := pg.IsSelectedInPage(entry.Index)
			priceStr := worldhelper.FormatCurrency(entry.Item.Price)
			countStr := ""
			if entry.Item.Count > 1 {
				countStr = fmt.Sprintf("x%d", entry.Item.Count)
			}
			styled.NewTableRow(table, columnWidths, []string{"", entry.Item.Label, priceStr, countStr}, aligns, &isSelected, res)
		}
		container.AddChild(table)
	}

	if len(currentTab.Items) == 0 {
		if currentTab.ID == "sell" {
			container.AddChild(styled.NewDescriptionText("売却可能なアイテムがありません", res))
		} else {
			container.AddChild(styled.NewDescriptionText("(商品なし)", res))
		}
	}

	return container
}

func (st *ShopMenuState) buildSpecContainer(world w.World, props shopProps, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	if tabIndex >= len(props.Tabs) {
		return container
	}
	tab := props.Tabs[tabIndex]
	if itemIndex >= len(tab.Items) {
		return container
	}
	item := tab.Items[itemIndex]

	// RawMasterからEntitySpecを取得して性能を表示
	rawMaster := world.Resources.RawMaster
	spec, err := rawMaster.NewItemSpec(item.Label)
	if err != nil {
		return container
	}

	views.UpdateSpecFromSpec(world, container, spec)
	return container
}

func (st *ShopMenuState) buildDescContainer(world w.World, tabs []shopTabData, tabIndex, itemIndex int, res *resources.UIResources) *widget.Container {
	container := styled.NewRowContainer()
	desc := " "

	if tabIndex < len(tabs) && itemIndex < len(tabs[tabIndex].Items) {
		item := tabs[tabIndex].Items[itemIndex]
		rawMaster := world.Resources.RawMaster
		spec, err := rawMaster.NewItemSpec(item.Label)
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

func (st *ShopMenuState) buildActionWindow(world w.World, windowProps shopWindowProps) *widget.Window {
	res := world.Resources.UIResources
	actionIndex, _ := hooks.GetState[int](st.windowMount, "shop_window_index")
	actionItems := st.getActionItems(world, windowProps.SelectedItem)

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
