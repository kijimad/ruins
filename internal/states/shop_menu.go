package states

import (
	"fmt"
	"log"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	"github.com/kijimaD/ruins/internal/widgets/tabmenu"
	"github.com/kijimaD/ruins/internal/widgets/views"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ShopMenuState はショップメニューのゲームステート
type ShopMenuState struct {
	es.BaseState[w.World]
	ui *ebitenui.UI

	menuView            *tabmenu.View
	selectedItem        tabmenu.Item      // 選択中のアイテム
	itemDesc            *widget.Text      // アイテムの概要
	specContainer       *widget.Container // 性能表示のコンテナ
	rootContainer       *widget.Container
	tabDisplayContainer *widget.Container // タブ表示のコンテナ
	categoryContainer   *widget.Container // カテゴリ一覧のコンテナ
	currencyText        *widget.Text      // 所持金表示

	// アクション選択ウィンドウ用
	actionWindow     *widget.Window // アクション選択ウィンドウ
	actionFocusIndex int            // アクションウィンドウ内のフォーカス
	actionItems      []string       // アクション項目リスト
	isWindowMode     bool           // ウィンドウ操作モードかどうか
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
func (st *ShopMenuState) OnStart(world w.World) error {
	st.ui = st.initUI(world)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *ShopMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *ShopMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// キー入力をActionに変換
	var action inputmapper.ActionID
	var ok bool
	if st.isWindowMode {
		action, ok = HandleWindowInput()
	} else {
		action, ok = st.HandleInput()
	}

	if ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	// アクションウィンドウ表示中はTabMenuの更新をスキップ
	if !st.isWindowMode {
		if err := st.menuView.Update(); err != nil {
			return es.Transition[w.World]{}, err
		}
	}
	st.ui.Update()

	return st.ConsumeTransition(), nil
}

// Draw はゲームステートの描画処理を行う
func (st *ShopMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.ui.Draw(screen)
	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *ShopMenuState) HandleInput() (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return inputmapper.ActionMenuCancel, true
	}

	return "", false
}

// DoAction はActionを実行する
func (st *ShopMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	// ウィンドウモード時のアクション処理
	if st.isWindowMode {
		switch action {
		case inputmapper.ActionWindowUp, inputmapper.ActionWindowDown:
			if UpdateFocusIndex(action, &st.actionFocusIndex, len(st.actionItems)) {
				st.updateActionWindowDisplay(world)
			}
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowConfirm:
			st.executeActionItem(world)
			return es.Transition[w.World]{Type: es.TransNone}, nil
		case inputmapper.ActionWindowCancel:
			st.closeActionWindow()
			return es.Transition[w.World]{Type: es.TransNone}, nil
		default:
			return es.Transition[w.World]{}, fmt.Errorf("ウィンドウモード時の未知のアクション: %s", action)
		}
	}

	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// ================

func (st *ShopMenuState) initUI(world w.World) *ebitenui.UI {
	res := world.Resources.UIResources

	// TabMenuの設定
	tabs := st.createTabs(world)
	config := tabmenu.Config{
		Tabs:             tabs,
		InitialTabIndex:  0,
		InitialItemIndex: 0,
		WrapNavigation:   true,
		ItemsPerPage:     20,
	}

	callbacks := tabmenu.Callbacks{
		OnSelectItem: func(_ int, _ int, tab tabmenu.TabItem, item tabmenu.Item) error {
			return st.handleItemSelection(world, tab, item)
		},
		OnCancel: func() {
			st.SetTransition(es.Transition[w.World]{Type: es.TransPop})
		},
		OnTabChange: func(_, _ int, _ tabmenu.TabItem) {
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
			st.updateCategoryDisplay(world)
		},
		OnItemChange: func(_ int, _, _ int, item tabmenu.Item) error {
			if err := st.handleItemChange(world, item); err != nil {
				return err
			}
			st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
			return nil
		},
	}

	st.menuView = tabmenu.NewView(config, callbacks, world)

	// アイテムの説明文
	itemDescContainer := styled.NewRowContainer()
	st.itemDesc = styled.NewMenuText(" ", world.Resources.UIResources) // 空文字だと初期状態の縦サイズがなくなる
	itemDescContainer.AddChild(st.itemDesc)

	st.specContainer = styled.NewVerticalContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)

	// 初期状態の表示を更新
	if err := st.updateInitialItemDisplay(world); err != nil {
		log.Fatalf("Failed to update initial item display: %v", err)
		return nil
	}

	// タブ表示のコンテナを作成
	st.tabDisplayContainer = styled.NewVerticalContainer()
	st.createTabDisplayUI(world)

	// カテゴリ一覧のコンテナを作成（横並び）
	st.categoryContainer = styled.NewRowContainer()
	st.createCategoryDisplayUI(world)

	// 所持金表示コンテナを作成
	currencyContainer := styled.NewRowContainer()
	st.currencyText = styled.NewMenuText("", world.Resources.UIResources)
	st.updateCurrencyDisplay(world)
	currencyContainer.AddChild(st.currencyText)

	st.rootContainer = styled.NewItemGridContainer(
		widget.ContainerOpts.BackgroundImage(res.Panel.ImageTrans),
	)
	{
		// 3x3グリッドレイアウト
		// 1行目
		st.rootContainer.AddChild(styled.NewTitleText("店", world.Resources.UIResources))
		st.rootContainer.AddChild(st.categoryContainer) // カテゴリ一覧の表示
		st.rootContainer.AddChild(currencyContainer)

		// 2行目
		st.rootContainer.AddChild(st.tabDisplayContainer)
		st.rootContainer.AddChild(widget.NewContainer())
		st.rootContainer.AddChild(st.specContainer)

		// 3行目
		st.rootContainer.AddChild(itemDescContainer)
		st.rootContainer.AddChild(widget.NewContainer())
		st.rootContainer.AddChild(widget.NewContainer())
	}

	return &ebitenui.UI{Container: st.rootContainer}
}

// createTabs はTabMenuで使用するタブを作成する
func (st *ShopMenuState) createTabs(world w.World) []tabmenu.TabItem {
	return []tabmenu.TabItem{
		{
			ID:    "buy",
			Label: "購入",
			Items: st.createBuyItems(world),
		},
		{
			ID:    "sell",
			Label: "売却",
			Items: st.createSellItems(world),
		},
	}
}

// createBuyItems は購入アイテムリストを作成
func (st *ShopMenuState) createBuyItems(world w.World) []tabmenu.Item {
	shopInventory := worldhelper.GetShopInventory()
	items := make([]tabmenu.Item, 0, len(shopInventory))

	// プレイヤーの所持金を取得
	var currency int
	worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
		currency = worldhelper.GetCurrency(world, playerEntity)
	})

	for _, itemName := range shopInventory {
		price := st.getItemPrice(world, itemName, true)
		canAfford := currency >= price

		items = append(items, tabmenu.Item{
			Label:            itemName,
			AdditionalLabels: []string{worldhelper.FormatCurrency(price)},
			UserData:         map[string]interface{}{"itemName": itemName, "price": price, "isBuy": true},
			Disabled:         !canAfford, // 所持金が足りない場合はDisabled
		})
	}

	return items
}

// createSellItems は売却アイテムリストを作成
func (st *ShopMenuState) createSellItems(world w.World) []tabmenu.Item {
	var items []tabmenu.Item

	worldhelper.QueryPlayer(world, func(_ ecs.Entity) {
		world.Manager.Join(
			world.Components.Item,
			world.Components.Name,
			world.Components.ItemLocationInBackpack,
		).Visit(ecs.Visit(func(entity ecs.Entity) {
			nameComp := world.Components.Name.Get(entity).(*gc.Name)
			itemName := nameComp.Name

			baseValue := worldhelper.GetItemValue(world, entity)
			price := worldhelper.CalculateSellPrice(baseValue)

			// 追加ラベルを作成
			var additionalLabels []string

			// 価格を先に追加（常に表示されるため、位置が揃う）
			additionalLabels = append(additionalLabels, worldhelper.FormatCurrency(price))

			// 個数がある場合は後に追加（オプション）
			if entity.HasComponent(world.Components.Stackable) {
				itemComp := world.Components.Item.Get(entity).(*gc.Item)
				if itemComp.Count > 1 {
					additionalLabels = append(additionalLabels, fmt.Sprintf("x%d", itemComp.Count))
				}
			}

			items = append(items, tabmenu.Item{
				Label:            itemName,
				AdditionalLabels: additionalLabels,
				UserData: map[string]interface{}{
					"itemName": itemName,
					"entity":   entity,
					"price":    price,
					"isBuy":    false,
				},
			})
		}))
	})

	if len(items) == 0 {
		items = append(items, tabmenu.Item{
			Label:    "売却可能なアイテムがありません",
			UserData: map[string]interface{}{},
		})
	}

	return items
}

// getItemPrice はアイテム名から価格を計算
func (st *ShopMenuState) getItemPrice(world w.World, itemName string, isBuy bool) int {
	rawMaster := world.Resources.RawMaster.(*raw.Master)
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

// handleItemSelection はアイテム選択時の処理
func (st *ShopMenuState) handleItemSelection(world w.World, _ tabmenu.TabItem, item tabmenu.Item) error {
	if item.UserData == nil {
		return nil
	}

	st.selectedItem = item
	st.showActionWindow(world, item)
	return nil
}

// handleItemChange はアイテムフォーカス変更時の処理
func (st *ShopMenuState) handleItemChange(world w.World, item tabmenu.Item) error {
	if item.UserData == nil {
		st.itemDesc.Label = " "
		st.specContainer.RemoveChildren()
		return nil
	}

	data := item.UserData.(map[string]interface{})
	itemName, _ := data["itemName"].(string)

	// 性能表示をクリア
	st.specContainer.RemoveChildren()

	// アイテムの情報を取得する
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	spec, err := rawMaster.NewItemSpec(itemName)
	if err != nil {
		st.itemDesc.Label = TextNoDescription
		return err
	}

	// Descriptionを取得
	if spec.Description != nil {
		st.itemDesc.Label = spec.Description.Description
	} else {
		st.itemDesc.Label = TextNoDescription
	}

	// EntitySpecから性能表示を更新
	views.UpdateSpecFromSpec(world, st.specContainer, spec)
	return nil
}

// handlePurchase はアイテムの購入処理
func (st *ShopMenuState) handlePurchase(world w.World, item tabmenu.Item) {
	data := item.UserData.(map[string]interface{})
	itemName, _ := data["itemName"].(string)

	worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
		err := worldhelper.BuyItem(world, playerEntity, itemName)
		if err != nil {
			// エラーの場合は何もしない（通貨不足など）
			// TODO(kijima): error を返すようにする
			return
		}
		// タブを再読み込み
		st.reloadTabs(world)
	})
}

// handleSell はアイテムの売却処理
func (st *ShopMenuState) handleSell(world w.World, item tabmenu.Item) {
	data := item.UserData.(map[string]interface{})
	itemName, _ := data["itemName"].(string)
	entity, ok := data["entity"].(ecs.Entity)
	if !ok {
		return
	}
	price, _ := data["price"].(int)

	worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
		err := worldhelper.SellItem(world, playerEntity, entity)
		if err != nil {
			st.itemDesc.Label = fmt.Sprintf("売却失敗: %v", err)
		} else {
			st.itemDesc.Label = fmt.Sprintf("%sを売却しました（%s）", itemName, worldhelper.FormatCurrency(price))
			// タブを再読み込み
			st.reloadTabs(world)
		}
	})
}

// reloadTabs はタブの内容を再読み込みする
func (st *ShopMenuState) reloadTabs(world w.World) {
	newTabs := st.createTabs(world)
	st.menuView.UpdateTabs(newTabs)
	// UpdateTabs後に表示を更新
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
	st.updateCategoryDisplay(world)
	st.updateCurrencyDisplay(world)
}

// createTabDisplayUI はタブ表示UIを作成する
func (st *ShopMenuState) createTabDisplayUI(_ w.World) {
	st.menuView.UpdateTabDisplayContainer(st.tabDisplayContainer)
}

// createCategoryDisplayUI はカテゴリ表示UIを作成する
func (st *ShopMenuState) createCategoryDisplayUI(world w.World) {
	st.updateCategoryDisplay(world)
}

// updateCategoryDisplay はカテゴリ表示を更新する
func (st *ShopMenuState) updateCategoryDisplay(world w.World) {
	// 既存の子要素をクリア
	st.categoryContainer.RemoveChildren()

	// 全カテゴリを横並びで表示
	currentTabIndex := st.menuView.GetCurrentTabIndex()
	tabs := st.createTabs(world) // 最新のタブ情報を取得

	for i, tab := range tabs {
		isSelected := i == currentTabIndex
		if isSelected {
			// 選択中のカテゴリは背景色付きで明るい文字色
			categoryWidget := styled.NewListItemText(tab.Label, consts.TextColor, true, world.Resources.UIResources)
			st.categoryContainer.AddChild(categoryWidget)
		} else {
			// 非選択のカテゴリは背景なしでグレー文字色
			categoryWidget := styled.NewListItemText(tab.Label, consts.ForegroundColor, false, world.Resources.UIResources)
			st.categoryContainer.AddChild(categoryWidget)
		}
	}
}

// showActionWindow はアクションウィンドウを表示する
func (st *ShopMenuState) showActionWindow(world w.World, item tabmenu.Item) {
	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", world.Resources.UIResources)
	st.actionWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アクション項目を準備
	st.actionItems = []string{}
	st.selectedItem = item

	data := item.UserData.(map[string]interface{})
	isBuy, _ := data["isBuy"].(bool)

	if isBuy {
		// 購入時は所持金をチェック
		price, _ := data["price"].(int)
		var canAfford bool
		worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
			currency := worldhelper.GetCurrency(world, playerEntity)
			canAfford = currency >= price
		})

		// 所持金が足りる場合のみ「購入する」を表示
		if canAfford {
			st.actionItems = append(st.actionItems, "購入する")
		}
	} else {
		st.actionItems = append(st.actionItems, "売却する")
	}
	st.actionItems = append(st.actionItems, TextClose)

	st.actionFocusIndex = 0
	st.isWindowMode = true

	// UI要素を作成（表示のみ、操作はキーボードで行う）
	st.createActionWindowUI(world, windowContainer)

	st.actionWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.actionWindow)
}

// createActionWindowUI はアクションウィンドウのUI要素を作成する
func (st *ShopMenuState) createActionWindowUI(world w.World, _ *widget.Container) {
	st.updateActionWindowDisplay(world)
}

// updateActionWindowDisplay はアクションウィンドウの表示を更新する
func (st *ShopMenuState) updateActionWindowDisplay(world w.World) {
	if st.actionWindow == nil {
		return
	}

	// 既存のウィンドウを閉じて新しく作成
	st.actionWindow.Close()

	windowContainer := styled.NewWindowContainer(world.Resources.UIResources)
	titleContainer := styled.NewWindowHeaderContainer("アクション選択", world.Resources.UIResources)
	st.actionWindow = styled.NewSmallWindow(titleContainer, windowContainer)

	// アクション項目を表示
	for i, action := range st.actionItems {
		isSelected := i == st.actionFocusIndex
		actionWidget := styled.NewListItemText(action, consts.TextColor, isSelected, world.Resources.UIResources)
		windowContainer.AddChild(actionWidget)
	}

	st.actionWindow.SetLocation(getCenterWinRect(world))
	st.ui.AddWindow(st.actionWindow)
}

// closeActionWindow はアクションウィンドウを閉じる
func (st *ShopMenuState) closeActionWindow() {
	if st.actionWindow != nil {
		st.actionWindow.Close()
		st.actionWindow = nil
	}
	st.isWindowMode = false
	st.actionFocusIndex = 0
	st.actionItems = nil
}

// executeActionItem は選択されたアクション項目を実行する
func (st *ShopMenuState) executeActionItem(world w.World) {
	if st.actionFocusIndex >= len(st.actionItems) {
		return
	}

	selectedAction := st.actionItems[st.actionFocusIndex]

	switch selectedAction {
	case "購入する":
		st.handlePurchase(world, st.selectedItem)
		st.closeActionWindow()
	case "売却する":
		st.handleSell(world, st.selectedItem)
		st.closeActionWindow()
	case TextClose:
		st.closeActionWindow()
	}
}

// updateInitialItemDisplay は初期状態のアイテム表示を更新する
func (st *ShopMenuState) updateInitialItemDisplay(world w.World) error {
	currentTab := st.menuView.GetCurrentTab()
	currentItemIndex := st.menuView.GetCurrentItemIndex()

	if len(currentTab.Items) > 0 && currentItemIndex >= 0 && currentItemIndex < len(currentTab.Items) {
		currentItem := currentTab.Items[currentItemIndex]
		if err := st.handleItemChange(world, currentItem); err != nil {
			return err
		}
	}
	return nil
}

// updateCurrencyDisplay は所持金表示を更新する
func (st *ShopMenuState) updateCurrencyDisplay(world w.World) {
	if st.currencyText == nil {
		return
	}

	currency := 0
	worldhelper.QueryPlayer(world, func(playerEntity ecs.Entity) {
		currency = worldhelper.GetCurrency(world, playerEntity)
	})

	st.currencyText.Label = worldhelper.FormatCurrency(currency)
}
