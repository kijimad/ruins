package states

import (
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/keybind"
	"github.com/kijimaD/ruins/internal/menuloop"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/kijimaD/ruins/internal/ui"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// auctionTabID は出荷場所メニューのタブ識別子。収納メニューの tabID とは別型にして、
// 各メニューの switch が自分のタブだけを網羅すればよいようにする。
type auctionTabID string

// 出荷場所メニューのタブ。進行中・出荷・集荷待ち・金銭・履歴で構成する
const (
	auctionTabStatus  auctionTabID = "status"
	auctionTabShip    auctionTabID = "ship"
	auctionTabPending auctionTabID = "pending"
	auctionTabFinance auctionTabID = "finance"
	auctionTabHistory auctionTabID = "history"
)

// AuctionMenuState は出荷場所のメニュー。金銭・積む・積荷・出品中・履歴のタブを持つ。
// 積むタブで落札済みの品を積荷へ入れると、集荷はターン経過で自動に行われる。
// 金銭タブで受取金と請求の明細を精算して所持金が動く。他のタブは読み取り専用の状況確認。
type AuctionMenuState struct {
	es.BaseState[w.World]
	stationEntity ecs.Entity // 積荷の収納を持つ出荷場所。荷物はステーションごと
	detail        overlay.Detail
	screen        *menuloop.Screen[AuctionProps]
}

var _ es.State[w.World] = &AuctionMenuState{}
var _ menuloop.KeyBindings = &AuctionMenuState{}

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

// KeyBindings は x の詳細表示を共通入力に足す
func (st *AuctionMenuState) KeyBindings() []keybind.Binding {
	return detailOpenBindings
}

// DoAction はActionを実行する。Enter は積む・出す、x は詳細を開く。出荷はターン経過で自動実行する
func (st *AuctionMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
	case inputmapper.ActionMenuSelect:
		if err := st.selectRow(world); err != nil {
			return es.Transition[w.World]{}, err
		}
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
	ID      auctionTabID
	Label   string
	Items   []auctionItemRow   // 積む・出荷タブの品
	Ledger  []auctionLedgerRow // 出品中・履歴タブの台帳
	Entries []gc.AuctionEntry  // 金銭タブの明細
}

// auctionItemRow は積む・出荷タブの1行。実体と、出品状況を表す表示を持つ
type auctionItemRow struct {
	Entity   ecs.Entity
	Name     string
	Status   string          // 落札済みか出品中
	Value    consts.Currency // 入札額。落札済みは落札額、出品中は現在値
	HasValue bool            // 金額を出すか
}

// auctionLedgerRow は進行中・履歴タブの1行
type auctionLedgerRow struct {
	Number  int
	Name    string
	Bid     consts.Currency
	Ship    consts.Currency
	Fee     consts.Currency
	Net     consts.Currency
	Turn    int
	Ongoing bool
	Shipped bool
}

// Fetch は世界から表示 props を構築する
func (st *AuctionMenuState) Fetch(world w.World) (AuctionProps, error) {
	return AuctionProps{
		Tabs: []auctionTabData{
			{ID: auctionTabStatus, Label: query.T(world, "In progress"), Ledger: st.statusRows(world)},
			{ID: auctionTabShip, Label: query.T(world, "Ship"), Items: st.stageItems(world)},
			{ID: auctionTabPending, Label: query.T(world, "Pending"), Items: st.shipItems(world)},
			{ID: auctionTabFinance, Label: query.T(world, "Finance"), Entries: query.GetAuctionHistory(world).Entries},
			{ID: auctionTabHistory, Label: query.T(world, "History"), Ledger: st.historyRows(world)},
		},
	}, nil
}

// Menu は一覧の構成を返す
func (st *AuctionMenuState) Menu(props AuctionProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		switch tab.ID {
		case auctionTabShip, auctionTabPending:
			itemCounts[i] = len(tab.Items)
		case auctionTabFinance:
			itemCounts[i] = len(tab.Entries)
		case auctionTabStatus, auctionTabHistory:
			itemCounts[i] = len(tab.Ledger)
		}
	}
	return menuloop.MenuConfig{Key: "auction", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuloop.ItemsPerPageAuto}
}

// stageItems は落札済みで持ち物にある品を積むタブへ並べる。積荷へ入れるとステーションの収納へ移る。
// 出荷できるのは落札を勝ち取った品に限る
func (st *AuctionMenuState) stageItems(world w.World) []auctionItemRow {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil
	}
	var candidates []ecs.Entity
	for _, e := range playerBackpackItems(world, player) {
		if world.Components.AuctionSold.Has(e) {
			candidates = append(candidates, e)
		}
	}
	return st.toItemRows(world, candidates)
}

// shipItems はこのステーションの積荷を出荷タブへ並べる。次の集荷で出荷される
func (st *AuctionMenuState) shipItems(world w.World) []auctionItemRow {
	items := query.GetStorageItems(world, st.stationEntity)
	return st.toItemRows(world, query.SortEntities(world, items))
}

func (st *AuctionMenuState) toItemRows(world w.World, entities []ecs.Entity) []auctionItemRow {
	rows := make([]auctionItemRow, len(entities))
	for i, e := range entities {
		row := auctionItemRow{Entity: e, Name: query.GetEntityName(e, world)}
		// 積む・出荷タブにはタグを貼った品しか来ないので、出品中か落札済みのいずれかになる。
		// 金額はどの状態でも入札額を出す。出品中は現在値、落札済みは落札額
		switch {
		case world.Components.AuctionSold.Has(e):
			row.Status = query.T(world, "Won")
			row.Value = world.Components.AuctionSold.Get(e).Bid
			row.HasValue = true
		case world.Components.AuctionListing.Has(e):
			row.Status = query.T(world, "Bidding")
			row.Value = world.Components.AuctionListing.Get(e).CurrentBid
			row.HasValue = true
		}
		rows[i] = row
	}
	return rows
}

// statusRows は進行中の取引を番号順に返す。出品中の品と、落札済みでまだ出荷していない品を並べる。
// 出荷が済むまで取引は完了しないので、どちらも一覧に残す
func (st *AuctionMenuState) statusRows(world w.World) []auctionLedgerRow {
	var rows []auctionLedgerRow
	lq := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	for lq.Next() {
		item := lq.Entity()
		l := world.Components.AuctionListing.Get(item)
		ship := query.AuctionShippingCost(world, item)
		fee := query.AuctionFee(l.CurrentBid)
		rows = append(rows, auctionLedgerRow{
			Number: l.Number, Name: query.GetEntityName(item, world),
			Bid: l.CurrentBid, Ship: ship, Fee: fee, Net: l.CurrentBid - ship - fee, Ongoing: true,
		})
	}
	sq := ecs.NewFilter1[gc.AuctionSold](world.ECS).Query()
	for sq.Next() {
		item := sq.Entity()
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
	for _, r := range slices.Backward(h.Records) {
		rows = append(rows, auctionLedgerRow{
			Number: r.Number, Name: r.Name, Bid: r.Bid, Ship: r.Ship, Fee: r.Fee, Net: r.Net, Turn: r.Turn, Shipped: true,
		})
	}
	return rows
}

// selectRow は Enter の対象を処理する。積むタブは積荷へ入れ、出荷タブは持ち物へ戻し、
// 金銭タブは明細を精算する。出品中と履歴タブは詳細を開くだけで状態は変えない
func (st *AuctionMenuState) selectRow(world w.World) error {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[cursor.TabIndex]
	switch tab.ID {
	case auctionTabShip:
		// 落札済みの品をステーションの積荷へ入れる。次の集荷で出荷される
		item, ok := st.selectedItem(tab, cursor.ItemIndex)
		if !ok {
			return nil
		}
		if !query.CanAddToStorage(world, st.stationEntity, item) {
			return nil
		}
		return lifecycle.MoveToStorage(world, item, st.stationEntity)
	case auctionTabPending:
		// 積荷から降ろして持ち物へ戻す。集荷前なら取り消せる
		item, ok := st.selectedItem(tab, cursor.ItemIndex)
		if !ok {
			return nil
		}
		player, err := query.GetPlayerEntity(world)
		if err != nil {
			return err
		}
		return lifecycle.MoveToBackpack(world, item, player)
	case auctionTabFinance:
		return st.settleEntry(world, cursor.ItemIndex)
	case auctionTabStatus, auctionTabHistory:
		st.detail.Open(world)
	}
	return nil
}

// settleEntry は金銭タブの選択中の明細を精算する。受取金は所持金へ加え、請求は引く。
// 精算した明細をログに出す。集金の一撃がここで起こる
func (st *AuctionMenuState) settleEntry(world w.World, index int) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	now := int(query.GetGameTime(world).TotalTurns)
	e, ok := query.SettleAuctionEntry(world, player, index, now)
	if !ok {
		return nil
	}
	switch e.Kind {
	case gc.AuctionEntryReceipt:
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Collected %s. Received %s.", gamelog.Tag("item", e.Name), e.Amount.String())).
			Log()
	case gc.AuctionEntryInvoice:
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "Paid %s: %s.", query.T(world, e.Name), e.Amount.String())).
			Log()
	}
	return nil
}

func (st *AuctionMenuState) selectedItem(tab auctionTabData, index int) (ecs.Entity, bool) {
	if index < 0 || index >= len(tab.Items) {
		return gc.InvalidEntity, false
	}
	return tab.Items[index].Entity, true
}

// ViewUI は View の internal/ui 版。オークションの各タブを自前 UI で組む。
func (st *AuctionMenuState) ViewUI(world w.World, props AuctionProps, cursor menuloop.Selection, res resources.UIResources) ui.Widget {
	labels := make([]string, len(props.Tabs))
	for i, tab := range props.Tabs {
		labels[i] = tab.Label
	}
	content := st.buildActiveUI(world, props, cursor.TabIndex, cursor.ItemIndex, cursor.PageSize, res)
	return buildTabScreenUI(world, res, "", labels, cursor.TabIndex, content, keybind.HelpHint(world))
}

// buildActiveUI は buildActiveContainer の internal/ui 版。タブ種別で中身を振り分ける。
func (st *AuctionMenuState) buildActiveUI(world w.World, props AuctionProps, tabIndex, itemIndex, perPage int, res resources.UIResources) []ui.Widget {
	if tabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[tabIndex]
	switch tab.ID {
	case auctionTabShip, auctionTabPending:
		rows := make([]menuRow, len(tab.Items))
		for i, it := range tab.Items {
			value := ""
			if it.HasValue {
				value = it.Value.String()
			}
			rows[i] = menuRow{Cells: []styled.Cell{styled.TextCell(it.Name), styled.TextCell(it.Status), styled.TextCell(value)}}
		}
		empty := query.T(world, "No items to ship.")
		if tab.ID == auctionTabPending {
			empty = query.T(world, "Nothing awaiting pickup.")
		}
		return renderMenuListUI(itemIndex, rows, []int{200, 90, 110},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight},
			menuListOpts{AlwaysIndicator: true, EmptyText: empty, ItemsPerPage: perPage}, res)
	case auctionTabFinance:
		rows := make([]menuRow, len(tab.Entries))
		for i, e := range tab.Entries {
			kind := query.T(world, "Receipt")
			name := e.Name
			amount := e.Amount.String()
			if e.Kind == gc.AuctionEntryInvoice {
				kind = query.T(world, "Invoice")
				name = query.T(world, e.Name)
				amount = (-e.Amount).String()
			}
			rows[i] = menuRow{Cells: []styled.Cell{styled.TextCell(name), styled.TextCell(kind), styled.TextCell(amount)}}
		}
		return renderMenuListUI(itemIndex, rows, []int{200, 80, 120},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight},
			menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No bills or receipts."), ItemsPerPage: perPage}, res)
	case auctionTabHistory:
		rows := make([]menuRow, len(tab.Ledger))
		for i, r := range tab.Ledger {
			rows[i] = menuRow{Cells: []styled.Cell{styled.TextCell("#" + strconv.Itoa(r.Number)), styled.TextCell(r.Name), styled.TextCell(r.Bid.String())}}
		}
		return renderMenuListUI(itemIndex, rows, []int{50, 200, 120},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignRight},
			menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No shipments yet."), ItemsPerPage: perPage}, res)
	default:
		rows := make([]menuRow, len(tab.Ledger))
		for i, r := range tab.Ledger {
			status := query.T(world, "Won")
			if r.Ongoing {
				status = query.T(world, "Bidding")
			}
			rows[i] = menuRow{Cells: []styled.Cell{styled.TextCell("#" + strconv.Itoa(r.Number)), styled.TextCell(r.Name), styled.TextCell(status), styled.TextCell(r.Bid.String())}}
		}
		return renderMenuListUI(itemIndex, rows, []int{50, 170, 70, 110},
			[]styled.TextAlign{styled.AlignLeft, styled.AlignLeft, styled.AlignLeft, styled.AlignRight},
			menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No deals in progress."), ItemsPerPage: perPage}, res)
	}
}

// detailContent は x で開く詳細の内容を返す。積む・出荷タブは品の詳細、状況タブは台帳の内訳
func (st *AuctionMenuState) detailContent(world w.World) (overlay.DetailContent, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return overlay.DetailContent{}, false
	}
	tab := props.Tabs[cursor.TabIndex]
	if tab.ID == auctionTabShip || tab.ID == auctionTabPending {
		item, ok := st.selectedItem(tab, cursor.ItemIndex)
		if !ok || !world.ECS.Alive(item) {
			return overlay.DetailContent{}, false
		}
		return overlay.EntityDetailContent(world, item), true
	}
	if tab.ID == auctionTabFinance {
		if cursor.ItemIndex < 0 || cursor.ItemIndex >= len(tab.Entries) {
			return overlay.DetailContent{}, false
		}
		return auctionEntryDetail(world, tab.Entries[cursor.ItemIndex]), true
	}
	if cursor.ItemIndex < 0 || cursor.ItemIndex >= len(tab.Ledger) {
		return overlay.DetailContent{}, false
	}
	return auctionLedgerDetail(world, tab.Ledger[cursor.ItemIndex]), true
}

// auctionEntryDetail は金銭明細1件の内訳を詳細内容にする。受取金は落札額から配送料と手数料を引いた
// 手取りの内訳を、請求は請求額を見せる
func auctionEntryDetail(world w.World, e gc.AuctionEntry) overlay.DetailContent {
	switch e.Kind {
	case gc.AuctionEntryReceipt:
		return overlay.DetailContent{
			Name: e.Name,
			Rows: []entityspec.SpecRow{
				{Label: query.T(world, "Kind"), Value: query.T(world, "Receipt")},
				{Label: query.T(world, "Bid"), Value: e.Bid.String()},
				{Label: query.T(world, "Shipping"), Value: e.Ship.String()},
				{Label: query.T(world, "Fee"), Value: e.Fee.String()},
				{Label: query.T(world, "Net"), Value: e.Amount.String()},
			},
		}
	case gc.AuctionEntryInvoice:
		return overlay.DetailContent{
			Name: query.T(world, e.Name),
			Rows: []entityspec.SpecRow{
				{Label: query.T(world, "Kind"), Value: query.T(world, "Invoice")},
				{Label: query.T(world, "Amount"), Value: e.Amount.String()},
			},
		}
	}
	return overlay.DetailContent{Name: e.Name}
}

// auctionLedgerDetail は台帳1行の内訳を詳細内容にする。価格は価格記号を付けて出す
func auctionLedgerDetail(world w.World, r auctionLedgerRow) overlay.DetailContent {
	bidLabel := query.T(world, "Bid")
	if r.Ongoing {
		bidLabel = query.T(world, "Current bid")
	}
	rows := []entityspec.SpecRow{
		{Label: query.T(world, "Number"), Value: "#" + strconv.Itoa(r.Number)},
		{Label: bidLabel, Value: r.Bid.String()},
		{Label: query.T(world, "Shipping"), Value: r.Ship.String()},
		{Label: query.T(world, "Fee"), Value: r.Fee.String()},
		{Label: query.T(world, "Net"), Value: r.Net.String()},
	}
	if r.Shipped {
		rows = append(rows, entityspec.SpecRow{Label: query.T(world, "Turn"), Value: strconv.Itoa(r.Turn)})
	}
	return overlay.DetailContent{Name: r.Name, Rows: rows}
}
