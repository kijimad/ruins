package states

import (
	"fmt"
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
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/entityspec"
	"github.com/kijimaD/ruins/internal/widgets/menuframe"
	"github.com/kijimaD/ruins/internal/widgets/overlay"
	"github.com/kijimaD/ruins/internal/widgets/styled"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// tabIDHistory は出荷実績を閲覧する履歴タブの識別子
const tabIDHistory tabID = "history"

// AuctionMenuState はオークション箱専用のメニュー。出品(Store)・取り出し(Retrieve)・
// 出荷履歴(History)の3タブを持つ。品を箱へ収納すると競売が始まる。競売はシステムが進める。
type AuctionMenuState struct {
	es.BaseState[w.World]
	boxEntity ecs.Entity
	detail    overlay.Detail
	screen    *menuloop.Screen[AuctionProps]
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
	// 収納の出し入れで所持品が変わると WeightDirty が立つ。再計算を回して総重量表示を更新する
	if err := runUpdaters(world, &gs.WeightDirtySystem{}); err != nil {
		return es.Transition[w.World]{}, err
	}
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

// DoAction はActionを実行する
func (st *AuctionMenuState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	switch action {
	case inputmapper.ActionMenuCancel, inputmapper.ActionCloseMenu:
		return es.Transition[w.World]{Type: es.TransPop}, nil
	case inputmapper.ActionOpenItemDetail:
		st.detail.Open(world)
	case inputmapper.ActionMenuSelect:
		if err := st.executeTransfer(world); err != nil {
			return es.Transition[w.World]{}, err
		}
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
	ID      tabID
	Label   string
	Items   []itemRowData      // 出品・取り出しタブの品
	Records []gc.AuctionRecord // 履歴タブの出荷実績。新しい順
}

// Fetch は世界から表示 props を構築する
func (st *AuctionMenuState) Fetch(world w.World) AuctionProps {
	return AuctionProps{
		Tabs: []auctionTabData{
			{ID: tabIDStore, Label: query.T(world, "Store"), Items: st.backpackItems(world)},
			{ID: tabIDRetrieve, Label: query.T(world, "Retrieve"), Items: st.boxItems(world)},
			{ID: tabIDHistory, Label: query.T(world, "History"), Records: st.historyRecords(world)},
		},
	}
}

// Menu は一覧の構成を返す
func (st *AuctionMenuState) Menu(props AuctionProps) menuloop.MenuConfig {
	itemCounts := make([]int, len(props.Tabs))
	for i, tab := range props.Tabs {
		if tab.ID == tabIDHistory {
			itemCounts[i] = len(tab.Records)
		} else {
			itemCounts[i] = len(tab.Items)
		}
	}
	return menuloop.MenuConfig{Key: "auction", TabCount: len(props.Tabs), ItemCounts: itemCounts, ItemsPerPage: menuItemsPerPage}
}

func (st *AuctionMenuState) boxItems(world w.World) []itemRowData {
	items := query.GetStorageItems(world, st.boxEntity)
	return toAuctionItemData(world, query.SortEntities(world, items))
}

func (st *AuctionMenuState) backpackItems(world w.World) []itemRowData {
	var entities []ecs.Entity
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		entities = append(entities, q.Entity())
	}
	return toAuctionItemData(world, query.SortEntities(world, entities))
}

func toAuctionItemData(world w.World, entities []ecs.Entity) []itemRowData {
	items := make([]itemRowData, len(entities))
	for i, e := range entities {
		item := itemRowData{Entity: e, Name: query.GetEntityName(e, world), Weight: query.GetEntityWeight(world, e).KgString()}
		if world.Components.Stackable.Has(e) {
			item.Count = world.Components.Stackable.Get(e).Count
		}
		items[i] = item
	}
	return items
}

// historyRecords は出荷実績を新しい順に返す。
func (st *AuctionMenuState) historyRecords(world w.World) []gc.AuctionRecord {
	h := query.GetAuctionHistory(world)
	records := make([]gc.AuctionRecord, 0, len(h.Records))
	for i := len(h.Records) - 1; i >= 0; i-- {
		records = append(records, h.Records[i])
	}
	return records
}

// executeTransfer は選択中のタブに応じて品を箱へ入れる、または取り出す。履歴タブは閲覧のみ。
func (st *AuctionMenuState) executeTransfer(world w.World) error {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return nil
	}
	tab := props.Tabs[cursor.TabIndex]
	switch tab.ID {
	case tabIDStore:
		item, ok := tabItemAt(tab, cursor.ItemIndex)
		if !ok {
			return nil
		}
		if !query.CanAddToStorage(world, st.boxEntity, item.Entity) {
			return nil
		}
		return lifecycle.MoveToStorage(world, item.Entity, st.boxEntity)
	case tabIDRetrieve:
		item, ok := tabItemAt(tab, cursor.ItemIndex)
		if !ok {
			return nil
		}
		player, err := query.GetPlayerEntity(world)
		if err != nil {
			return err
		}
		return lifecycle.MoveToBackpack(world, item.Entity, player)
	case tabIDHistory:
		// 履歴は閲覧のみ。選択しても何もしない
	}
	return nil
}

func tabItemAt(tab auctionTabData, index int) (itemRowData, bool) {
	if index < 0 || index >= len(tab.Items) {
		return itemRowData{}, false
	}
	return tab.Items[index], true
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

// detailContent は x で開く詳細の内容を返す。出品・取り出しタブは品の詳細、履歴タブは出荷実績の内訳。
func (st *AuctionMenuState) detailContent(world w.World) (overlay.DetailContent, bool) {
	props := st.screen.Props()
	cursor := st.screen.Selection()
	if cursor.TabIndex >= len(props.Tabs) {
		return overlay.DetailContent{}, false
	}
	tab := props.Tabs[cursor.TabIndex]
	if tab.ID == tabIDHistory {
		if cursor.ItemIndex < 0 || cursor.ItemIndex >= len(tab.Records) {
			return overlay.DetailContent{}, false
		}
		return auctionRecordDetail(world, tab.Records[cursor.ItemIndex]), true
	}
	item, ok := tabItemAt(tab, cursor.ItemIndex)
	if !ok || !world.ECS.Alive(item.Entity) {
		return overlay.DetailContent{}, false
	}
	return overlay.EntityDetailContent(world, item.Entity), true
}

// auctionRecordDetail は出荷実績1件の内訳を詳細内容にする。
func auctionRecordDetail(world w.World, r gc.AuctionRecord) overlay.DetailContent {
	return overlay.DetailContent{
		Name: r.Name,
		Rows: []entityspec.SpecRow{
			{Label: query.T(world, "Bid"), Value: strconv.Itoa(r.Bid)},
			{Label: query.T(world, "Shipping"), Value: strconv.Itoa(r.Ship)},
			{Label: query.T(world, "Fee"), Value: strconv.Itoa(r.Fee)},
			{Label: query.T(world, "Net"), Value: strconv.Itoa(r.Net)},
			{Label: query.T(world, "Turn"), Value: strconv.Itoa(r.Turn)},
		},
	}
}

func (st *AuctionMenuState) buildActiveContainer(world w.World, props AuctionProps, tabIndex, itemIndex int, res resources.UIResources) *widget.Container {
	if tabIndex >= len(props.Tabs) {
		return styled.NewVerticalContainer()
	}
	tab := props.Tabs[tabIndex]
	if tab.ID == tabIDHistory {
		// 各行は品名と落札額だけ。内訳は x の詳細で見る
		rows := make([]menuRow, len(tab.Records))
		for i, r := range tab.Records {
			rows[i] = menuRow{Cells: []styled.Cell{styled.TextCell(r.Name), styled.TextCell(strconv.Itoa(r.Bid))}}
		}
		return renderMenuList(itemIndex, rows, []int{280, 100}, []styled.TextAlign{styled.AlignLeft, styled.AlignRight},
			menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No shipments yet.")}, res)
	}
	columnWidths, aligns := itemMenuColumns(260, menuColumn{Width: 80, Align: styled.AlignRight})
	rows := make([]menuRow, len(tab.Items))
	for i, it := range tab.Items {
		rows[i] = itemMenuRow(world, it.Entity, it.Weight)
	}
	return renderMenuList(itemIndex, rows, columnWidths, aligns,
		menuListOpts{AlwaysIndicator: true, EmptyText: query.T(world, "No items")}, res)
}
