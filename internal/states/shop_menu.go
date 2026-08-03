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

// ShopMenuState はショップメニューのゲームステート
type ShopMenuState struct {
	es.BaseState[w.World]
	detail    menuscreen.Detail       // 詳細モーダルの表示状態とページ送り
	actionWin menuscreen.ActionWindow // 購入・売却のアクション選択ウィンドウ
	rebuild   bool                    // 次フレームで UI を作り直すか
	menuMount *hooks.Mount[shopProps]
	widget    *ebitenui.UI
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
	st.menuMount = hooks.NewMount[shopProps]()
	st.detail = menuscreen.NewDetail(st.detailContent)
	st.actionWin = menuscreen.NewActionWindow(st.actionWindowContent)
	return nil
}

// OnStop はステートが停止される際に呼ばれる
func (st *ShopMenuState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *ShopMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 入力処理。詳細・アクション窓が開いていればそちらが優先し、通常のメニュー入力は止まる
	if st.detail.Active() {
		if st.detail.HandleInput(world) {
			st.rebuild = true
		}
	} else if st.actionWin.Active() {
		dirty, err := st.actionWin.HandleInput(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if dirty {
			st.rebuild = true
		}
	} else if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
		st.menuMount.Dispatch(action)
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

	if st.menuMount.Update() || st.widget == nil || st.rebuild {
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

// HandleInput はキー入力をActionに変換する。アクション窓の入力は Update 側で actionWin が扱う
func (st *ShopMenuState) HandleInput(_ *config.Config) (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return HandleMenuInput()
}

// DoAction はActionを実行する
func (st *ShopMenuState) DoAction(_ w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open()
		st.rebuild = true
	case inputmapper.ActionMenuSelect:
		st.actionWin.Open()
		st.rebuild = true
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("shopMenu: 未対応のアクション: %s", action)
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
				Weight: query.GetEntityWeight(world, entity).KgString(),
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
	return spec.Weight.KgString()
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

// actionWindowContent は現在カーソルが当たっている商品の見出しと選択肢を返す。アクション窓の唯一の定義点。
// 選択肢の実行内容も Run に閉じ込め、購入・売却・閉じるを1箇所で定義する
func (st *ShopMenuState) actionWindowContent(world w.World) (string, []menuscreen.Action, bool) {
	item, ok := st.selectedShopItem()
	if !ok || item.Label == "" {
		return "", nil, false
	}
	var actions []menuscreen.Action
	if item.IsBuy {
		canAfford := false
		query.Player(world, func(p ecs.Entity) { canAfford = query.GetCurrency(world, p) >= item.Price })
		if canAfford {
			actions = append(actions, menuscreen.Action{Label: TextBuy, Run: func(world w.World) error {
				var err error
				query.Player(world, func(p ecs.Entity) { err = gameaction.BuyItem(world, p, item.Label) })
				if err != nil {
					return fmt.Errorf("購入に失敗: %w", err)
				}
				return nil
			}})
		}
	} else {
		actions = append(actions, menuscreen.Action{Label: TextSell, Run: func(world w.World) error {
			var err error
			query.Player(world, func(p ecs.Entity) { err = gameaction.SellItem(world, p, item.Entity) })
			if err != nil {
				return fmt.Errorf("売却に失敗: %w", err)
			}
			return nil
		}})
	}
	actions = append(actions, menuscreen.Action{Label: TextClose})
	return "アクション選択", actions, true
}

// selectedShopItem は現在カーソルが当たっている商品を返す
func (st *ShopMenuState) selectedShopItem() (shopItemData, bool) {
	props := st.menuMount.GetProps()
	menuState, _ := hooks.GetState[hooks.TabMenuState](st.menuMount, "shop")
	if menuState.TabIndex >= len(props.Tabs) {
		return shopItemData{}, false
	}
	items := props.Tabs[menuState.TabIndex].Items
	if menuState.ItemIndex >= len(items) {
		return shopItemData{}, false
	}
	return items[menuState.ItemIndex], true
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
	if st.detail.Active() {
		if win := st.detail.Window(world, getCenterWinRect(world)); win != nil {
			eui.AddWindow(win)
		}
	}

	// アクション選択ウィンドウ
	if st.actionWin.Active() {
		if win := st.actionWin.Window(world, getCenterWinRect(world)); win != nil {
			eui.AddWindow(win)
		}
	}

	return eui
}

// detailContent は現在カーソルが当たっている商品の詳細内容を raw 定義から解決する。詳細モーダルの唯一の定義点
func (st *ShopMenuState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	name, desc, spec, ok := st.selectedDetail(world)
	if !ok {
		return menuscreen.DetailContent{}, false
	}
	return menuscreen.DetailContent{Name: name, Desc: desc, Spec: &spec}, true
}

// selectedDetail は現在カーソルが当たっている商品の名前・説明・性能を raw 定義から解決する
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
