package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
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
	merchant ecs.Entity     // 品揃えを LocationInStorage で持つ商人。売買の相手
	detail   overlay.Detail // 詳細モーダル。overlay として Screen に登録する
	screen   *menuloop.Screen[ShopProps]
}

// State interface ================

var _ es.State[w.World] = &ShopMenuState{}

// shopSellTabID は売却タブの識別子。空表示の文言分岐に使う。
const shopSellTabID = "sell"

var _ menuloop.KeyBindings = &ShopMenuState{}

// OnStart はステートが開始される際に呼ばれる
func (st *ShopMenuState) OnStart(_ w.World) error {
	st.detail = overlay.NewEntityDetail(func() (ecs.Entity, bool) {
		it, ok := st.selectedShopItem()
		return it.Entity, ok
	})
	st.screen = menuloop.NewScreen[ShopProps](st, &st.detail)
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

// KeyBindings は x の詳細表示を共通入力に足す
func (st *ShopMenuState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
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
	default:
		return es.Transition[w.World]{}, fmt.Errorf("shopMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// ================
// Props
// ================

// ShopProps は画面の表示 props。menuloop.Screen の型引数として渡す
type ShopProps struct {
	Tabs []shopTabData
}

type shopTabData struct {
	ID    string
	Label string
	Items []shopItemData
}

// shopItemData は一覧1行分。1行は1スタックで、売買はスタック丸ごと行う。
// 名前・重量・個数は実体から都度出せるので持たず、プレイヤーの所持金や倍率が要る値だけを持つ
type shopItemData struct {
	Entity   ecs.Entity      // スタック代表の実体。表示も操作もこれから解決する
	Count    int             // スタックの個数。束ねた結果を持ち回り、行ごとに数え直さない
	Price    consts.Currency // 行の合計額。単価は価値と交渉スキルの倍率から出し、スタック個数を掛ける
	IsBuy    bool            // 買いタブの行なら真
	Disabled bool            // 所持金が足りず選べない
}

// Fetch は世界から表示 props を構築する。menuloop.Model の Model 部にあたる。
// ショップはプレイヤーの操作でしか開かないので、プレイヤー不在は不変条件違反として返す。
// 価格は query.BuyPrice/SellPrice が取引と揃えて出す
func (st *ShopMenuState) Fetch(world w.World) (ShopProps, error) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return ShopProps{}, err
	}
	currency := query.GetCurrency(world, player)

	return ShopProps{
		Tabs: []shopTabData{
			{ID: "buy", Label: query.T(world, "Buy"), Items: st.createBuyItems(world, player, currency)},
			{ID: shopSellTabID, Label: query.T(world, "Sell"), Items: st.createSellItems(world, player)},
		},
	}, nil
}

// Menu は一覧の構成を返す。menuloop.Model の Menu 部にあたる
func (st *ShopMenuState) Menu(props ShopProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Items)
	}
	return menuloop.MenuConfig{Key: "shop", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

// createBuyItems は商人の在庫アイテムを買いタブへ並べる。同一スタックは1行に束ね、
// 額はスタック個数分の合計にする。購入はスタック丸ごとなので、表示額と支払額が一致する
func (st *ShopMenuState) createBuyItems(world w.World, player ecs.Entity, currency consts.Currency) []shopItemData {
	stacks := query.StorageStacks(world, st.merchant)
	items := make([]shopItemData, 0, len(stacks))

	for _, stack := range stacks {
		// BuyPrice は価値×スタック個数で既に全量の額を返す
		total := query.BuyPrice(world, player, stack.Rep)
		items = append(items, shopItemData{
			Entity:   stack.Rep,
			Count:    stack.Count,
			Price:    total,
			IsBuy:    true,
			Disabled: currency < total,
		})
	}

	return items
}

// createSellItems はプレイヤーの持ち物を売りタブへ並べる。売ると実体が商人の在庫へ移る。
// 同一スタックは1行に束ね、額はスタック個数分の合計にする
func (st *ShopMenuState) createSellItems(world w.World, player ecs.Entity) []shopItemData {
	stacks := query.BackpackStacks(world, player)
	items := make([]shopItemData, 0, len(stacks))
	for _, stack := range stacks {
		// SellPrice は価値×スタック個数で既に全量の額を返す
		items = append(items, shopItemData{
			Entity: stack.Rep,
			Count:  stack.Count,
			Price:  query.SellPrice(world, player, stack.Rep),
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
		var err error
		query.Player(world, func(p ecs.Entity) { err = gameaction.BuyStock(world, p, item.Entity) })
		if err != nil {
			return fmt.Errorf("failed to buy: %w", err)
		}
		return nil
	}
	// 選択が古く実体が消えていれば何もしない。dead entity への移動は panic するため先に守る
	if !world.ECS.Alive(item.Entity) {
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

// ViewUI は View の internal/ui 版。購入・売却タブとアイコン付き商品一覧を自前 UI で組む。
func (st *ShopMenuState) ViewUI(world w.World, props ShopProps, cursor menuloop.Selection, res resources.UIResources) ui.Widget {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	content := st.buildItemListUI(world, props.Tabs, cursor.TabIndex, cursor.ItemIndex, cursor.PageSize, res)
	return buildTabScreenUI(world, res, "", labels, cursor.TabIndex, content, keybind.HelpHint(world))
}

// buildItemListUI は buildItemContainer の internal/ui 版。
func (st *ShopMenuState) buildItemListUI(world w.World, tabs []shopTabData, tabIndex, itemIndex, perPage int, res resources.UIResources) []ui.Widget {
	if tabIndex >= len(tabs) {
		return nil
	}
	currentTab := tabs[tabIndex]
	columnWidths, aligns := itemMenuColumns(0, menuColumn{Width: 80, Align: styled.AlignRight}, menuColumn{Width: 90, Align: styled.AlignRight})
	rows := make([]menuRow, len(currentTab.Items))
	for i, it := range currentTab.Items {
		total := query.GetEntityWeight(world, it.Entity) * consts.Milligram(it.Count)
		rows[i] = itemMenuRow(world, it.Entity, it.Count, it.Price.String(), total.KgString())
	}
	emptyText := query.T(world, "No goods")
	if currentTab.ID == shopSellTabID {
		emptyText = query.T(world, "No items to sell")
	}
	return renderMenuListUI(itemIndex, rows, columnWidths, aligns, menuListOpts{AlwaysIndicator: true, EmptyText: emptyText, ItemsPerPage: perPage}, res)
}
