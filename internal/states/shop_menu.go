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
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/gameaction"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ShopMenuState はショップメニューのゲームステート。
// 商人の在庫を売買する。買うと在庫が減り、売ると売った品が在庫に並ぶ
type ShopMenuState struct {
	es.BaseState[w.World]
	merchant ecs.Entity        // 品揃えを LocationInStorage で持つ商人。売買の相手
	detail   menuscreen.Detail // 詳細モーダル。overlay として Screen に登録する
	screen   *menurt.Screen[ShopProps]
}

// State interface ================

var _ es.State[w.World] = &ShopMenuState{}
var _ menurt.ExtraInput = &ShopMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *ShopMenuState) OnStart(_ w.World) error {
	st.detail = menuscreen.NewEntityDetail(func() (ecs.Entity, bool) {
		it, ok := st.selectedShopItem()
		return it.Entity, ok
	})
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
		st.detail.Open(world)
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

// shopItemData は一覧1行分。名前・重量・個数は実体から都度出せるので持たず、
// プレイヤーの所持金や倍率が要る値だけを持つ
type shopItemData struct {
	Entity   ecs.Entity // 在庫・持ち物の実体。表示も操作もこれから解決する
	Price    int        // 価値と交渉スキルの倍率から出す。実体だけでは決まらない
	IsBuy    bool       // 買いタブの行なら真
	Disabled bool       // 所持金が足りず選べない
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

// createBuyItems は商人の在庫を買いタブへ並べる。アイテムと隊員候補が同列に並ぶ
func (st *ShopMenuState) createBuyItems(world w.World, currency int, buyPriceMod consts.Percent) []shopItemData {
	stock := query.GetStorageItems(world, st.merchant)
	items := make([]shopItemData, 0, len(stock))

	for _, entity := range stock {
		base := query.GetItemValue(world, entity) * query.GetEntityCount(world, entity)
		price := buyPriceMod.ApplyInt(query.CalculateBuyPrice(base))
		items = append(items, shopItemData{
			Entity:   entity,
			Price:    price,
			IsBuy:    true,
			Disabled: currency < price,
		})
	}

	return items
}

// createSellItems はプレイヤーの持ち物を売りタブへ並べる。売ると実体が商人の在庫へ移る。
// プレイヤーが居ないときは何も並べない。存在確認を先に済ませ、収集クエリは query.Player の
// コールバック外で回してクエリのネストを避ける
func (st *ShopMenuState) createSellItems(world w.World, sellPriceMod consts.Percent) []shopItemData {
	if _, err := query.GetPlayerEntity(world); err != nil {
		return nil
	}

	var items []shopItemData
	sellQuery := ecs.NewFilter2[gc.Name, gc.LocationInBackpack](world.ECS).Query()
	for sellQuery.Next() {
		entity := sellQuery.Entity()
		base := query.GetItemValue(world, entity) * query.GetEntityCount(world, entity)
		price := sellPriceMod.ApplyInt(query.CalculateSellPrice(base))
		items = append(items, shopItemData{
			Entity: entity,
			Price:  price,
			IsBuy:  false,
		})
	}

	return items
}

// ================
// Window
// ================

// buySellSelected は現在カーソルが当たっている行を、買いタブなら購入・雇用、売りタブなら売却する。
// 決定で即実行し、途中のアクション選択は挟まない。購入・雇用は所持金が足りなければ何もしない
func (st *ShopMenuState) buySellSelected(world w.World) error {
	item, ok := st.selectedShopItem()
	if !ok {
		return nil
	}
	if item.IsBuy {
		canAfford := false
		query.Player(world, func(p ecs.Entity) { canAfford = query.GetCurrency(world, p) >= item.Price })
		if !canAfford {
			return nil
		}
		// 選択が古く実体が消えていれば何もしない。dead entity への Has/移動は panic するため先に守る
		if !world.ECS.Alive(item.Entity) {
			return nil
		}
		// 在庫の類型で分ける。多数のアイテム類型から隊員候補だけ雇用へ、残りは購入へ振る
		var err error
		cat, _ := world.Components.CategoryOf(gc.ItemTypeCategoryKey, item.Entity)
		switch cat {
		case gc.CategoryRecruit:
			query.Player(world, func(p ecs.Entity) { err = gameaction.HireRecruit(world, p, item.Entity) })
		default:
			query.Player(world, func(p ecs.Entity) { err = gameaction.BuyStock(world, p, item.Entity) })
		}
		if err != nil {
			return fmt.Errorf("failed to buy: %w", err)
		}
		return nil
	}
	var err error
	query.Player(world, func(p ecs.Entity) { err = gameaction.SellStock(world, p, st.merchant, item.Entity) })
	if err != nil {
		return fmt.Errorf("failed to sell: %w", err)
	}
	return nil
}

// selectedShopItem は現在カーソルが当たっている行を返す
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
		// 名前・個数・重量は実体から都度出す。一覧の実体は毎フレーム集め直すので描画時も生存している
		name := query.T(world, world.Components.Name.Get(it.Entity).Name)
		count := query.GetEntityCount(world, it.Entity)
		weight := query.GetEntityWeight(world, it.Entity).KgString()
		rows[i] = menuRow{Cells: []string{nameWithCount(name, count), query.FormatCurrency(it.Price), weight}}
	}
	emptyText := query.T(world, "No goods")
	if currentTab.ID == "sell" {
		emptyText = query.T(world, "No items to sell")
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: emptyText}, res)
}
