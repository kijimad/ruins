package states

import (
	"fmt"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menurt"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ShopMenuState はショップメニューのゲームステート
type ShopMenuState struct {
	es.BaseState[w.World]
	detail menuscreen.Detail // 詳細モーダル。overlay として Screen に登録する
	screen *menurt.Screen[ShopProps]
}

// State interface ================

var _ es.State[w.World] = &ShopMenuState{}
var _ menurt.ExtraInput = &ShopMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *ShopMenuState) OnStart(_ w.World) error {
	st.detail = menuscreen.NewDetail(st.detailContent)
	st.screen = menurt.NewScreen[ShopProps](st, &st.detail)
	return nil
}

// Update はゲームステートの更新処理を行う
func (st *ShopMenuState) Update(world w.World) (es.Transition[w.World], error) {
	// 売買で所持品が変わると WeightDirty が立つ。この画面でも再計算を回し、総重量の表示を売買のたびに更新する
	if err := runUpdaters(world, &gs.WeightDirtySystem{}); err != nil {
		return es.Transition[w.World]{}, err
	}
	return st.screen.Update(world)
}

// Draw はゲームステートの描画処理を行う
func (st *ShopMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// ExtraInput は共通入力に加える独自キーを返す。x で選択中商品の詳細モーダルを開く
func (st *ShopMenuState) ExtraInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return "", false
}

// DoAction はActionを実行する
func (st *ShopMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open()
	case inputmapper.ActionMenuSelect:
		if err := st.buySellSelected(world); err != nil {
			return es.Transition[w.World]{}, err
		}
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("shopMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// ShopProps は画面の表示 props。menurt.Screen の型引数として渡す
type ShopProps struct {
	Tabs []shopTabData
}

type shopTabData struct {
	ID    string
	Label string
	Items []shopItemData
}

type shopItemData struct {
	ItemID   string // アイテムの同定キー。NewItemSpec/BuyItem/価格はこれで引く
	Label    string // 表示名
	Weight   string
	Price    int
	Count    int // 売却時のアイテム個数
	Entity   ecs.Entity
	IsBuy    bool
	Disabled bool
}

// Fetch は世界から表示 props を構築する。menurt.Model の Model 部にあたる
func (st *ShopMenuState) Fetch(world w.World) ShopProps {
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

	return ShopProps{
		Tabs: st.createTabs(world, currency, buyPriceMod, sellPriceMod),
	}
}

// Menu は一覧の構成を返す。menurt.Model の Menu 部にあたる
func (st *ShopMenuState) Menu(props ShopProps) menurt.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menurt.MenuConfig{Key: "shop", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuItemsPerPage}
}

func (st *ShopMenuState) createTabs(world w.World, currency int, buyPriceMod, sellPriceMod consts.Percent) []shopTabData {
	return []shopTabData{
		{ID: "buy", Label: query.T(world, "Buy"), Items: st.createBuyItems(world, currency, buyPriceMod)},
		{ID: "sell", Label: query.T(world, "Sell"), Items: st.createSellItems(world, sellPriceMod)},
	}
}

func (st *ShopMenuState) createBuyItems(world w.World, currency int, buyPriceMod consts.Percent) []shopItemData {
	shopInventory := gameaction.GetShopInventory()
	items := make([]shopItemData, 0, len(shopInventory))

	for _, itemID := range shopInventory {
		price := buyPriceMod.ApplyInt(st.getItemPrice(world, itemID, true))
		canAfford := currency >= price

		label := itemID
		if itemDef, err := raw.FindItem(world.Resources.RawMaster, itemID); err == nil {
			label = itemDef.Name
		}

		items = append(items, shopItemData{
			ItemID:   itemID,
			Label:    label,
			Weight:   shopItemWeight(world, itemID),
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
		sellQuery := ecs.NewFilter3[gc.Name, gc.RawID, gc.LocationInBackpack](world.ECS).Query()
		for sellQuery.Next() {
			entity := sellQuery.Entity()
			nameComp := world.Components.Name.Get(entity)
			rawID := world.Components.RawID.Get(entity)

			baseValue := query.GetItemValue(world, entity)
			price := sellPriceMod.ApplyInt(query.CalculateSellPrice(baseValue))

			count := query.GetEntityCount(world, entity)

			items = append(items, shopItemData{
				ItemID: rawID.ID,
				Label:  nameComp.Name,
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
func shopItemWeight(world w.World, itemID string) string {
	spec, err := raw.NewItemSpec(world.Resources.RawMaster, itemID)
	if err != nil || spec.Weight == nil {
		return ""
	}
	return spec.Weight.KgString()
}

func (st *ShopMenuState) getItemPrice(world w.World, itemID string, isBuy bool) int {
	itemDef, err := raw.FindItem(world.Resources.RawMaster, itemID)
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

// buySellSelected は現在カーソルが当たっている商品を、購入タブなら購入、売却タブなら売却する。
// 決定で即実行し、途中のアクション選択は挟まない。購入は所持金が足りなければ何もしない
func (st *ShopMenuState) buySellSelected(world w.World) error {
	item, ok := st.selectedShopItem()
	if !ok || item.ItemID == "" {
		return nil
	}
	if item.IsBuy {
		canAfford := false
		query.Player(world, func(p ecs.Entity) { canAfford = query.GetCurrency(world, p) >= item.Price })
		if !canAfford {
			return nil
		}
		var err error
		query.Player(world, func(p ecs.Entity) { err = gameaction.BuyItem(world, p, item.ItemID) })
		if err != nil {
			return fmt.Errorf("failed to buy: %w", err)
		}
		return nil
	}
	var err error
	query.Player(world, func(p ecs.Entity) { err = gameaction.SellItem(world, p, item.Entity) })
	if err != nil {
		return fmt.Errorf("failed to sell: %w", err)
	}
	return nil
}

// selectedShopItem は現在カーソルが当たっている商品を返す
func (st *ShopMenuState) selectedShopItem() (shopItemData, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return shopItemData{}, false
	}
	items := props.Tabs[cursor.TabIndex].Items
	if cursor.ItemIndex >= len(items) {
		return shopItemData{}, false
	}
	return items[cursor.ItemIndex], true
}

// ================
// View
// ================

// View は props を UI へ組む純粋な描画。menurt.Model の View 部にあたる
func (st *ShopMenuState) View(world w.World, props ShopProps, cursor menurt.Selection, res resources.UIResources) *ebitenui.UI {
	// 購入と売却をタブ帯に寄せ、本体は1カラムの一覧にする。性能は x の詳細モーダルで見る
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	return newTabScreenUI(res, tabScreen{
		TabLabels: labels,
		TabIndex:  cursor.TabIndex,
		Content:   st.buildItemContainer(world, props.Tabs, cursor.TabIndex, cursor.ItemIndex, res),
		Footer:    menuNavHint(world, true, query.T(world, "x Details")),
	})
}

// detailContent は現在カーソルが当たっている商品の詳細内容を raw 定義から解決する。詳細モーダルの唯一の定義点。
func (st *ShopMenuState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	item, spec, ok := st.selectedDetail(world)
	if !ok {
		return menuscreen.DetailContent{}, false
	}
	// 価格・重さは一覧に出すので、詳細の説明は raw のアイテム説明だけにする
	desc := ""
	if spec.Description != nil {
		desc = spec.Description.Description
	}
	return menuscreen.DetailContent{Name: item.Label, Desc: desc, Spec: &spec}, true
}

// selectedDetail は現在カーソルが当たっている商品と、その raw 由来の性能を解決する
func (st *ShopMenuState) selectedDetail(world w.World) (shopItemData, gc.EntitySpec, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return shopItemData{}, gc.EntitySpec{}, false
	}
	items := props.Tabs[cursor.TabIndex].Items
	if cursor.ItemIndex >= len(items) {
		return shopItemData{}, gc.EntitySpec{}, false
	}
	item := items[cursor.ItemIndex]
	s, err := raw.NewItemSpec(world.Resources.RawMaster, item.ItemID)
	if err != nil {
		return shopItemData{}, gc.EntitySpec{}, false
	}
	return item, s, true
}

func (st *ShopMenuState) buildItemContainer(world w.World, tabs []shopTabData, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	if tabIndex >= len(tabs) {
		return styled.NewVerticalContainer()
	}

	currentTab := tabs[tabIndex]
	// 名前+個数、価格、重さの3列。名前を伸縮させ、価格・重さを右側にまとめる。
	// 重さは最も重い値、15.00kg 相当、が収まる幅を固定で取り、値の桁数で価格位置がぶれないようにする。
	// 売却の個数は名前に x個数 として添える。性能は x の詳細モーダルで見る
	columnWidths := []int{0, 80, 90}
	aligns := []styled.TextAlign{styled.AlignLeft, styled.AlignRight, styled.AlignRight}
	rows := make([]menuRow, len(currentTab.Items))
	for i, it := range currentTab.Items {
		rows[i] = menuRow{Cells: []string{nameWithCount(it.Label, it.Count), query.FormatCurrency(it.Price), it.Weight}}
	}
	emptyText := query.T(world, "No goods")
	if currentTab.ID == "sell" {
		emptyText = query.T(world, "No items to sell")
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: emptyText}, res)
}
