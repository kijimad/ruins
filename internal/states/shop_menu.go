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
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/pagination"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// shopSubState はショップメニュー内のサブステート
type shopSubState int

const (
	shopSubStateMenu   shopSubState = iota // メニュー選択
	shopSubStateWindow                     // アクションウィンドウ
)

// ShopMenuState はショップメニューのゲームステート
type ShopMenuState struct {
	es.BaseState[w.World]
	subState    shopSubState
	showDetail  bool // x の詳細モーダルを表示中か
	detailPage  int  // 詳細モーダルの表示ページ
	rebuild     bool // 次フレームで UI を作り直すか
	menuMount   *hooks.Mount[shopProps]
	windowMount *hooks.Mount[shopWindowProps]
	widget      *ebitenui.UI
}

// State interface ================

var _ es.State[w.World] = &ShopMenuState{}
var _ Configurable = &ShopMenuState{}

// StateConfig は背景のブラーと暗幕を無効にする。後ろのフィールドをそのまま見せる
func (st *ShopMenuState) StateConfig() StateConfig {
	return StateConfig{BlurBackground: false}
}

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
		ItemsPerPage: menuItemsPerPage,
	})

	// ウィンドウ用のステート
	st.setupWindowState(world)

	menuDirty := st.menuMount.Update()
	windowDirty := st.windowMount.Update()
	if menuDirty || windowDirty || st.widget == nil || st.rebuild {
		st.widget = st.buildUI(world)
		st.rebuild = false
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
	ki := input.GetSharedKeyboardInput()
	if st.showDetail {
		if ki.IsKeyJustPressed(ebiten.KeyEscape) || ki.IsKeyJustPressed(ebiten.KeyX) || ki.IsEnterJustPressedOnce() {
			return inputmapper.ActionMenuCancel, true
		}
		if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowLeft) {
			return inputmapper.ActionMenuLeft, true
		}
		if ki.IsKeyPressedWithRepeat(ebiten.KeyArrowRight) {
			return inputmapper.ActionMenuRight, true
		}
		return "", false
	}
	switch st.subState {
	case shopSubStateMenu:
		if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
			return inputmapper.ActionOpenItemDetail, true
		}
		return HandleMenuInput()
	case shopSubStateWindow:
		return HandleWindowInput()
	}
	return "", false
}

// DoAction はActionを実行する
func (st *ShopMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	if st.showDetail {
		switch action {
		case inputmapper.ActionMenuCancel:
			st.showDetail = false
			st.rebuild = true
		case inputmapper.ActionMenuLeft:
			if st.detailPage > 0 {
				st.detailPage--
				st.rebuild = true
			}
		case inputmapper.ActionMenuRight:
			if _, _, spec, ok := st.selectedDetail(world); ok && st.detailPage < menuscreen.DetailPageCountFromSpec(world, spec)-1 {
				st.detailPage++
				st.rebuild = true
			}
		default:
			// 詳細表示中は他のアクションを無視する
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

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
		case inputmapper.ActionOpenItemDetail:
			st.showDetail = true
			st.detailPage = 0
			st.rebuild = true
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
	Weight   string
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
	buyPriceMod, sellPriceMod := consts.PercentBase, consts.PercentBase
	query.Player(world, func(playerEntity ecs.Entity) {
		currency = query.GetCurrency(world, playerEntity)
		if world.Components.CharModifiers.Has(playerEntity) {
			mods := world.Components.CharModifiers.Get(playerEntity)
			buyPriceMod = mods.BuyPrice
			sellPriceMod = mods.SellPrice
		}
	})

	return shopProps{
		Tabs:     st.createTabs(world, currency, buyPriceMod, sellPriceMod),
		Currency: currency,
	}
}

func (st *ShopMenuState) createTabs(world w.World, currency int, buyPriceMod, sellPriceMod consts.Percent) []shopTabData {
	return []shopTabData{
		{ID: "buy", Label: "購入", Items: st.createBuyItems(world, currency, buyPriceMod)},
		{ID: "sell", Label: "売却", Items: st.createSellItems(world, sellPriceMod)},
	}
}

func (st *ShopMenuState) createBuyItems(world w.World, currency int, buyPriceMod consts.Percent) []shopItemData {
	shopInventory := gameaction.GetShopInventory()
	items := make([]shopItemData, 0, len(shopInventory))

	for _, itemName := range shopInventory {
		price := buyPriceMod.ApplyInt(st.getItemPrice(world, itemName, true))
		canAfford := currency >= price

		items = append(items, shopItemData{
			Label:    itemName,
			Weight:   shopItemWeight(world, itemName),
			Price:    price,
			IsBuy:    true,
			Disabled: !canAfford,
		})
	}

	return items
}

func (st *ShopMenuState) createSellItems(world w.World, sellPriceMod consts.Percent) []shopItemData {
	var items []shopItemData

	query.Player(world, func(_ ecs.Entity) {
		sellQuery := ecs.NewFilter2[gc.Name, gc.LocationInBackpack](world.ECS).Query()
		for sellQuery.Next() {
			entity := sellQuery.Entity()
			nameComp := world.Components.Name.Get(entity)
			itemName := nameComp.Name

			baseValue := query.GetItemValue(world, entity)
			price := sellPriceMod.ApplyInt(query.CalculateSellPrice(baseValue))

			count := query.GetEntityCount(world, entity)

			items = append(items, shopItemData{
				Label:  itemName,
				Weight: query.GetEntityWeight(world, entity).String(),
				Price:  price,
				Count:  count,
				Entity: entity,
				IsBuy:  false,
			})
		}
	})

	return items
}

// shopItemWeight は raw 定義から商品1個の重量表記を返す。重量を持たない品は空文字を返す
func shopItemWeight(world w.World, label string) string {
	spec, err := raw.NewItemSpec(world.Resources.RawMaster, label)
	if err != nil || spec.Weight == nil {
		return ""
	}
	return spec.Weight.String()
}

func (st *ShopMenuState) getItemPrice(world w.World, itemName string, isBuy bool) int {
	itemDef, err := raw.FindItem(world.Resources.RawMaster, itemName)
	if err != nil {
		return 0
	}
	baseValue := int(itemDef.Value)
	if isBuy {
		return query.CalculateBuyPrice(baseValue)
	}
	return query.CalculateSellPrice(baseValue)
}

// ================
// Window
// ================

func (st *ShopMenuState) setupWindowState(world w.World) {
	windowProps := st.windowMount.GetProps()
	actionItems := st.getActionItems(world, windowProps.SelectedItem)
	hooks.UseState(st.windowMount.Store(), "shop_window_index", 0, menuscreen.WindowCursorReducer(len(actionItems)))
}

func (st *ShopMenuState) getActionItems(world w.World, item shopItemData) []string {
	if item.Label == "" {
		return []string{TextClose}
	}

	actionItems := []string{}

	if item.IsBuy {
		var canAfford bool
		query.Player(world, func(playerEntity ecs.Entity) {
			currency := query.GetCurrency(world, playerEntity)
			canAfford = currency >= item.Price
		})
		if canAfford {
			actionItems = append(actionItems, TextBuy)
		}
	} else {
		actionItems = append(actionItems, TextSell)
	}
	actionItems = append(actionItems, TextClose)

	return actionItems
}

func (st *ShopMenuState) handleItemSelection(_ w.World) error {
	props := st.menuMount.GetProps()
	menuState, ok := hooks.GetState[hooks.TabMenuState](st.menuMount, "shop")
	if !ok {
		return fmt.Errorf("shopの取得に失敗")
	}
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

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

	var actionErr error
	switch selectedAction {
	case TextBuy:
		query.Player(world, func(playerEntity ecs.Entity) {
			actionErr = gameaction.BuyItem(world, playerEntity, windowProps.SelectedItem.Label)
		})
		if actionErr != nil {
			return fmt.Errorf("購入に失敗: %w", actionErr)
		}
		st.subState = shopSubStateMenu
	case TextSell:
		query.Player(world, func(playerEntity ecs.Entity) {
			actionErr = gameaction.SellItem(world, playerEntity, windowProps.SelectedItem.Entity)
		})
		if actionErr != nil {
			return fmt.Errorf("売却に失敗: %w", actionErr)
		}
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
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "shop")
	tabIndex := menuState.TabIndex
	itemIndex := menuState.ItemIndex

	// 購入と売却をタブ帯に寄せ、本体は1カラムの一覧にする。性能は x の詳細モーダルで見る
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}

	eui := newTabScreenUI(res, tabScreen{
		Header:    fmt.Sprintf("所持 %s", query.FormatCurrency(props.Currency)),
		TabLabels: labels,
		TabIndex:  tabIndex,
		Content:   st.buildItemContainer(props.Tabs, tabIndex, itemIndex, res),
		Footer:    menuNavHint(true, "x 詳細"),
	})

	// 詳細モーダル
	if st.showDetail {
		if name, desc, spec, ok := st.selectedDetail(world); ok {
			eui.AddWindow(menuscreen.BuildDetailWindowFromSpec(world, getCenterWinRect(world), name, desc, spec, st.detailPage))
		}
	}

	// アクション選択ウィンドウ
	if st.subState == shopSubStateWindow {
		eui.AddWindow(st.buildActionWindow(world, st.windowMount.GetProps()))
	}

	return eui
}

// selectedDetail は現在カーソルが当たっている商品の詳細を raw 定義から解決する
func (st *ShopMenuState) selectedDetail(world w.World) (name, desc string, spec gc.EntitySpec, ok bool) {
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "shop")
	if menuState.TabIndex >= len(props.Tabs) {
		return "", "", gc.EntitySpec{}, false
	}
	items := props.Tabs[menuState.TabIndex].Items
	if menuState.ItemIndex >= len(items) {
		return "", "", gc.EntitySpec{}, false
	}
	label := items[menuState.ItemIndex].Label
	s, err := raw.NewItemSpec(world.Resources.RawMaster, label)
	if err != nil {
		return "", "", gc.EntitySpec{}, false
	}
	d := ""
	if s.Description != nil {
		d = s.Description.Description
	}
	return label, d, s, true
}

func (st *ShopMenuState) buildItemContainer(tabs []shopTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	container := styled.NewVerticalContainer()
	if tabIndex >= len(tabs) {
		return container
	}

	currentTab := tabs[tabIndex]
	pg := pagination.New(itemIndex, len(currentTab.Items), menuItemsPerPage)

	// ページインジケーター（上部固定位置、右寄せ）
	container.AddChild(newPageIndicator(pg, res))

	// 名前+個数、重量、価格の3列。売却の個数は名前に x個数 として添える
	columnWidths := []int{200, 70, 80}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight, styled.AlignRight}
	table := styled.NewTableContainer(columnWidths, res)
	for _, entry := range pagination.VisibleEntries(currentTab.Items, pg) {
		isSelected := pg.IsSelectedInPage(entry.Index)
		priceStr := query.FormatCurrency(entry.Item.Price)
		styled.NewTableRow(table, columnWidths, []string{nameWithCount(entry.Item.Label, entry.Item.Count), entry.Item.Weight, priceStr}, aligns, &isSelected, res)
	}
	container.AddChild(table)

	if len(currentTab.Items) == 0 {
		if currentTab.ID == "sell" {
			container.AddChild(styled.NewDescriptionText("売却可能なアイテムがありません", res))
		} else {
			container.AddChild(styled.NewDescriptionText("(商品なし)", res))
		}
	}

	return container
}

func (st *ShopMenuState) buildActionWindow(world w.World, windowProps shopWindowProps) *widget.Window {
	actionIndex, _ := hooks.GetState[int](st.windowMount, "shop_window_index")
	actionItems := st.getActionItems(world, windowProps.SelectedItem)
	return menuscreen.BuildActionWindow(world, getCenterWinRect(world), "アクション選択", actionItems, actionIndex)
}
