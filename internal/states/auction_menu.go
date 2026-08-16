package states

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// 状況確認メニューのタブ識別子。出品中・落札済み・履歴を読み取り専用で見る
const (
	tabIDListing tabID = "listing"
	tabIDSold    tabID = "sold"
	tabIDHistory tabID = "history"
)

// AuctionMenuState は出荷場所の状況確認メニュー。出品中の現在値、落札済みの手取り、
// 発送済みの履歴を読み取り専用で確認する。出品はアイテムの出品動詞、出荷は出荷場所の
// 相互作用が担い、このメニューからは金銭のやり取りをしない。
type AuctionMenuState struct {
	es.BaseState[w.World]
	detail overlay.Detail
	screen *menuloop.Screen[AuctionProps]
}

var _ es.State[w.World] = &AuctionMenuState{}
var _ menuloop.ExtraInput = &AuctionMenuState{}

// OnStart はステート開始時に画面を組む
func (st *AuctionMenuState) OnStart(_ w.World) error {
	st.detail = overlay.NewDetail(st.detailContent)
	st.screen = menuloop.NewScreen[AuctionProps](st, &st.detail)
	return nil
}

// Update は毎フレームの更新処理を行う
func (st *AuctionMenuState) Update(world w.World) (es.Transition[w.World], error) {
	return st.screen.Update(world)
}

// Draw は画面を描画する
func (st *AuctionMenuState) Draw(_ w.World, screen *ebiten.Image) error {
	st.screen.Draw(screen)
	return nil
}

// ExtraInput は x で選択中の詳細モーダルを開く
func (st *AuctionMenuState) ExtraInput() (inputmapper.ActionID, bool) {
	ki := input.GetSharedKeyboardInput()
	if ki.IsKeyJustPressed(ebiten.KeyX) && !ki.IsKeyPressed(ebiten.KeyShift) {
		return inputmapper.ActionOpenItemDetail, true
	}
	return "", false
}

// DoAction はActionを実行する。状況確認専用なので決定は詳細を開くだけで、状態は変えない
func (st *AuctionMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionMenuSelect, inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
	case inputmapper.ActionMenuUp, inputmapper.ActionMenuDown, inputmapper.ActionMenuLeft, inputmapper.ActionMenuRight, inputmapper.ActionMenuTabNext, inputmapper.ActionMenuTabPrev:
		// Dispatchで処理される
	default:
		return es.Transition[w.World]{}, fmt.Errorf("auctionMenu: unsupported action: %s", action)
	}
	return es.Transition[w.World]{Type: es.TransNone}, nil
}

// AuctionProps は画面の表示 props
type AuctionProps struct {
	Tabs []auctionTabData
}

type auctionTabData struct {
	ID    tabID
	Label string
	Rows  []auctionLedgerRow
}

// auctionLedgerRow は状況の1行。出品中は現在値を、落札済みと履歴は落札額と手取りを持つ。
type auctionLedgerRow struct {
	Number  int
	Name    string
	Bid     int  // 出品中は現在値、落札済みと履歴は落札額
	Ship    int  // 発送料
	Fee     int  // 手数料
	Net     int  // 手取り
	Turn    int  // 発送したターン。履歴のみ意味を持つ
	Ongoing bool // 出品中で入札継続中か
	Shipped bool // 発送済みで履歴に載ったか
}

// Fetch は世界から表示 props を構築する
func (st *AuctionMenuState) Fetch(world w.World) AuctionProps {
	return AuctionProps{
		Tabs: []auctionTabData{
			{ID: tabIDListing, Label: query.T(world, "Listing"), Rows: st.listingRows(world)},
			{ID: tabIDSold, Label: query.T(world, "Won"), Rows: st.soldRows(world)},
			{ID: tabIDHistory, Label: query.T(world, "History"), Rows: st.historyRows(world)},
		},
	}
}

// Menu は一覧の構成を返す
func (st *AuctionMenuState) Menu(props AuctionProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		itemCounts[i] = len(tab.Rows)
	}
	return menuloop.MenuConfig{Key: "auction", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuItemsPerPage}
}

// listingRows は出品中の品を番号順に返す。各行は現在値と、その値で確定したときの内訳を持つ
func (st *AuctionMenuState) listingRows(world w.World) []auctionLedgerRow {
	var rows []auctionLedgerRow
	q := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	for q.Next() {
		item := q.Entity()
		l := world.Components.AuctionListing.Get(item)
		ship := query.AuctionShippingCost(world, item)
		fee := query.AuctionFee(l.CurrentBid)
		rows = append(rows, auctionLedgerRow{
			Number: l.Number, Name: query.GetEntityName(item, world),
			Bid: l.CurrentBid, Ship: ship, Fee: fee, Net: l.CurrentBid - ship - fee, Ongoing: true,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Number < rows[j].Number })
	return rows
}

// soldRows は落札済みで未出荷の品を番号順に返す。各行は落札額と出荷したときの手取りを持つ
func (st *AuctionMenuState) soldRows(world w.World) []auctionLedgerRow {
	var rows []auctionLedgerRow
	q := ecs.NewFilter1[gc.AuctionSold](world.ECS).Query()
	for q.Next() {
		item := q.Entity()
		s := world.Components.AuctionSold.Get(item)
		ship := query.AuctionShippingCost(world, item)
		fee := query.AuctionFee(s.Bid)
		rows = append(rows, auctionLedgerRow{
			Number: s.Number, Name: query.GetEntityName(item, world),
			Bid: s.Bid, Ship: ship, Fee: fee, Net: s.Bid - ship - fee,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Number < rows[j].Number })
	return rows
}

// historyRows は発送済みの実績を新しい順に返す
func (st *AuctionMenuState) historyRows(world w.World) []auctionLedgerRow {
	h := query.GetAuctionHistory(world)
	rows := make([]auctionLedgerRow, 0, len(h.Records))
	for i := len(h.Records) - 1; i >= 0; i-- {
		r := h.Records[i]
		rows = append(rows, auctionLedgerRow{
			Number: r.Number, Name: r.Name, Bid: r.Bid, Ship: r.Ship, Fee: r.Fee, Net: r.Net, Turn: r.Turn, Shipped: true,
		})
	}
	return rows
}

// View は props を UI へ組む
func (st *AuctionMenuState) View(world w.World, props AuctionProps, cursor menuloop.Selection, res resources.UIResources) *ebitenui.UI {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	return menuframe.NewTabScreen(res, menuframe.TabScreen{
		TabLabels: labels,
		TabIndex:  cursor.TabIndex,
		Content:   st.buildActiveContainer(world, props, cursor.TabIndex, cursor.ItemIndex, res),
		Footer:    menuNavHint(world, true, query.T(world, "x Details")),
	})
}

// detailContent は x で開く詳細の内容を返す。選択中の行の内訳を価格記号付きで見せる
func (st *AuctionMenuState) detailContent(world w.World) (overlay.DetailContent, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return overlay.DetailContent{}, false
	}
	tab := props.Tabs[cursor.TabIndex]
	if cursor.ItemIndex < 0 || cursor.ItemIndex >= len(tab.Rows) {
		return overlay.DetailContent{}, false
	}
	return auctionLedgerDetail(world, tab.Rows[cursor.ItemIndex]), true
}

// auctionLedgerDetail は台帳1行の内訳を詳細内容にする。価格は価格記号を付けて出す。
// 出品中は現在値、それ以外は落札額を見せる。発送済みは確定ターンを添える。
func auctionLedgerDetail(world w.World, r auctionLedgerRow) overlay.DetailContent {
	bidLabel := query.T(world, "Bid")
	if r.Ongoing {
		bidLabel = query.T(world, "Current bid")
	}
	rows := []entityspec.SpecRow{
		{Label: query.T(world, "Number"), Value: "#" + strconv.Itoa(r.Number)},
		{Label: bidLabel, Value: query.FormatCurrency(r.Bid)},
		{Label: query.T(world, "Shipping"), Value: query.FormatCurrency(r.Ship)},
		{Label: query.T(world, "Fee"), Value: query.FormatCurrency(r.Fee)},
		{Label: query.T(world, "Net"), Value: query.FormatCurrency(r.Net)},
	}
	if r.Shipped {
		rows = append(rows, entityspec.SpecRow{Label: query.T(world, "Turn"), Value: strconv.Itoa(r.Turn)})
	}
	return overlay.DetailContent{Name: r.Name, Rows: rows}
}

func (st *AuctionMenuState) buildActiveContainer(world w.World, props AuctionProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	if tabIndex >= len(props.Tabs) {
		return styled.NewVerticalContainer()
	}
	tab := props.Tabs[tabIndex]
	// 各行は番号・品名・値。値は出品中なら現在値、それ以外は手取り。内訳は x の詳細で見る
	rows := make([]menuRow, len(tab.Rows))
	for i, r := range tab.Rows {
		value := r.Net
		if r.Ongoing {
			value = r.Bid
		}
		rows[i] = menuRow{Cells: []styled.Cell{
			styled.TextCell("#" + strconv.Itoa(r.Number)),
			styled.TextCell(r.Name),
			styled.TextCell(query.FormatCurrency(value)),
		}}
	}
	return renderMenuList(itemIndex, rows, []int{50, 200, 120},
		[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight},
		menuListOpts{AlwaysIndicator: true, EmptyText: auctionEmptyText(world, tab.ID)}, res)
}

// auctionEmptyText はタブごとの空表示を返す
func auctionEmptyText(world w.World, id tabID) string {
	switch id {
	case tabIDListing:
		return query.T(world, "Nothing listed.")
	case tabIDSold:
		return query.T(world, "No won items.")
	case tabIDHistory:
		return query.T(world, "No shipments yet.")
	case tabIDStore, tabIDRetrieve:
	}
	return query.T(world, "No items")
}
