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

type shopItemData struct {
	Entity    ecs.Entity // 在庫・持ち物の実体。買=これを移動、売=これを移動、雇用=これを活性化
	ItemID    string     // アイテムの raw 同定キー。詳細表示に使う。隊員候補は空
	Label     string     // 表示名
	Weight    string
	Price     int
	Count     int // 実体の個数
	IsBuy     bool
	IsRecruit bool // 隊員候補なら真。買うと雇用になる
	Disabled  bool
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
		price := buyPriceMod.ApplyInt(query.CalculateBuyPrice(query.StockBaseValue(world, entity)))
		data := shopItemData{
			Entity:    entity,
			Price:     price,
			Count:     query.GetEntityCount(world, entity),
			IsBuy:     true,
			IsRecruit: query.IsRecruit(world, entity),
			Disabled:  currency < price,
		}
		name := world.Components.Name.Get(entity).Name
		if data.IsRecruit {
			// 候補名はローマ字の固有名なので言語に依らずそのまま出す
			data.Label = name
		} else {
			data.ItemID = world.Components.RawID.Get(entity).ID
			data.Label = query.T(world, name)
			data.Weight = query.GetEntityWeight(world, entity).KgString()
		}
		items = append(items, data)
	}

	return items
}

// createSellItems はプレイヤーの持ち物を売りタブへ並べる。売ると実体が商人の在庫へ移る
func (st *ShopMenuState) createSellItems(world w.World, sellPriceMod consts.Percent) []shopItemData {
	var items []shopItemData

	query.Player(world, func(_ ecs.Entity) {
		sellQuery := ecs.NewFilter3[gc.Name, gc.RawID, gc.LocationInBackpack](world.ECS).Query()
		for sellQuery.Next() {
			entity := sellQuery.Entity()
			nameComp := world.Components.Name.Get(entity)
			rawID := world.Components.RawID.Get(entity)

			price := sellPriceMod.ApplyInt(query.CalculateSellPrice(query.StockBaseValue(world, entity)))

			items = append(items, shopItemData{
				Entity: entity,
				ItemID: rawID.ID,
				Label:  query.T(world, nameComp.Name),
				Weight: query.GetEntityWeight(world, entity).KgString(),
				Price:  price,
				Count:  query.GetEntityCount(world, entity),
				IsBuy:  false,
			})
		}
	})

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
		var err error
		if item.IsRecruit {
			query.Player(world, func(p ecs.Entity) { err = gameaction.HireRecruit(world, p, item.Entity) })
		} else {
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

// detailContent は現在カーソルが当たっている行の詳細内容を返す。詳細モーダルの唯一の定義点。
// アイテムは raw 定義から性能を、隊員候補は能力値を出す
func (st *ShopMenuState) detailContent(world w.World) (menuscreen.DetailContent, bool) {
	item, ok := st.selectedShopItem()
	if !ok {
		return menuscreen.DetailContent{}, false
	}

	if item.IsRecruit {
		if !world.ECS.Alive(item.Entity) || !world.Components.Abilities.Has(item.Entity) {
			return menuscreen.DetailContent{}, false
		}
		a := world.Components.Abilities.Get(item.Entity)
		stats := query.T(world, "Vit%d Str%d Sen%d Dex%d Agi%d Def%d", a.Vitality.Base, a.Strength.Base, a.Sensation.Base, a.Dexterity.Base, a.Agility.Base, a.Defense.Base)
		return menuscreen.DetailContent{Name: item.Label, Desc: stats}, true
	}

	spec, err := raw.NewItemSpec(world.Resources.RawMaster, item.ItemID)
	if err != nil {
		return menuscreen.DetailContent{}, false
	}
	// 価格・重さは一覧に出すので、詳細の説明は raw のアイテム説明だけにする
	desc := ""
	if spec.Description != nil {
		desc = query.T(world, spec.Description.Description)
	}
	return menuscreen.DetailContent{Name: item.Label, Desc: desc, Spec: &spec}, true
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
